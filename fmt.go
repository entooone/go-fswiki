package fswiki

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/mattn/go-runewidth"
)

type TableAlignType int

const (
	TableAlignLeft TableAlignType = iota
	TableAlignRight
)

type FormatOption struct {
	// TableAlign はテーブルの整形における文字列の配置を指定します。
	TableAlign TableAlignType
	// TableInsertSpaceToEndOfCell はテーブルの整形におけるセルの末尾にスペースを挿入するかどうかを指定します。
	TableInsertSpaceToEndOfCell bool
}

func FormatDocumentWithOption(r io.Reader, option FormatOption) ([]byte, error) {
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

	for i, n := range nodes {
		switch n.Kind {
		case NodeHeadingOpen:
			fmt.Fprintf(buf, "%s ", strings.Repeat("!", 5-n.Level))
		case NodeHeadingClose:
			fmt.Fprintf(buf, "\n")
		case NodeUnorderedListOpen:
			listHeader = "*"
			listDepth++
		case NodeOrderedListOpen:
			listHeader = "+"
			listDepth++
		case NodeUnorderedListClose, NodeOrderedListClose:
			listDepth--
		case NodeListItemOpen:
			fmt.Fprintf(buf, "%s ", strings.Repeat(listHeader, listDepth))
		case NodeListItemClose:
			fmt.Fprintf(buf, "\n")
		case NodeParagraphOpen:
		case NodeParagraphClose:
			fmt.Fprintf(buf, "\n")
		case NodePreformatted:
			bs := bufio.NewScanner(strings.NewReader(n.Content))
			for bs.Scan() {
				fmt.Fprintf(buf, " %s\n", bs.Text())
			}
		case NodeTableOpen:
			table = make([][]string, 0)
			colwidth = make([]int, 1)
			commentsInTable = make([]string, 0)
			ti, tj = 0, 0
			isTable = true
		case NodeTableClose:
			for i, row := range table {
				for j, s := range row {
					cell := s

					switch option.TableAlign {
					case TableAlignLeft:
						if j < len(row)-1 {
							cell = runewidth.FillRight(cell, colwidth[j])
						}
					case TableAlignRight:
						cell = runewidth.FillLeft(cell, colwidth[j])
					}

					if option.TableInsertSpaceToEndOfCell && j < len(row)-1 {
						cell = fmt.Sprintf(",%s ", cell)
					} else {
						cell = fmt.Sprintf(",%s", cell)
					}

					fmt.Fprintf(buf, cell)
				}
				fmt.Fprintf(buf, "\n")
				fmt.Fprintf(buf, commentsInTable[i])
			}
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
			fmt.Fprintf(buf, "\n")
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
				if strings.Index(table[ti][tj], ",") != -1 {
					table[ti][tj] = fmt.Sprintf("\"%s\"", table[ti][tj])
				}
				w := runewidth.StringWidth(table[ti][tj])
				if colwidth[tj] < w {
					colwidth[tj] = w
				}
			} else {
				tbuf.WriteTo(buf)
			}
		}

		if i < len(nodes)-1 {
			switch n.Kind {
			case NodeHeadingClose, NodeParagraphClose, NodePreformatted, NodePlugin, NodeTableClose:
				fmt.Fprintf(buf, "\n")
			case NodeOrderedListClose, NodeUnorderedListClose:
				if listDepth == 0 {
					fmt.Fprintf(buf, "\n")
				}
			}
		}
	}

	return buf.Bytes(), nil
}

func FormatDocument(r io.Reader) ([]byte, error) {
	return FormatDocumentWithOption(r, FormatOption{
		TableAlign:                  TableAlignRight,
		TableInsertSpaceToEndOfCell: false,
	})
}
