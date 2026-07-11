// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

import "testing"

func TestAddChildAndPrepend(t *testing.T) {
	d, _ := XML(`<root/>`)
	root := d.Root()
	a := d.NewElement("a")
	b := d.NewElement("b")
	root.AddChild(a)
	root.AddChild(b)
	if root.ToXML() != "<root>\n  <a/>\n  <b/>\n</root>" {
		t.Fatalf("append: %q", root.ToXML())
	}
	c := d.NewElement("c")
	root.Prepend(c)
	if root.ToXML() != "<root>\n  <c/>\n  <a/>\n  <b/>\n</root>" {
		t.Fatalf("prepend: %q", root.ToXML())
	}
	// prepend into empty
	empty := d.NewElement("e")
	empty.Prepend(d.NewElement("f"))
	if empty.ToXML() != "<e>\n  <f/>\n</e>" {
		t.Fatalf("prepend empty: %q", empty.ToXML())
	}
}

func TestSiblingInsertion(t *testing.T) {
	d, _ := XML(`<root><b/></root>`)
	root := d.Root()
	b := root.FirstChild()
	b.AddPreviousSibling(d.NewElement("a"))
	b.AddNextSibling(d.NewElement("c"))
	if root.ToXML() != "<root>\n  <a/>\n  <b/>\n  <c/>\n</root>" {
		t.Fatalf("siblings: %q", root.ToXML())
	}
	// insert after last updates lastChild
	c := root.LastChild()
	c.AddNextSibling(d.NewElement("z"))
	if root.LastChild().Name != "z" {
		t.Fatal("lastChild not updated")
	}
	// insert before first updates firstChild
	a := root.FirstChild()
	a.AddPreviousSibling(d.NewElement("y"))
	if root.FirstChild().Name != "y" {
		t.Fatal("firstChild not updated")
	}
}

func TestRemoveAndReplace(t *testing.T) {
	d, _ := XML(`<root><a/><b/><c/></root>`)
	root := d.Root()
	b := root.FirstChild().Next()
	b.Remove()
	if root.ToXML() != "<root>\n  <a/>\n  <c/>\n</root>" {
		t.Fatalf("remove middle: %q", root.ToXML())
	}
	// replace first
	a := root.FirstChild()
	a.Replace(d.NewElement("x"))
	if root.ToXML() != "<root>\n  <x/>\n  <c/>\n</root>" {
		t.Fatalf("replace first: %q", root.ToXML())
	}
	// replace last
	last := root.LastChild()
	last.Replace(d.NewElement("y"))
	if root.LastChild().Name != "y" {
		t.Fatalf("replace last: %q", root.ToXML())
	}
	// replace middle
	d2, _ := XML(`<r><a/><m/><z/></r>`)
	m := d2.Root().FirstChild().Next()
	m.Replace(d2.NewElement("q"))
	if d2.Root().ToXML() != "<r>\n  <a/>\n  <q/>\n  <z/>\n</r>" {
		t.Fatalf("replace middle: %q", d2.Root().ToXML())
	}
}

func TestReparenting(t *testing.T) {
	d, _ := XML(`<root><a><child/></a><b/></root>`)
	root := d.Root()
	a := root.FirstChild()
	b := a.Next()
	child := a.FirstChild()
	b.AddChild(child) // move child from a to b
	if root.ToXML() != "<root>\n  <a/>\n  <b>\n    <child/>\n  </b>\n</root>" {
		t.Fatalf("reparent: %q", root.ToXML())
	}
}

func TestSetContent(t *testing.T) {
	d, _ := XML(`<root><a/><b/></root>`)
	root := d.Root()
	root.SetContent("hello")
	if root.ToXML() != `<root>hello</root>` {
		t.Fatalf("setcontent element: %q", root.ToXML())
	}
	// on a text node it sets content directly
	txt := d.NewText("x")
	txt.SetContent("y")
	if txt.content != "y" {
		t.Fatal("setcontent text")
	}
}

func TestFactories(t *testing.T) {
	d, _ := XML(`<r/>`)
	if d.NewElement("x").Type != ElementNode {
		t.Fatal("NewElement")
	}
	if d.NewText("t").Type != TextNode {
		t.Fatal("NewText")
	}
	if d.NewCDATA("c").Type != CDATANode {
		t.Fatal("NewCDATA")
	}
	if d.NewComment("c").Type != CommentNode {
		t.Fatal("NewComment")
	}
	// qualified element name splits prefix
	el := d.NewElement("ns:tag")
	if el.Prefix != "ns" || el.Name != "tag" {
		t.Fatal("NewElement qname")
	}
}
