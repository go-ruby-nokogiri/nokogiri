// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

import (
	"encoding/xml"
	"io"
	"strings"
)

// XML parses a well-formed XML document into the shared Node tree, matching
// Nokogiri::XML. Namespaces are resolved: an element or attribute in a namespace
// carries its resolved URI, and the literal prefix as written is preserved for
// round-tripping. Unlike Nokogiri::HTML this path is strict; a malformed document
// returns an error.
func XML(s string) (*Document, error) {
	doc := &Document{html: false}
	doc.Type = DocumentNode
	doc.doc = doc

	dec := xml.NewDecoder(strings.NewReader(s))
	dec.Strict = true
	// Keep the raw prefixes rather than have the decoder rewrite them, so we can
	// reproduce the source and resolve prefixes ourselves for XPath.
	var stack []*Node
	cur := &doc.Node

	push := func(n *Node) {
		cur.appendChildRaw(n)
		n.doc = doc
		stack = append(stack, cur)
		cur = n
	}
	_ = push

	for {
		tok, err := dec.RawToken()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			el := newXMLElement(t)
			el.doc = doc
			cur.appendChildRaw(el)
			stack = append(stack, cur)
			cur = el
		case xml.EndElement:
			if len(stack) == 0 {
				return nil, &xml.SyntaxError{Msg: "unexpected end element " + t.Name.Local}
			}
			cur = stack[len(stack)-1]
			stack = stack[:len(stack)-1]
		case xml.CharData:
			cur.appendChildRaw(&Node{Type: TextNode, content: string(t), doc: doc})
		case xml.Comment:
			cur.appendChildRaw(&Node{Type: CommentNode, content: string(t), doc: doc})
		case xml.ProcInst:
			cur.appendChildRaw(&Node{
				Type: ProcessingInstructionNode, Name: t.Target,
				content: string(t.Inst), doc: doc,
			})
		case xml.Directive:
			cur.appendChildRaw(&Node{Type: DoctypeNode, Name: directiveName(string(t)), content: string(t), doc: doc})
		}
	}
	if len(stack) != 0 {
		return nil, &xml.SyntaxError{Msg: "unexpected EOF: unclosed element"}
	}
	resolveNamespaces(&doc.Node, nil)
	return doc, nil
}

// newXMLElement builds an element node from a RawToken StartElement, splitting
// qualified names into prefix + local and separating xmlns declarations from
// ordinary attributes.
func newXMLElement(t xml.StartElement) *Node {
	prefix, local := splitQName(t.Name.Local)
	el := &Node{Type: ElementNode, Name: local, Prefix: prefix}
	for _, a := range t.Attr {
		aPrefix, aLocal := splitQName(a.Name.Local)
		switch {
		case a.Name.Local == "xmlns":
			el.nsDecls = append(el.nsDecls, &Namespace{Prefix: "", URI: a.Value})
		case aPrefix == "xmlns":
			el.nsDecls = append(el.nsDecls, &Namespace{Prefix: aLocal, URI: a.Value})
		default:
			el.Attrs = append(el.Attrs, &Attr{Name: aLocal, Prefix: aPrefix, Value: a.Value})
		}
	}
	return el
}

// nsScope maps a prefix to its URI at a point in the tree.
type nsScope map[string]string

// resolveNamespaces walks the tree assigning NsURI to elements/attributes from
// the in-scope xmlns declarations.
func resolveNamespaces(n *Node, scope nsScope) {
	if len(n.nsDecls) > 0 {
		next := make(nsScope, len(scope)+len(n.nsDecls))
		for k, v := range scope {
			next[k] = v
		}
		for _, d := range n.nsDecls {
			next[d.Prefix] = d.URI
		}
		scope = next
	}
	if n.Type == ElementNode {
		if uri, ok := scope[n.Prefix]; ok {
			n.NsURI = uri
		}
		for _, a := range n.Attrs {
			if a.Prefix != "" && a.Prefix != "xml" {
				if uri, ok := scope[a.Prefix]; ok {
					a.Namespace = uri
				}
			}
		}
	}
	for c := n.firstChild; c != nil; c = c.next {
		resolveNamespaces(c, scope)
	}
}

// directiveName extracts the leading token of an XML directive, e.g. "DOCTYPE".
func directiveName(s string) string {
	s = strings.TrimSpace(s)
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' || s[i] == '\t' || s[i] == '\n' {
			return s[:i]
		}
	}
	return s
}
