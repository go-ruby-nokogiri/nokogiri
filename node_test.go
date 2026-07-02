// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

import "testing"

const navXML = `<root><a>1</a><!--c--><b>2</b><d><e>3</e></d></root>`

func TestNavigation(t *testing.T) {
	d, err := XML(navXML)
	if err != nil {
		t.Fatal(err)
	}
	root := d.Root()
	if root.NodeType() != ElementNode || !root.IsElement() {
		t.Fatal("root type")
	}
	if root.Document() != d {
		t.Fatal("document link")
	}
	a := root.FirstChild()
	if a.Name != "a" || a.Parent() != root {
		t.Fatal("first child / parent")
	}
	if root.LastChild().Name != "d" {
		t.Fatal("last child")
	}
	// a -> comment -> b -> d
	if a.Next().NodeType() != CommentNode || !a.Next().IsComment() {
		t.Fatal("next comment")
	}
	if a.NextElement().Name != "b" {
		t.Fatal("next element skips comment")
	}
	b := a.NextElement()
	if b.PreviousElement().Name != "a" {
		t.Fatal("previous element")
	}
	if b.Previous().NodeType() != CommentNode {
		t.Fatal("previous comment")
	}
	// deep node reaches root
	e := root.LastChild().FirstChild()
	if e.Root() != root {
		t.Fatal("root from deep")
	}
	if e.NextElement() != nil || e.PreviousElement() != nil {
		t.Fatal("only child element siblings nil")
	}
}

func TestChildren(t *testing.T) {
	d, _ := XML(navXML)
	root := d.Root()
	if root.Children().Length() != 4 {
		t.Fatalf("children = %d", root.Children().Length())
	}
	if root.ElementChildren().Length() != 3 {
		t.Fatalf("element children = %d", root.ElementChildren().Length())
	}
}

func TestNodeNameAndTypes(t *testing.T) {
	d, _ := XML(`<r xmlns:x="urn:x"><x:child/><![CDATA[data]]><?pi go?></r>`)
	root := d.Root()
	if root.NodeName() != "r" {
		t.Fatalf("name %q", root.NodeName())
	}
	c := root.FirstChild()
	if c.NodeName() != "x:child" {
		t.Fatalf("qname %q", c.NodeName())
	}
	cdata := c.Next()
	if !cdata.IsCDATA() || !cdata.IsText() || cdata.NodeName() != "#cdata-section" {
		t.Fatalf("cdata %q", cdata.NodeName())
	}
	pi := cdata.Next()
	if pi.NodeType() != ProcessingInstructionNode || pi.NodeName() != "pi" {
		t.Fatalf("pi %q", pi.NodeName())
	}
	// document node name
	if d.Node.NodeName() != "document" {
		t.Fatalf("doc name %q", d.Node.NodeName())
	}
}

func TestTextNodeName(t *testing.T) {
	d, _ := XML(`<r>hi</r>`)
	txt := d.Root().FirstChild()
	if txt.NodeName() != "text" {
		t.Fatalf("text name %q", txt.NodeName())
	}
	if d.Root().Content() != "hi" {
		t.Fatalf("content %q", d.Root().Content())
	}
}

func TestCommentNodeName(t *testing.T) {
	d, _ := XML(`<r><!--x--></r>`)
	c := d.Root().FirstChild()
	if c.NodeName() != "comment" {
		t.Fatalf("comment name %q", c.NodeName())
	}
}

func TestRootNilWhenNoElement(t *testing.T) {
	n := &Node{Type: ElementNode}
	if n.Root() != nil {
		t.Fatal("root without doc should be nil")
	}
}
