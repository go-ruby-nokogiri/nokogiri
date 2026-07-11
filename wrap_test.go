// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

import "testing"

func TestWrapXML(t *testing.T) {
	d, _ := XML(`<root><a>hi</a></root>`)
	a, _ := d.Node.AtXPath("//a", nil)
	w, err := a.Wrap("<wrapper/>")
	if err != nil {
		t.Fatal(err)
	}
	if w.Name != "wrapper" {
		t.Fatalf("wrapper name = %q", w.Name)
	}
	if got := d.Root().ToXML(); got != "<root>\n  <wrapper>\n    <a>hi</a>\n  </wrapper>\n</root>" {
		t.Fatalf("wrapped = %q", got)
	}
	if a.Parent() != w || w.Parent().Name != "root" {
		t.Fatal("wrap did not re-parent correctly")
	}
}

func TestWrapHTML(t *testing.T) {
	d, _ := HTML(`<html><body><span>x</span></body></html>`)
	span, _ := d.AtCSS("span")
	w, err := span.Wrap("<div></div>")
	if err != nil {
		t.Fatal(err)
	}
	if w.Name != "div" || span.Parent() != w {
		t.Fatalf("html wrap failed: %+v", w)
	}
	if w.ToHTML() != "<div><span>x</span></div>" {
		t.Fatalf("html wrap serialize = %q", w.ToHTML())
	}
}

func TestWrapParseError(t *testing.T) {
	d, _ := XML(`<r><a/></r>`)
	a, _ := d.Node.AtXPath("//a", nil)
	if _, err := a.Wrap("<<<"); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestWrapNoElement(t *testing.T) {
	d, _ := XML(`<r><a/></r>`)
	a, _ := d.Node.AtXPath("//a", nil)
	if _, err := a.Wrap("<!--just a comment-->"); err == nil {
		t.Fatal("expected no-element error")
	}
}

func TestWrapDetached(t *testing.T) {
	// Wrapping a node with no parent still nests it inside the wrapper.
	d := NewDocument()
	n := d.NewElement("lonely")
	w, err := n.Wrap("<box/>")
	if err != nil {
		t.Fatal(err)
	}
	if n.Parent() != w || w.ToXML() != "<box>\n  <lonely/>\n</box>" {
		t.Fatalf("detached wrap = %q", w.ToXML())
	}
}
