// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

import "testing"

func TestSerializeXMLFormatting(t *testing.T) {
	// Element-only children are indented; a level with a text/CDATA child (and its
	// whole subtree) stays inline, reproducing libxml2's sticky-downward rule.
	cases := map[string]string{
		`<a><b/><c/></a>`:           "<a>\n  <b/>\n  <c/>\n</a>",
		`<a><b><c/></b></a>`:        "<a>\n  <b>\n    <c/>\n  </b>\n</a>",
		`<p>x <span>y</span> z</p>`: "<p>x <span>y</span> z</p>",
		`<a><?p 1?><?q 2?></a>`:     "<a>\n  <?p 1?>\n  <?q 2?>\n</a>",
		`<a><![CDATA[<r>]]></a>`:    "<a><![CDATA[<r>]]></a>",
	}
	for in, want := range cases {
		d, err := XML(in)
		if err != nil {
			t.Fatalf("parse %q: %v", in, err)
		}
		if got := d.Root().ToXML(); got != want {
			t.Fatalf("to_xml %q = %q, want %q", in, got, want)
		}
	}
}

func TestSerializeXMLDocumentDeclaration(t *testing.T) {
	d, _ := XML(`<?xml version="1.0" encoding="ISO-8859-1"?>` + "\n" + `<r><a/></r>`)
	if d.encoding != "ISO-8859-1" {
		t.Fatalf("encoding = %q", d.encoding)
	}
	want := "<?xml version=\"1.0\" encoding=\"ISO-8859-1\"?>\n<r>\n  <a/>\n</r>\n"
	if got := d.Node.ToXML(); got != want {
		t.Fatalf("doc to_xml = %q", got)
	}
	// Without a declaration the canonical one is still emitted, and doc-level
	// whitespace-only text is dropped.
	d2, _ := XML("  <r/>  ")
	if got := d2.Node.ToXML(); got != "<?xml version=\"1.0\"?>\n<r/>\n" {
		t.Fatalf("no-decl to_xml = %q", got)
	}
}

func TestDeclEncoding(t *testing.T) {
	cases := map[string]string{
		`version="1.0"`:                  "",
		`version="1.0" encoding="UTF-8"`: "UTF-8",
		`encoding='latin1'`:              "latin1",
		`encoding`:                       "",
		`encoding = `:                    "",
		`encoding=x`:                     "",
		`encoding="unterminated`:         "",
	}
	for in, want := range cases {
		if got := declEncoding(in); got != want {
			t.Fatalf("declEncoding(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSerializeHTML5RawText(t *testing.T) {
	d, _ := HTML5(`<div><script>if(a<b&&c>d){}</script><style>a>b{x:1}</style></div>`)
	div, _ := d.AtCSS("div")
	if got := div.ToHTML(); got != `<div><script>if(a<b&&c>d){}</script><style>a>b{x:1}</style></div>` {
		t.Fatalf("raw text = %q", got)
	}
	// inner_html of a raw-text element is emitted verbatim.
	s, _ := d.AtCSS("script")
	if got := s.InnerHTML(); got != `if(a<b&&c>d){}` {
		t.Fatalf("inner_html script = %q", got)
	}
}

func TestSerializeHTML5EmptyAndDocument(t *testing.T) {
	d, _ := HTML5(`<div></div>`)
	div, _ := d.AtCSS("div")
	if got := div.ToHTML(); got != `<div></div>` {
		t.Fatalf("empty div = %q", got)
	}
	// Full HTML5 document serialization: implied html/head/body, void <br>.
	d2, _ := HTML5(`<p>hi<br>x</p>`)
	if got := d2.Node.ToHTML(); got != `<html><head></head><body><p>hi<br>x</p></body></html>` {
		t.Fatalf("html5 doc = %q", got)
	}
}

func TestSerializeRawTextNonTextChild(t *testing.T) {
	// A raw-text element containing a non-text child (constructed by hand) takes
	// the recursive branch inside the raw-text serializer.
	d := NewDocument()
	d.html = true
	s := d.NewElement("script")
	s.AddChild(d.NewComment("c"))
	if got := s.ToHTML(); got != "<script><!--c--></script>" {
		t.Fatalf("raw text non-text child = %q", got)
	}
}

func TestHTML5Alias(t *testing.T) {
	d, err := HTML5(`<p>hi</p>`)
	if err != nil {
		t.Fatal(err)
	}
	p, _ := d.AtCSS("p")
	if p == nil || p.Text() != "hi" {
		t.Fatalf("HTML5 alias parse failed: %+v", p)
	}
}
