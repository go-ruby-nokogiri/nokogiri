// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

import (
	"strings"

	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// HTML parses a real-world HTML document with the lenient, HTML5 tree-building
// algorithm (via the pure-Go golang.org/x/net/html parser), matching
// Nokogiri::HTML / Nokogiri::HTML5. Malformed "tag soup" is recovered exactly as
// a browser would: missing end tags are inferred, misnested tags are corrected,
// and implied <html>/<head>/<body> wrappers are added.
func HTML(s string) (*Document, error) {
	root, err := xhtml.Parse(strings.NewReader(s))
	if err != nil {
		return nil, err
	}
	doc := &Document{html: true}
	doc.Type = DocumentNode
	doc.doc = doc
	for c := root.FirstChild; c != nil; c = c.NextSibling {
		if n := convertHTML(c, doc); n != nil {
			doc.appendConverted(n)
		}
	}
	return doc, nil
}

// HTMLFragment parses a fragment of HTML with no implied document wrappers,
// matching Nokogiri::HTML::DocumentFragment.parse. The returned Document's
// children are the fragment's top-level nodes.
func HTMLFragment(s string) (*Document, error) {
	body := &xhtml.Node{Type: xhtml.ElementNode, Data: "body", DataAtom: atom.Body}
	nodes, err := xhtml.ParseFragment(strings.NewReader(s), body)
	if err != nil {
		return nil, err
	}
	doc := &Document{html: true}
	doc.Type = DocumentNode
	doc.doc = doc
	for _, c := range nodes {
		if n := convertHTML(c, doc); n != nil {
			doc.appendConverted(n)
		}
	}
	return doc, nil
}

// appendConverted appends an already-converted subtree as a child of the document
// node, wiring the parent/sibling links.
func (d *Document) appendConverted(n *Node) {
	n.parent = &d.Node
	if d.lastChild == nil {
		d.firstChild = n
		d.lastChild = n
		return
	}
	n.prev = d.lastChild
	d.lastChild.next = n
	d.lastChild = n
}

// convertHTML translates an x/net/html node (and its subtree) into our shared
// Node tree.
func convertHTML(h *xhtml.Node, doc *Document) *Node {
	var n *Node
	switch h.Type {
	case xhtml.ElementNode:
		n = &Node{Type: ElementNode, Name: h.Data, doc: doc}
		for _, a := range h.Attr {
			at := &Attr{Name: a.Key, Value: a.Val, Prefix: a.Namespace}
			n.Attrs = append(n.Attrs, at)
		}
	case xhtml.TextNode:
		return &Node{Type: TextNode, content: h.Data, doc: doc}
	case xhtml.CommentNode:
		return &Node{Type: CommentNode, content: h.Data, doc: doc}
	case xhtml.DoctypeNode:
		return &Node{Type: DoctypeNode, Name: h.Data, doc: doc}
	default:
		return nil
	}
	for c := h.FirstChild; c != nil; c = c.NextSibling {
		child := convertHTML(c, doc)
		if child == nil {
			continue
		}
		n.appendChildRaw(child)
	}
	return n
}

// appendChildRaw wires child as the last child during parsing (no re-parenting
// bookkeeping needed since the child is fresh).
func (n *Node) appendChildRaw(child *Node) {
	child.parent = n
	if n.lastChild == nil {
		n.firstChild = child
		n.lastChild = child
		return
	}
	child.prev = n.lastChild
	n.lastChild.next = child
	n.lastChild = child
}
