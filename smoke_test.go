// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

import "testing"

func TestHTMLSmoke(t *testing.T) {
	doc, err := HTML(`<html><body><p class="a">Hello</p></body></html>`)
	if err != nil {
		t.Fatal(err)
	}
	root := doc.Root()
	if root == nil || root.Name != "html" {
		t.Fatalf("root = %v", root)
	}
	if got := doc.Text(); got != "Hello" {
		t.Fatalf("text = %q", got)
	}
}

func TestXMLSmoke(t *testing.T) {
	doc, err := XML(`<r><a x="1">t</a></r>`)
	if err != nil {
		t.Fatal(err)
	}
	root := doc.Root()
	if root.Name != "r" {
		t.Fatalf("root = %q", root.Name)
	}
	a := root.FirstChild()
	if v := a.Attribute("x"); v != "1" {
		t.Fatalf("x = %q", v)
	}
	if got := root.ToXML(); got != `<r><a x="1">t</a></r>` {
		t.Fatalf("toxml = %q", got)
	}
}
