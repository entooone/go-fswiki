package fswiki

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

func trimPrefixSpace(str string) string {
	return strings.TrimPrefix(str, " ")
}

//go:generate stringer -type NodeKind
type NodeKind int

const (
	NodeUnknown NodeKind = iota
	NodeHeadingOpen
	NodeHeadingClose
	NodeUnorderedListOpen
	NodeUnorderedListClose
	NodeOrderedListOpen
	NodeOrderedListClose
	NodeListItemOpen
	NodeListItemClose
	NodeParagraphOpen
	NodeParagraphClose
	NodePreformatted
	NodeTableOpen
	NodeTableClose
	NodeTableHeaderOpen
	NodeTableHeaderClose
	NodeTableRowOpen
	NodeTableRowClose
	NodeTableDataOpen
	NodeTableDataClose
	NodeStrongOpen
	NodeStrongClose
	NodeEMOpen
	NodeEMClose
	NodeInline
	NodeText
	NodeSoftBreak
	NodeComment
	NodePlugin
)

type Node struct {
	Kind     NodeKind
	Tag      string
	Attrs    map[string]string
	Content  string
	Level    int
	Markup   string
	Children []Node
}

type listElementType int

const (
	typeUnknown listElementType = iota
	typeUL
	typeOL
)

type parserState struct {
	isParagraph    bool
	isList         bool
	listDepth      int
	listType       listElementType
	isPreformatted bool
	isTable        bool
	isPlugin       bool
}

const maxListDepth = 3

type parser struct {
	curState  parserState
	prevState parserState
	etypeList [maxListDepth]listElementType
}

func (p *parser) parseStrong(str string) ([]Node, int) {
	if !strings.HasPrefix(str, "'''") {
		return nil, 0
	}

	start := 3
	for i := start; i < len(str); i++ {
		if strings.HasPrefix(str[i:], "'''") {
			children := make([]Node, 0, 2)
			children = append(children, Node{
				Kind: NodeStrongOpen,
			})
			children = append(children,
				p.parseInlineChildren(str[start:i])...)
			children = append(children, Node{
				Kind: NodeStrongClose,
			})
			return children, i + 3
		}
	}

	return nil, 0
}

func (p *parser) parseEM(str string) ([]Node, int) {
	if !strings.HasPrefix(str, "''") {
		return nil, 0
	}

	start := 2
	for i := start; i < len(str); i++ {
		if strings.HasPrefix(str[i:], "''") {
			children := make([]Node, 0, 2)
			children = append(children, Node{
				Kind: NodeEMOpen,
			})
			children = append(children,
				p.parseInlineChildren(str[start:i])...)
			children = append(children, Node{
				Kind: NodeEMClose,
			})
			return children, i + 2
		}
	}

	return nil, 0
}

func (p *parser) parseInlineChildren(str string) []Node {
	children := make([]Node, 0)
	var ptr int
	for i := 0; i < len(str); i++ {
		s := str[i:]
		var n int
		var nodes []Node

		switch {
		case strings.HasPrefix(s, "'''"):
			nodes, n = p.parseStrong(s)
		case strings.HasPrefix(s, "''"):
			nodes, n = p.parseEM(s)
		}

		if nodes != nil && n != 0 {
			children = append(children,
				Node{
					Kind:    NodeText,
					Content: str[ptr:i],
				})
			children = append(children, nodes...)
			i += n
			ptr = i
		}
	}

	if str[ptr:] != "" {
		children = append(children,
			Node{
				Kind:    NodeText,
				Content: str[ptr:],
			})
	}

	return children
}

func (p *parser) parseInline(nodes []Node, str string) ([]Node, error) {
	if nodes[len(nodes)-1].Kind != NodeInline {
		nodes = append(nodes, Node{
			Kind:     NodeInline,
			Children: make([]Node, 0),
		})
	} else {
		n := &nodes[len(nodes)-1]
		n.Children = append(n.Children, Node{
			Kind: NodeSoftBreak,
		})
	}

	n := &nodes[len(nodes)-1]
	n.Children = append(n.Children,
		p.parseInlineChildren(str)...)

	return nodes, nil
}

func (p *parser) parseMultiLineMarkup(nodes []Node) []Node {
	cur, pre := &p.curState, &p.prevState

	// paragraph
	if !pre.isParagraph && cur.isParagraph {
		nodes = append(nodes, Node{
			Kind: NodeParagraphOpen,
		})
	}
	if pre.isParagraph && !cur.isParagraph {
		nodes = append(nodes, Node{
			Kind: NodeParagraphClose,
		})
	}

	// list
	if pre.listType != cur.listType {
		var kind NodeKind
		switch pre.listType {
		case typeOL:
			kind = NodeOrderedListClose
		case typeUL:
			kind = NodeUnorderedListClose
		}
		for i := pre.listDepth - 1; i >= 0; i-- {
			nodes = append(nodes, Node{
				Kind: kind,
			})
		}
		pre.listDepth = 0
	}

	if pre.listDepth < cur.listDepth {
		var kind NodeKind
		switch cur.listType {
		case typeOL:
			kind = NodeOrderedListOpen
		case typeUL:
			kind = NodeUnorderedListOpen
		}
		for i := pre.listDepth; i < cur.listDepth; i++ {
			p.etypeList[i] = cur.listType
			nodes = append(nodes, Node{
				Kind: kind,
			})
		}
	}

	if pre.listDepth > cur.listDepth {
		for i := pre.listDepth - 1; i >= cur.listDepth; i-- {
			var kind NodeKind
			switch p.etypeList[i] {
			case typeOL:
				kind = NodeOrderedListClose
			case typeUL:
				kind = NodeUnorderedListClose
			}
			nodes = append(nodes, Node{
				Kind: kind,
			})
		}
	}

	// preformatted
	if !pre.isPreformatted && cur.isPreformatted {
		nodes = append(nodes, Node{
			Kind: NodePreformatted,
		})
	}

	// table
	if !pre.isTable && cur.isTable {
		nodes = append(nodes, Node{
			Kind: NodeTableOpen,
		})
	}
	if pre.isTable && !cur.isTable {
		nodes = append(nodes, Node{
			Kind: NodeTableClose,
		})
	}

	return nodes
}

func (p *parser) parseLineBreak(nodes []Node) []Node {
	nodes = p.parseMultiLineMarkup(nodes)
	return nodes
}

func (p *parser) parseHeading(nodes []Node, level int, str string) []Node {
	nodes = p.parseMultiLineMarkup(nodes)

	nodes = append(nodes, Node{
		Kind:  NodeHeadingOpen,
		Level: 5 - level,
	})
	nodes, _ = p.parseInline(nodes, str)
	nodes = append(nodes, Node{
		Kind:  NodeHeadingClose,
		Level: 5 - level,
	})
	return nodes
}

func (p *parser) parseList(nodes []Node, level int, etype listElementType, str string) []Node {
	p.curState.listDepth = level
	p.curState.listType = etype

	nodes = p.parseMultiLineMarkup(nodes)

	nodes = append(nodes, Node{
		Kind: NodeListItemOpen,
	})
	nodes, _ = p.parseInline(nodes, str)
	nodes = append(nodes, Node{
		Kind: NodeListItemClose,
	})
	return nodes
}

func (p *parser) parsePreformatted(nodes []Node, str string) []Node {
	p.curState.isPreformatted = true
	nodes = p.parseMultiLineMarkup(nodes)

	n := &nodes[len(nodes)-1]
	if n.Kind == NodePreformatted {
		if n.Content == "" {
			n.Content += str
		} else {
			n.Content += "\n" + str
		}
	}

	return nodes
}

func (p *parser) parseTable(nodes []Node, str string) []Node {
	p.curState.isTable = true
	nodes = p.parseMultiLineMarkup(nodes)

	var openKind, closeKind NodeKind
	if !p.prevState.isTable {
		openKind = NodeTableHeaderOpen
		closeKind = NodeTableHeaderClose
	} else {
		openKind = NodeTableDataOpen
		closeKind = NodeTableDataClose
	}

	parseTableElement := func(nodes []Node, str string) []Node {
		nodes = append(nodes, Node{
			Kind: openKind,
		})
		nodes, _ = p.parseInline(nodes, str)
		nodes = append(nodes, Node{
			Kind: closeKind,
		})
		return nodes
	}

	nodes = append(nodes, Node{
		Kind: NodeTableRowOpen,
	})
LOOP:
	for i := 0; i < len(str); i++ {
		if strings.HasPrefix(str[i:], ",\"") {
			for j := i + 2; j < len(str); j++ {
				if strings.HasPrefix(str[j:], "\"") {
					nodes = parseTableElement(nodes, str[i+2:j])
					i = j
					continue LOOP
				}
			}
		}

		if strings.HasPrefix(str[i:], ",") {
			for j := i + 1; j < len(str); j++ {
				if strings.HasPrefix(str[j:], ",") {
					nodes = parseTableElement(nodes, str[i+1:j])
					i = j - 1
					continue LOOP
				}
			}

			nodes = parseTableElement(nodes, str[i+1:])
		}
	}
	nodes = append(nodes, Node{
		Kind: NodeTableRowClose,
	})

	return nodes
}

func (p *parser) parseComment(nodes []Node, str string) []Node {
	nodes = append(nodes, Node{
		Kind:    NodeComment,
		Content: str,
	})
	p.curState = p.prevState

	return nodes
}

func (p *parser) parsePlugin(nodes []Node, str string) []Node {
	p.curState.isPlugin = true
	nodes = p.parseMultiLineMarkup(nodes)

	if strings.HasSuffix(str, "}}") {
		str = str[:len(str)-2]
		p.curState.isPlugin = false
	}

	s := strings.SplitN(str, " ", 1)
	pluginName := s[0]
	node := Node{
		Kind: NodePlugin,
		Tag:  pluginName,
	}

	if len(s) > 1 {
		node.Content = s[1]
	} else if p.curState.isPlugin {
		node.Content = "\n"
	}

	nodes = append(nodes, node)

	return nodes
}

func (p *parser) parseParagraph(nodes []Node, str string) []Node {
	p.curState.isParagraph = true
	nodes = p.parseMultiLineMarkup(nodes)
	nodes, _ = p.parseInline(nodes, str)
	return nodes
}

func (p *parser) parse(nodes []Node, str string) ([]Node, error) {
	if p.prevState.isPlugin {
		if str == "}}" {
			p.prevState.isPlugin = false
		} else {
			nodes[len(nodes)-1].Content += fmt.Sprintf("%s\n", str)
		}
		return nodes, nil
	}

	switch {
	case str == "":
		nodes = p.parseLineBreak(nodes)
	case strings.HasPrefix(str, "!!!"):
		nodes = p.parseHeading(nodes, 3, trimPrefixSpace(str[3:]))
	case strings.HasPrefix(str, "!!"):
		nodes = p.parseHeading(nodes, 2, trimPrefixSpace(str[2:]))
	case strings.HasPrefix(str, "!"):
		nodes = p.parseHeading(nodes, 1, trimPrefixSpace(str[1:]))
	case strings.HasPrefix(str, "***"):
		nodes = p.parseList(nodes, 3, typeUL, trimPrefixSpace(str[3:]))
	case strings.HasPrefix(str, "**"):
		nodes = p.parseList(nodes, 2, typeUL, trimPrefixSpace(str[2:]))
	case strings.HasPrefix(str, "*"):
		nodes = p.parseList(nodes, 1, typeUL, trimPrefixSpace(str[1:]))
	case strings.HasPrefix(str, "+++"):
		nodes = p.parseList(nodes, 3, typeOL, trimPrefixSpace(str[3:]))
	case strings.HasPrefix(str, "++"):
		nodes = p.parseList(nodes, 2, typeOL, trimPrefixSpace(str[2:]))
	case strings.HasPrefix(str, "+"):
		nodes = p.parseList(nodes, 1, typeOL, trimPrefixSpace(str[1:]))
	case strings.HasPrefix(str, " "):
		nodes = p.parsePreformatted(nodes, str[1:])
	case strings.HasPrefix(str, ","):
		nodes = p.parseTable(nodes, str)
	case strings.HasPrefix(str, "//"):
		nodes = p.parseComment(nodes, trimPrefixSpace(str[2:]))
	case strings.HasPrefix(str, "{{"):
		nodes = p.parsePlugin(nodes, trimPrefixSpace(str[2:]))
	default:
		nodes = p.parseParagraph(nodes, str)
	}

	// update state
	p.prevState = p.curState
	p.curState = parserState{}

	return nodes, nil
}

func Parse(r io.Reader) ([]Node, error) {
	nodes := make([]Node, 0)

	p := &parser{}

	bs := bufio.NewScanner(r)
	for bs.Scan() {
		var err error
		nodes, err = p.parse(nodes, bs.Text())
		if err != nil {
			return nil, err
		}
	}
	// Post task
	nodes = p.parseMultiLineMarkup(nodes)

	return nodes, nil
}
