// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

// Package nokogiri is a pure-Go (no cgo) reimplementation of the core of Ruby's
// Nokogiri HTML/XML toolkit. Nokogiri is normally a C extension over libxml2 and
// libxslt; this library instead builds on the pure-Go golang.org/x/net/html
// tag-soup parser (for Nokogiri::HTML) and encoding/xml (for Nokogiri::XML),
// exposes a single Node tree over both, and layers an XPath 1.0 engine plus a
// CSS-selector-to-XPath compiler on top so that at_css/css/at_xpath/xpath behave
// as Ruby programs expect — all with CGO_ENABLED=0.
package nokogiri

import "strings"

// NodeType enumerates the node kinds Nokogiri exposes. The numeric values match
// libxml2's node type constants that Nokogiri surfaces through Node#type so that
// ported Ruby code comparing against Nokogiri::XML::Node::ELEMENT_NODE (== 1) and
// friends keeps working.
type NodeType int

const (
	// ElementNode is a tag such as <div> or <book>.
	ElementNode NodeType = 1
	// AttributeNode is an attribute; only produced when an attribute is exposed
	// as a node (e.g. via an XPath attribute axis result).
	AttributeNode NodeType = 2
	// TextNode is character data between tags.
	TextNode NodeType = 3
	// CDATANode is a <![CDATA[ ... ]]> section.
	CDATANode NodeType = 4
	// CommentNode is an <!-- ... --> comment.
	CommentNode NodeType = 8
	// DocumentNode is the root document container.
	DocumentNode NodeType = 9
	// DoctypeNode is a <!DOCTYPE ...> declaration.
	DoctypeNode NodeType = 10
	// ProcessingInstructionNode is a <?target data?> instruction.
	ProcessingInstructionNode NodeType = 7
)

// Attr is a single attribute on an element. Namespace holds the resolved
// namespace URI when the attribute is namespaced (empty otherwise); Prefix holds
// the literal prefix as written (e.g. "xml" for xml:lang).
type Attr struct {
	Name      string
	Value     string
	Prefix    string
	Namespace string
}

// Namespace is an in-scope xmlns declaration visible from a node.
type Namespace struct {
	Prefix string // "" for the default namespace
	URI    string
}

// Node is the shared DOM node produced by BOTH the HTML and the XML parsers and
// consumed by the XPath and CSS engines. It is a mutable, doubly-linked tree: a
// node knows its parent, first/last child, and previous/next sibling. The public
// method set mirrors the slice of Nokogiri::XML::Node that this library targets.
type Node struct {
	Type NodeType

	// name is the element/PI/attribute name (local part kept in Name, prefix in
	// Prefix). For text/comment/cdata nodes it is the libxml2 pseudo-name
	// ("text", "comment", "#cdata-section").
	Name   string
	Prefix string // namespace prefix as written, e.g. "svg" in <svg:rect>
	NsURI  string // resolved namespace URI for this element, if any

	// content holds character data for text/comment/cdata/PI nodes.
	content string

	Attrs []*Attr

	// nsDecls are the xmlns declarations introduced *on* this element.
	nsDecls []*Namespace

	// doc points back at the owning document (nil only for a detached fragment
	// root before insertion). It is set for every node reachable from a Document.
	doc *Document

	parent                *Node
	firstChild, lastChild *Node
	prev, next            *Node
}

// Document is the root of a parsed tree. It embeds a Node (the DocumentNode) and
// records whether it came from the HTML or the XML parser, which controls default
// serialization (HTML void elements, etc.).
type Document struct {
	Node
	html bool
	// encoding is the character encoding declared in the source XML declaration
	// (e.g. "UTF-8"), preserved so #to_xml can reproduce it; empty when none.
	encoding string
	// errors accumulates non-fatal parse diagnostics (Nokogiri exposes #errors).
	errors []string
}

// NodeType returns the node's kind.
func (n *Node) NodeType() NodeType { return n.Type }

// Document returns the owning document.
func (n *Node) Document() *Document { return n.doc }

// Parent returns the node's parent, or nil at the document root.
func (n *Node) Parent() *Node { return n.parent }

// FirstChild returns the first child node, or nil.
func (n *Node) FirstChild() *Node { return n.firstChild }

// LastChild returns the last child node, or nil.
func (n *Node) LastChild() *Node { return n.lastChild }

// Next returns the next sibling node, or nil. Named to mirror Nokogiri#next.
func (n *Node) Next() *Node { return n.next }

// Previous returns the previous sibling node, or nil.
func (n *Node) Previous() *Node { return n.prev }

// NextElement returns the next sibling that is an element (Nokogiri#next_element).
func (n *Node) NextElement() *Node {
	for s := n.next; s != nil; s = s.next {
		if s.Type == ElementNode {
			return s
		}
	}
	return nil
}

// PreviousElement returns the previous element sibling.
func (n *Node) PreviousElement() *Node {
	for s := n.prev; s != nil; s = s.prev {
		if s.Type == ElementNode {
			return s
		}
	}
	return nil
}

// Children returns the element/text/comment children as a NodeSet.
func (n *Node) Children() *NodeSet {
	var out []*Node
	for c := n.firstChild; c != nil; c = c.next {
		out = append(out, c)
	}
	return &NodeSet{nodes: out}
}

// ElementChildren returns only the element children (Nokogiri#element_children).
func (n *Node) ElementChildren() *NodeSet {
	var out []*Node
	for c := n.firstChild; c != nil; c = c.next {
		if c.Type == ElementNode {
			out = append(out, c)
		}
	}
	return &NodeSet{nodes: out}
}

// Root returns the document's root element (the first element child of the
// document node), reached from any node.
func (n *Node) Root() *Node {
	d := n.doc
	if d == nil {
		return nil
	}
	for c := d.firstChild; c != nil; c = c.next {
		if c.Type == ElementNode {
			return c
		}
	}
	return nil
}

// IsElement reports whether n is an element node.
func (n *Node) IsElement() bool { return n.Type == ElementNode }

// IsText reports whether n is a text or CDATA node.
func (n *Node) IsText() bool { return n.Type == TextNode || n.Type == CDATANode }

// IsComment reports whether n is a comment node.
func (n *Node) IsComment() bool { return n.Type == CommentNode }

// IsCDATA reports whether n is a CDATA node.
func (n *Node) IsCDATA() bool { return n.Type == CDATANode }

// NodeName returns the libxml2/Nokogiri node name: the qualified tag name for
// elements, or a pseudo-name for the other node types.
func (n *Node) NodeName() string {
	switch n.Type {
	case TextNode:
		return "text"
	case CommentNode:
		return "comment"
	case CDATANode:
		return "#cdata-section"
	case DocumentNode:
		return "document"
	case ProcessingInstructionNode:
		return n.Name
	default:
		if n.Prefix != "" {
			return n.Prefix + ":" + n.Name
		}
		return n.Name
	}
}

// Text returns the concatenated character data of the node and all descendants,
// matching Nokogiri#text / #content / #inner_text.
func (n *Node) Text() string {
	var b strings.Builder
	// Leaf nodes whose value is their own character data (text/cdata, and the
	// attribute/comment/PI leaves for which #text simply returns the content)
	// yield that content directly; an element yields the text of its descendants
	// (comments and PIs do not contribute, matching libxml2/XPath string-value).
	switch n.Type {
	case AttributeNode, CommentNode, ProcessingInstructionNode:
		return n.content
	}
	var walk func(*Node)
	walk = func(x *Node) {
		switch x.Type {
		case TextNode, CDATANode:
			b.WriteString(x.content)
		case CommentNode, ProcessingInstructionNode:
			// skip: not part of an element's text
		default:
			for c := x.firstChild; c != nil; c = c.next {
				walk(c)
			}
		}
	}
	walk(n)
	return b.String()
}

// Content is an alias for Text (Nokogiri#content).
func (n *Node) Content() string { return n.Text() }
