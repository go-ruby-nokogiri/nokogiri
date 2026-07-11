// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

import "testing"

func TestNamespacesInScope(t *testing.T) {
	d, _ := XML(`<root xmlns="urn:def" xmlns:a="urn:a"><a:child xmlns:b="urn:b"><a:g/></a:child></root>`)
	g, _ := d.Node.AtXPath("//a:g", map[string]string{"a": "urn:a"})
	ns := g.Namespaces()
	want := map[string]string{"xmlns": "urn:def", "xmlns:a": "urn:a", "xmlns:b": "urn:b"}
	if len(ns) != len(want) {
		t.Fatalf("g namespaces = %v", ns)
	}
	for k, v := range want {
		if ns[k] != v {
			t.Fatalf("g namespaces[%q] = %q, want %q", k, ns[k], v)
		}
	}
	root := d.Root().Namespaces()
	if root["xmlns"] != "urn:def" || root["xmlns:a"] != "urn:a" || len(root) != 2 {
		t.Fatalf("root namespaces = %v", root)
	}
}

func TestNamespacesNearestWins(t *testing.T) {
	// The nearer redeclaration of prefix "a" must shadow the outer one.
	d, _ := XML(`<r xmlns:a="urn:outer"><m xmlns:a="urn:inner"><a:x/></m></r>`)
	x, _ := d.Node.AtXPath("//*[local-name()='x']", nil)
	if got := x.Namespaces()["xmlns:a"]; got != "urn:inner" {
		t.Fatalf("nearest ns = %q, want urn:inner", got)
	}
}

func TestNamespaceObject(t *testing.T) {
	d, _ := XML(`<root xmlns:a="urn:a"><a:g/><plain/></root>`)
	g, _ := d.Node.AtXPath("//a:g", map[string]string{"a": "urn:a"})
	ns := g.Namespace()
	if ns == nil || ns.Prefix != "a" || ns.URI != "urn:a" {
		t.Fatalf("g.Namespace = %+v", ns)
	}
	plain, _ := d.Node.AtXPath("//plain", nil)
	if plain.Namespace() != nil {
		t.Fatalf("plain.Namespace = %+v, want nil", plain.Namespace())
	}
}

func TestAddNamespaceNew(t *testing.T) {
	d, _ := XML(`<r/>`)
	r := d.Root()
	got := r.AddNamespace("x", "urn:x")
	if got.Prefix != "x" || got.URI != "urn:x" {
		t.Fatalf("AddNamespace returned %+v", got)
	}
	if r.ToXML() != `<r xmlns:x="urn:x"/>` {
		t.Fatalf("after add = %q", r.ToXML())
	}
	if r.Namespaces()["xmlns:x"] != "urn:x" {
		t.Fatalf("namespaces after add = %v", r.Namespaces())
	}
}

func TestAddNamespaceDefaultAndUpdate(t *testing.T) {
	d, _ := XML(`<r xmlns:x="urn:old"><x:c/></r>`)
	r := d.Root()
	// Redeclaring an existing prefix updates its URI in place and re-resolves.
	r.AddNamespace("x", "urn:new")
	c := r.FirstChild()
	if c.NsURI != "urn:new" {
		t.Fatalf("child NsURI after update = %q, want urn:new", c.NsURI)
	}
	if got := r.ToXML(); got != "<r xmlns:x=\"urn:new\">\n  <x:c/>\n</r>" {
		t.Fatalf("update serialize = %q", got)
	}
	// Adding a default namespace on a nested element resolves for that element.
	inner := d.NewElement("d")
	c.AddChild(inner)
	inner.AddNamespace("", "urn:default")
	if inner.NsURI != "urn:default" {
		t.Fatalf("inner default NsURI = %q", inner.NsURI)
	}
}
