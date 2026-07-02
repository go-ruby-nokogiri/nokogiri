// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

import "testing"

func TestBuilderXML(t *testing.T) {
	b := NewBuilder()
	b.Element("catalog", func(b *Builder) {
		b.Element("book", func(b *Builder) {
			b.Attr("id", "b1")
			b.ElementText("title", "Alpha")
			b.Comment("note")
			b.CDATA("<raw>")
		})
	})
	want := `<catalog><book id="b1"><title>Alpha</title><!--note--><![CDATA[<raw>]]></book></catalog>`
	if got := b.ToXML(); got != want {
		t.Fatalf("builder xml:\n got %q\nwant %q", got, want)
	}
	if b.Root().Name != "catalog" {
		t.Fatal("root")
	}
	if b.Document().Root() == nil {
		t.Fatal("document")
	}
	// query the built tree
	set, err := b.Document().CSS("book#b1 title")
	if err != nil || set.Length() != 1 {
		t.Fatalf("query built: %v %d", err, set.Length())
	}
}

func TestBuilderHTML(t *testing.T) {
	b := NewHTMLBuilder()
	b.Element("div", func(b *Builder) {
		b.Element("br", nil) // void, no closure
		b.Text("hi")
	})
	if got := b.ToHTML(); got != `<div><br>hi</div>` {
		t.Fatalf("builder html: %q", got)
	}
}

func TestBuilderAttrOnNonElement(t *testing.T) {
	b := NewBuilder()
	// Attr at document level (current is document node) is a no-op.
	b.Attr("x", "1")
	b.Element("r", nil)
	if b.ToXML() != `<r/>` {
		t.Fatalf("builder attr no-op: %q", b.ToXML())
	}
}
