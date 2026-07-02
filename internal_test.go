// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

import (
	"strings"
	"testing"
)

func TestErrorTypes(t *testing.T) {
	if evalError("boom").Error() != "boom" {
		t.Fatal("evalError")
	}
	if parseError("bad").Error() != "bad" {
		t.Fatal("parseError")
	}
}

func TestAttributeWildcardAndPI(t *testing.T) {
	d, _ := XML(`<r a="1" b="2"><?php x?></r>`)
	// @* selects all attributes
	set, _ := d.XPath("//r/@*")
	if set.Length() != 2 {
		t.Fatalf("@*: %d", set.Length())
	}
	// PI name mismatch
	set, _ = d.XPath("//processing-instruction('other')")
	if set.Length() != 0 {
		t.Fatalf("pi mismatch: %d", set.Length())
	}
}

func TestNamespaceWildcard(t *testing.T) {
	d, _ := XML(`<r><a/><b/></r>`)
	// namespace axis yields nothing (we do not model namespace nodes)
	set, _ := d.XPath("//r/namespace::*")
	if set.Length() != 0 {
		t.Fatalf("namespace axis: %d", set.Length())
	}
}

func TestNameTestNonMatching(t *testing.T) {
	d, _ := XML(`<r><a/></r>`)
	// attribute-axis name test against a non-attribute context yields nothing
	set, _ := d.XPath("//text()[self::a]")
	if set.Length() != 0 {
		t.Fatalf("name test non-element: %d", set.Length())
	}
	// name test on comment node fails element check
	d2, _ := XML(`<r><!--x--></r>`)
	set, _ = d2.XPath("//comment()/self::a")
	if set.Length() != 0 {
		t.Fatal("name test on comment")
	}
}

func TestNsMapMismatch(t *testing.T) {
	d, _ := XML(`<root xmlns:a="urn:a"><a:x/></root>`)
	// prefix registered to a URI that does not match the element's -> no match
	set, _ := d.Node.XPath("//a:x", map[string]string{"a": "urn:other"})
	if set.Length() != 0 {
		t.Fatalf("ns mismatch: %d", set.Length())
	}
}

func TestPanicPaths(t *testing.T) {
	d, _ := XML(`<r/>`)
	// A location step applied to a non-node-set operand: "1/a" -> toNodeList panic
	if _, err := d.XPath("(1)/a"); err == nil {
		t.Fatal("expected node-set error")
	}
	// undefined variable
	if _, err := evalXPath("$missing", &d.Node, nil, nil); err == nil {
		t.Fatal("expected undefined variable error")
	}
	// count() over a non-node-set argument
	if _, err := d.XPath("count(1)"); err == nil {
		t.Fatal("expected count arg error")
	}
	// sum over non-node-set
	if _, err := d.XPath("sum('x')"); err == nil {
		t.Fatal("expected sum arg error")
	}
}

func TestEatOpError(t *testing.T) {
	// unbalanced paren triggers eatOp failure
	if _, err := parseXPath("(1"); err == nil {
		t.Fatal("expected eatOp error")
	}
}

func TestStartsStepForms(t *testing.T) {
	d, _ := XML(`<r><a/></r>`)
	// "/" then a step, "/" alone
	set, _ := d.XPath("/")
	if set.Length() != 1 { // the document node
		t.Fatalf("root only: %d", set.Length())
	}
	// axis step after //
	set, _ = d.XPath("//child::a")
	if set.Length() != 1 {
		t.Fatalf("//child::a: %d", set.Length())
	}
	// @ attribute step start
	d2, _ := XML(`<r x="1"/>`)
	set, _ = d2.XPath("/r/@x")
	if set.Length() != 1 {
		t.Fatalf("@x: %d", set.Length())
	}
}

func TestUnknownAxisAndPseudo(t *testing.T) {
	d, _ := XML(`<r/>`)
	if _, err := d.XPath("bogus-axis::x"); err == nil {
		t.Fatal("expected unknown axis error")
	}
	hd, _ := HTML(`<div><p>x</p></div>`)
	if _, err := hd.CSS("p::unknown-pseudo-element-thing("); err == nil {
		t.Fatal("expected css error")
	}
}

func TestCompileNotVariants(t *testing.T) {
	d, _ := HTML(`<ul><li id="a" class="x">1</li><li id="b">2</li><li>3</li></ul>`)
	cases := []struct {
		sel  string
		want int
	}{
		{"li:not(#a)", 2},
		{"li:not(.x)", 2},
		{"li:not([id])", 1},
		{"li:not(*)", 0},
		{"li:not(:first-child)", 2},
	}
	for _, c := range cases {
		set, err := d.CSS(c.sel)
		if err != nil {
			t.Errorf("%q: %v", c.sel, err)
			continue
		}
		if set.Length() != c.want {
			t.Errorf("%q: got %d want %d", c.sel, set.Length(), c.want)
		}
	}
	// :not with a type selector
	d2, _ := HTML(`<div><p>a</p><span>b</span></div>`)
	set, _ := d2.CSS("div > :not(p)")
	if set.Length() != 1 {
		t.Fatalf(":not(type): %d", set.Length())
	}
	// :not() error propagation (bad inner attribute)
	if _, err := d.CSS("li:not([)"); err == nil {
		t.Fatal(":not bad inner")
	}
}

func TestSerializeStandaloneDocumentChildIteration(t *testing.T) {
	// A DocumentNode with mixed children serialized directly.
	d, _ := XML(`<?xml-stylesheet href="x"?><!--top--><r/>`)
	out := d.Node.ToXML()
	if !strings.Contains(out, "<r/>") || !strings.Contains(out, "<!--top-->") {
		t.Fatalf("document children: %q", out)
	}
}

func TestEscapeAllSpecials(t *testing.T) {
	n := &Node{Type: ElementNode, Name: "e"}
	n.Attrs = []*Attr{{Name: "a", Value: "<>&\""}}
	n.content = ""
	txt := &Node{Type: TextNode, content: "<>&"}
	n.firstChild, n.lastChild = txt, txt
	txt.parent = n
	if n.ToXML() != `<e a="&lt;&gt;&amp;&quot;">&lt;&gt;&amp;</e>` {
		t.Fatalf("escape: %q", n.ToXML())
	}
}

func TestDirectiveNameForms(t *testing.T) {
	if directiveName("DOCTYPE html") != "DOCTYPE" {
		t.Fatal("space")
	}
	if directiveName("ELEMENT") != "ELEMENT" {
		t.Fatal("no space")
	}
	if directiveName("A\tB") != "A" {
		t.Fatal("tab")
	}
	if directiveName("A\nB") != "A" {
		t.Fatal("newline")
	}
}

func TestIsCDATASpanBounds(t *testing.T) {
	if isCDATASpan("abc", -1, 2) {
		t.Fatal("negative lo")
	}
	if isCDATASpan("abc", 0, 99) {
		t.Fatal("hi past end")
	}
	if isCDATASpan("abc", 2, 1) {
		t.Fatal("lo>=hi")
	}
	if !isCDATASpan("  <![CDATA[x]]>", 0, 15) {
		t.Fatal("leading ws cdata")
	}
}
