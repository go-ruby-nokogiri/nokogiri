// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

import "testing"

func TestDecimalLiteralLeadingDot(t *testing.T) {
	d, _ := XML(`<r><n>1</n></r>`)
	if evalNum(t, d, ".5 + .5") != 1 {
		t.Error("leading-dot number literal")
	}
}

func TestSiblingInsertLastFirstWithParent(t *testing.T) {
	d, _ := XML(`<r><only/></r>`)
	root := d.Root()
	only := root.FirstChild()
	// only.next is nil and only has a parent -> updates parent.lastChild
	only.AddNextSibling(d.NewElement("after"))
	if root.LastChild().Name != "after" {
		t.Fatalf("lastChild: %s", root.LastChild().Name)
	}
	// first child, prev nil, has parent -> updates parent.firstChild
	first := root.FirstChild()
	first.AddPreviousSibling(d.NewElement("before"))
	if root.FirstChild().Name != "before" {
		t.Fatalf("firstChild: %s", root.FirstChild().Name)
	}
}

func TestXMLDoctypeSerialization(t *testing.T) {
	d, err := XML(`<!DOCTYPE greeting SYSTEM "hello.dtd"><greeting/>`)
	if err != nil {
		t.Fatal(err)
	}
	out := d.Node.ToXML()
	if want := `<!DOCTYPE greeting SYSTEM "hello.dtd">`; out[:len(want)] != want {
		t.Fatalf("xml doctype serialize: %q", out)
	}
}

func TestRootNilNoElementChild(t *testing.T) {
	// A document whose only children are comments has no root element.
	d, err := XML(`<!-- just a comment --><r/>`)
	if err != nil {
		t.Fatal(err)
	}
	if d.Root().Name != "r" {
		t.Fatal("root should be r")
	}
	// A document node with no element child at all.
	empty := &Document{}
	empty.Type = DocumentNode
	empty.doc = empty
	empty.AddChild(empty.NewComment("x"))
	if empty.Root() != nil {
		t.Fatal("root nil when no element child")
	}
}

func TestAttributeAxisOnNonElement(t *testing.T) {
	d, _ := XML(`<r>text</r>`)
	// attribute axis from a text node yields nothing
	set, _ := d.XPath("//text()/@*")
	if set.Length() != 0 {
		t.Fatalf("attr axis on text: %d", set.Length())
	}
}

func TestNameTestAxisMismatch(t *testing.T) {
	d, _ := XML(`<r a="1"><a/></r>`)
	// child axis with an element name test never matches attribute nodes, and
	// the attribute axis with a name test never matches elements.
	set, _ := d.XPath("//r/child::a") // element child named a
	if set.Length() != 1 {
		t.Fatalf("child a: %d", set.Length())
	}
	set, _ = d.XPath("//r/attribute::a") // attribute named a
	if set.Length() != 1 || set.First().Type != AttributeNode {
		t.Fatalf("attribute a: %d", set.Length())
	}
}

func TestPrecedingAxisFull(t *testing.T) {
	d, _ := XML(`<r><a><b/></a><c><d/></c></r>`)
	// preceding of d: a, b (not c, not ancestors r)
	set, _ := d.XPath("//d/preceding::*")
	names := map[string]bool{}
	set.Each(func(n *Node) { names[n.Name] = true })
	if !names["a"] || !names["b"] || names["r"] || names["c"] {
		t.Fatalf("preceding set: %v", names)
	}
}

func TestParseNodeTestBadFunction(t *testing.T) {
	d, _ := XML(`<r/>`)
	// a function name used where a node test is expected
	if _, err := d.XPath("//count()"); err == nil {
		t.Error("expected node-test error for count()")
	}
}

func TestRoundSpecial(t *testing.T) {
	d, _ := XML(`<r/>`)
	// round(NaN) and round(Inf) return the value unchanged
	if evalStr(t, d, "string(round(number('x')))") != "NaN" {
		t.Error("round NaN")
	}
	if evalStr(t, d, "string(round(1 div 0))") != "Infinity" {
		t.Error("round Inf")
	}
}
