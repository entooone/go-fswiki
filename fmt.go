package fswiki

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/mattn/go-runewidth"
)

func FormatDocument(r io.Reader) ([]byte, error) {
	nodes, err := Parse(r)
	if err != nil {
		return nil, err
	}

	buf := &bytes.Buffer{}

	listHeader := ""
	listDepth := 0

	var (
		table           [][]string
		colwidth        []int
		commentsInTable []string
		ti, tj          int
		isTable         bool
	)

	for _, n := range nodes {
		switch n.Kind {
		case NodeHeadingOpen:
			fmt.Fprintf(buf, "%s ", strings.Repeat("!", 5-n.Level))
		case NodeHeadingClose:
			fmt.Fprintf(buf, "\n\n")
		case NodeUnorderedListOpen:
			listHeader = "*"
			listDepth++
		case NodeOrderedListOpen:
			listHeader = "+"
			listDepth++
		case NodeUnorderedListClose, NodeOrderedListClose:
			listDepth--
			if listDepth == 0 {
				fmt.Fprintf(buf, "\n")
			}
		case NodeListItemOpen:
			fmt.Fprintf(buf, "%s ", strings.Repeat(listHeader, listDepth))
		case NodeListItemClose:
			fmt.Fprintf(buf, "\n")
		case NodeParagraphOpen:
			fmt.Fprintf(buf, "")
		case NodeParagraphClose:
			fmt.Fprintf(buf, "\n\n")
		case NodePreformatted:
			bs := bufio.NewScanner(strings.NewReader(n.Content))
			for bs.Scan() {
				fmt.Fprintf(buf, " %s\n", bs.Text())
			}
			fmt.Fprintf(buf, "\n")
		case NodeTableOpen:
			table = make([][]string, 0)
			colwidth = make([]int, 1)
			commentsInTable = make([]string, 0)
			ti, tj = 0, 0
			isTable = true
		case NodeTableClose:
			for i, row := range table {
				for j, s := range row {
					fmt.Fprintf(buf, ",%s", runewidth.FillLeft(s, colwidth[j]))
				}
				fmt.Fprintf(buf, "\n")
				fmt.Fprintf(buf, commentsInTable[i])
			}
			fmt.Fprintf(buf, "\n")
			isTable = false
		case NodeTableRowOpen:
			tj = 0
			table = append(table, make([]string, len(colwidth)))
			commentsInTable = append(commentsInTable, "")
		case NodeComment:
			if isTable {
				commentsInTable[ti-1] += fmt.Sprintf("//%s\n", n.Content)
			} else {
				fmt.Fprintf(buf, "//%s\n", n.Content)
			}
		case NodeTableRowClose:
			ti++
		case NodeTableHeaderOpen, NodeTableDataOpen:
			if len(colwidth) <= tj {
				for i := range table {
					table[i] = append(table[i], "")
				}
				colwidth = append(colwidth, 0)
			}
		case NodeTableHeaderClose, NodeTableDataClose:
			tj++
		case NodePlugin:
			if n.Content == "" {
				fmt.Fprintf(buf, "{{%s}}", n.Tag)
			} else if strings.HasPrefix(n.Content, "\n") {
				fmt.Fprintf(buf, "{{%s%s}}", n.Tag, n.Content)
			} else {
				fmt.Fprintf(buf, "{{%s %s}}", n.Tag, n.Content)
			}
			fmt.Fprintf(buf, "\n\n")
		case NodeInline:
			tbuf := &bytes.Buffer{}
			for _, c := range n.Children {
				switch c.Kind {
				case NodeText:
					fmt.Fprintf(tbuf, "%s", strings.TrimSpace(c.Content))
				case NodeSoftBreak:
					fmt.Fprintf(tbuf, "\n")
				case NodeStrongOpen, NodeStrongClose:
					fmt.Fprintf(tbuf, "'''")
				case NodeEMOpen, NodeEMClose:
					fmt.Fprintf(tbuf, "''")
				}
			}
			if isTable {
				table[ti][tj] = tbuf.String()
				w := runewidth.StringWidth(table[ti][tj])
				if colwidth[tj] < w {
					colwidth[tj] = w
				}
			} else {
				tbuf.WriteTo(buf)
			}
		}
	}

	return buf.Bytes(), nil
}
