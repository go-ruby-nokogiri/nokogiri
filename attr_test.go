// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

import "testing"

func TestAttributes(t *testing.T) {
	d, _ := XML(`<r a="1" ns:b="2"/>`)
	root := d.Root()
	if v, ok := root.Get("a"); !ok || v != "1" {
		t.Fatal("get a")
	}
	if v, ok := root.Get("ns:b"); !ok || v != "2" {
		t.Fatal("get qualified")
	}
	// bare local name matches qualified attr
	if v, ok := root.Get("b"); !ok || v != "2" {
		t.Fatal("get local")
	}
	if _, ok := root.Get("missing"); ok {
		t.Fatal("missing should be absent")
	}
	if root.Attribute("a") != "1" || root.Attribute("missing") != "" {
		t.Fatal("Attribute")
	}
	if !root.HasAttribute("a") || root.HasAttribute("nope") {
		t.Fatal("HasAttribute")
	}
}

func TestSetRemoveAttribute(t *testing.T) {
	d, _ := XML(`<r a="1"/>`)
	root := d.Root()
	root.SetAttribute("a", "9")   // update existing
	root.SetAttribute("c", "3")   // create new
	root.SetAttribute("p:q", "4") // qualified new
	if root.Attribute("a") != "9" || root.Attribute("c") != "3" || root.Attribute("p:q") != "4" {
		t.Fatalf("set: %q", root.ToXML())
	}
	root.RemoveAttribute("c")
	if root.HasAttribute("c") {
		t.Fatal("remove failed")
	}
	root.RemoveAttribute("nope") // no-op
	// qualified remove
	root.RemoveAttribute("p:q")
	if root.HasAttribute("p:q") {
		t.Fatal("qualified remove failed")
	}
}

func TestAttributesMap(t *testing.T) {
	d, _ := XML(`<r a="1" x:b="2"/>`)
	m := d.Root().Attributes()
	if m["a"].Value != "1" || m["x:b"].Value != "2" {
		t.Fatalf("attributes map: %v", m)
	}
}

func TestNamespaceDeclarations(t *testing.T) {
	d, _ := XML(`<r xmlns:x="urn:x" xmlns="urn:default"/>`)
	decls := d.Root().NamespaceDeclarations()
	if len(decls) != 2 {
		t.Fatalf("decls = %d", len(decls))
	}
}
