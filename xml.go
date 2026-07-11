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
	// encoding/xml refuses a non-UTF-8 encoding declaration unless a CharsetReader
	// is supplied. We pass the bytes through unchanged so ASCII-compatible
	// declarations (ISO-8859-1, windows-1252, US-ASCII, …) parse structurally and
	// the declared encoding is preserved for #to_xml; we do not transcode the
	// bytes, which is where this differs from libxml2's full charset support.
	dec.CharsetReader = func(_ string, input io.Reader) (io.Reader, error) { return input, nil }
	// Keep the raw prefixes rather than have the decoder rewrite them, so we can
	// reproduce the source and resolve prefixes ourselves for XPath.
	var stack []*Node
	cur := &doc.Node

	for {
		startOff := dec.InputOffset()
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
			// encoding/xml collapses CDATA into CharData, so recover the CDATA node
			// type by inspecting the source span this token was decoded from.
			nt := TextNode
			if isCDATASpan(s, int(startOff), int(dec.InputOffset())) {
				nt = CDATANode
			}
			// libxml2/Nokogiri drop whitespace-only text nodes at the document level
			// (before/after the root element); keep them everywhere else.
			if cur == &doc.Node && nt == TextNode && strings.TrimSpace(string(t)) == "" {
				continue
			}
			cur.appendChildRaw(&Node{Type: nt, content: string(t), doc: doc})
		case xml.Comment:
			cur.appendChildRaw(&Node{Type: CommentNode, content: string(t), doc: doc})
		case xml.ProcInst:
			// The XML declaration is surfaced by encoding/xml as a "xml" proc-inst;
			// Nokogiri models it as document metadata (version/encoding), not a node.
			if t.Target == "xml" && cur == &doc.Node {
				doc.encoding = declEncoding(string(t.Inst))
				continue
			}
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

// newXMLElement builds an element node from a RawToken StartElement. In RawToken
// mode encoding/xml already splits a "prefix:local" name into Name.Space (the raw
// prefix) and Name.Local, so we take the prefix from there; xmlns declarations are
// separated from ordinary attributes.
func newXMLElement(t xml.StartElement) *Node {
	el := &Node{Type: ElementNode, Name: t.Name.Local, Prefix: t.Name.Space}
	for _, a := range t.Attr {
		switch {
		case a.Name.Space == "" && a.Name.Local == "xmlns":
			el.nsDecls = append(el.nsDecls, &Namespace{Prefix: "", URI: a.Value})
		case a.Name.Space == "xmlns":
			el.nsDecls = append(el.nsDecls, &Namespace{Prefix: a.Name.Local, URI: a.Value})
		default:
			el.Attrs = append(el.Attrs, &Attr{Name: a.Name.Local, Prefix: a.Name.Space, Value: a.Value})
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

// isCDATASpan reports whether the source bytes for a CharData token (from lo to
// hi) begin with a CDATA section marker, allowing us to distinguish CDATA from
// ordinary text after encoding/xml has collapsed the two.
func isCDATASpan(src string, lo, hi int) bool {
	if lo < 0 || hi > len(src) || lo >= hi {
		return false
	}
	seg := src[lo:hi]
	for len(seg) > 0 && (seg[0] == ' ' || seg[0] == '\t' || seg[0] == '\n' || seg[0] == '\r') {
		seg = seg[1:]
	}
	return strings.HasPrefix(seg, "<![CDATA[")
}

// declEncoding extracts the encoding pseudo-attribute from an XML declaration's
// instruction body (e.g. `version="1.0" encoding="UTF-8"` -> "UTF-8"). It returns
// "" when no encoding is declared.
func declEncoding(inst string) string {
	i := strings.Index(inst, "encoding")
	if i < 0 {
		return ""
	}
	rest := inst[i+len("encoding"):]
	rest = strings.TrimLeft(rest, " \t\r\n")
	if len(rest) == 0 || rest[0] != '=' {
		return ""
	}
	rest = strings.TrimLeft(rest[1:], " \t\r\n")
	if len(rest) == 0 || (rest[0] != '"' && rest[0] != '\'') {
		return ""
	}
	q := rest[0]
	rest = rest[1:]
	if j := strings.IndexByte(rest, q); j >= 0 {
		return rest[:j]
	}
	return ""
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
