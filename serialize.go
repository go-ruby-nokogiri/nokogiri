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

// htmlRawText are the HTML elements whose character-data children are serialized
// verbatim (no entity escaping), matching the WHATWG "raw text" / "escapable raw
// text" serialization used by Nokogiri::HTML5.
var htmlRawText = map[string]bool{
	"script": true, "style": true, "xmp": true, "iframe": true,
	"noembed": true, "noframes": true, "plaintext": true, "noscript": true,
}

// xmlIndent is libxml2's default indentation unit (two spaces per level), which
// Nokogiri#to_xml reproduces.
const xmlIndent = "  "

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

// writeNode serializes n (and its subtree) to b. html selects HTML rules (void
// elements, raw-text script/style, no self-closing); format enables libxml2-style
// pretty-printing (indentation), which applies to XML only; level is the current
// indentation depth.
func writeNode(b *strings.Builder, n *Node, html, format bool, level int) {
	switch n.Type {
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
		writeElement(b, n, html, format, level)
	case DocumentNode:
		for c := n.firstChild; c != nil; c = c.next {
			writeNode(b, c, html, format, level)
		}
	}
}

// writeElement serializes an element node, applying the void/raw-text HTML rules
// and libxml2's sticky-downward formatting rule: an element whose children are all
// non-character-data (elements, comments, PIs) is printed with each child on its
// own indented line, and — crucially — once a level is un-formatted (because it
// holds any text/CDATA), the entire subtree below it is printed inline too.
func writeElement(b *strings.Builder, n *Node, html, format bool, level int) {
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

	lname := strings.ToLower(name)
	if html && htmlVoidElements[lname] {
		b.WriteString(">")
		return
	}
	if n.firstChild == nil {
		if html {
			b.WriteString("></")
			b.WriteString(name)
			b.WriteString(">")
		} else {
			b.WriteString("/>")
		}
		return
	}
	if html && htmlRawText[lname] {
		b.WriteString(">")
		for c := n.firstChild; c != nil; c = c.next {
			if c.Type == TextNode || c.Type == CDATANode {
				b.WriteString(c.content)
			} else {
				writeNode(b, c, html, false, level+1)
			}
		}
		b.WriteString("</")
		b.WriteString(name)
		b.WriteString(">")
		return
	}

	localFormat := format
	if localFormat {
		for c := n.firstChild; c != nil; c = c.next {
			if c.Type == TextNode || c.Type == CDATANode {
				localFormat = false
				break
			}
		}
	}
	b.WriteString(">")
	for c := n.firstChild; c != nil; c = c.next {
		if localFormat {
			b.WriteString("\n")
			b.WriteString(strings.Repeat(xmlIndent, level+1))
		}
		writeNode(b, c, html, localFormat, level+1)
	}
	if localFormat {
		b.WriteString("\n")
		b.WriteString(strings.Repeat(xmlIndent, level))
	}
	b.WriteString("</")
	b.WriteString(name)
	b.WriteString(">")
}

// writeXMLDocument serializes a DocumentNode with XML rules, emitting the XML
// declaration (preserving the parsed encoding) and a trailing newline after each
// top-level node, exactly as Nokogiri::XML::Document#to_xml does.
func writeXMLDocument(b *strings.Builder, n *Node) {
	b.WriteString(`<?xml version="1.0"`)
	if n.doc != nil && n.doc.encoding != "" {
		b.WriteString(` encoding="`)
		b.WriteString(n.doc.encoding)
		b.WriteString(`"`)
	}
	b.WriteString("?>\n")
	for c := n.firstChild; c != nil; c = c.next {
		writeNode(b, c, false, true, 0)
		b.WriteString("\n")
	}
}

// ToHTML serializes the node using HTML rules (Nokogiri#to_html). HTML output is
// not pretty-printed (WHATWG serialization adds no whitespace).
func (n *Node) ToHTML() string {
	var b strings.Builder
	writeNode(&b, n, true, false, 0)
	return b.String()
}

// ToXML serializes the node using XML rules (Nokogiri#to_xml), reproducing
// libxml2's indentation. A DocumentNode additionally emits the XML declaration and
// a trailing newline.
func (n *Node) ToXML() string {
	var b strings.Builder
	if n.Type == DocumentNode {
		writeXMLDocument(&b, n)
	} else {
		writeNode(&b, n, false, true, 0)
	}
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
// Children of a raw-text element (script/style/…) are emitted verbatim, matching
// WHATWG serialization.
func (n *Node) InnerHTML() string {
	var b strings.Builder
	raw := htmlRawText[strings.ToLower(n.NodeName())]
	for c := n.firstChild; c != nil; c = c.next {
		if raw && (c.Type == TextNode || c.Type == CDATANode) {
			b.WriteString(c.content)
		} else {
			writeNode(&b, c, true, false, 0)
		}
	}
	return b.String()
}

// InnerXML serializes the node's children with XML rules (no pretty-printing).
func (n *Node) InnerXML() string {
	var b strings.Builder
	for c := n.firstChild; c != nil; c = c.next {
		writeNode(&b, c, false, false, 0)
	}
	return b.String()
}
