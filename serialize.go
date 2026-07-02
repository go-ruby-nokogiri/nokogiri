// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

import "strings"

// htmlVoidElements are the HTML elements serialized without an end tag.
var htmlVoidElements = map[string]bool{
	"area": true, "base": true, "br": true, "col": true, "embed": true,
	"hr": true, "img": true, "input": true, "link": true, "meta": true,
	"param": true, "source": true, "track": true, "wbr": true,
}

// escapeText escapes character data for text content (& < >).
func escapeText(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// escapeAttr escapes an attribute value (& < > ").
func escapeAttr(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// serialize writes n (and its subtree) to b. html selects HTML rules (void
// elements, no self-closing) vs XML rules (empty elements self-close).
func serialize(b *strings.Builder, n *Node, html bool) {
	switch n.Type {
	case DocumentNode:
		for c := n.firstChild; c != nil; c = c.next {
			serialize(b, c, html)
		}
	case TextNode:
		b.WriteString(escapeText(n.content))
	case CDATANode:
		b.WriteString("<![CDATA[")
		b.WriteString(n.content)
		b.WriteString("]]>")
	case CommentNode:
		b.WriteString("<!--")
		b.WriteString(n.content)
		b.WriteString("-->")
	case DoctypeNode:
		if html {
			b.WriteString("<!DOCTYPE ")
			b.WriteString(n.Name)
			b.WriteString(">")
		} else {
			b.WriteString("<!")
			b.WriteString(n.content)
			b.WriteString(">")
		}
	case ProcessingInstructionNode:
		b.WriteString("<?")
		b.WriteString(n.Name)
		if n.content != "" {
			b.WriteString(" ")
			b.WriteString(n.content)
		}
		b.WriteString("?>")
	case ElementNode:
		serializeElement(b, n, html)
	}
}

func serializeElement(b *strings.Builder, n *Node, html bool) {
	name := n.NodeName()
	b.WriteString("<")
	b.WriteString(name)
	for _, d := range n.nsDecls {
		b.WriteString(" xmlns")
		if d.Prefix != "" {
			b.WriteString(":")
			b.WriteString(d.Prefix)
		}
		b.WriteString(`="`)
		b.WriteString(escapeAttr(d.URI))
		b.WriteString(`"`)
	}
	for _, a := range n.Attrs {
		b.WriteString(" ")
		b.WriteString(a.qualified())
		b.WriteString(`="`)
		b.WriteString(escapeAttr(a.Value))
		b.WriteString(`"`)
	}

	if html && htmlVoidElements[strings.ToLower(name)] {
		b.WriteString(">")
		return
	}
	if n.firstChild == nil && !html {
		b.WriteString("/>")
		return
	}
	b.WriteString(">")
	for c := n.firstChild; c != nil; c = c.next {
		serialize(b, c, html)
	}
	b.WriteString("</")
	b.WriteString(name)
	b.WriteString(">")
}

// ToHTML serializes the node using HTML rules (Nokogiri#to_html).
func (n *Node) ToHTML() string {
	var b strings.Builder
	serialize(&b, n, true)
	return b.String()
}

// ToXML serializes the node using XML rules (Nokogiri#to_xml).
func (n *Node) ToXML() string {
	var b strings.Builder
	serialize(&b, n, false)
	return b.String()
}

// ToS serializes the node using the owning document's default (HTML rules for an
// HTML document, XML rules otherwise), matching Nokogiri#to_s.
func (n *Node) ToS() string {
	if n.doc != nil && n.doc.html {
		return n.ToHTML()
	}
	return n.ToXML()
}

// InnerHTML serializes the node's children with HTML rules (Nokogiri#inner_html).
func (n *Node) InnerHTML() string {
	var b strings.Builder
	for c := n.firstChild; c != nil; c = c.next {
		serialize(&b, c, true)
	}
	return b.String()
}

// InnerXML serializes the node's children with XML rules.
func (n *Node) InnerXML() string {
	var b strings.Builder
	for c := n.firstChild; c != nil; c = c.next {
		serialize(&b, c, false)
	}
	return b.String()
}
