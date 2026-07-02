// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

// Builder programmatically constructs an XML (or HTML) document tree, the Go
// analogue of Nokogiri::XML::Builder. Where Ruby leans on method_missing and
// blocks, this exposes an explicit, chainable API: Element opens a child element,
// runs the supplied closure to populate it, and closes it; Text/Comment/CDATA add
// leaf nodes; Attr sets an attribute on the current element.
type Builder struct {
	doc *Document
	cur *Node
}

// NewBuilder starts a new XML document builder.
func NewBuilder() *Builder {
	d := &Document{html: false}
	d.Type = DocumentNode
	d.doc = d
	return &Builder{doc: d, cur: &d.Node}
}

// NewHTMLBuilder starts a new HTML document builder (HTML serialization rules).
func NewHTMLBuilder() *Builder {
	b := NewBuilder()
	b.doc.html = true
	return b
}

// Document returns the built document.
func (b *Builder) Document() *Document { return b.doc }

// Root returns the document's root element (nil until one is added).
func (b *Builder) Root() *Node { return b.doc.Root() }

// Element opens a child element named name under the current node, invokes fn (if
// non-nil) with the builder positioned inside it, then restores the previous
// current node. Returns the builder for chaining.
func (b *Builder) Element(name string, fn func(*Builder)) *Builder {
	el := b.doc.NewElement(name)
	b.cur.AddChild(el)
	if fn != nil {
		prev := b.cur
		b.cur = el
		fn(b)
		b.cur = prev
	}
	return b
}

// ElementText is the common leaf case: an element whose only content is text.
func (b *Builder) ElementText(name, text string) *Builder {
	return b.Element(name, func(inner *Builder) {
		inner.Text(text)
	})
}

// Attr sets an attribute on the current element.
func (b *Builder) Attr(name, value string) *Builder {
	if b.cur.Type == ElementNode {
		b.cur.SetAttribute(name, value)
	}
	return b
}

// Text appends a text node to the current element.
func (b *Builder) Text(s string) *Builder {
	b.cur.AddChild(b.doc.NewText(s))
	return b
}

// Comment appends a comment node to the current element.
func (b *Builder) Comment(s string) *Builder {
	b.cur.AddChild(b.doc.NewComment(s))
	return b
}

// CDATA appends a CDATA node to the current element.
func (b *Builder) CDATA(s string) *Builder {
	b.cur.AddChild(b.doc.NewCDATA(s))
	return b
}

// ToXML serializes the built document with XML rules.
func (b *Builder) ToXML() string { return b.doc.ToXML() }

// ToHTML serializes the built document with HTML rules.
func (b *Builder) ToHTML() string { return b.doc.ToHTML() }
