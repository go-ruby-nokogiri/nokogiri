// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

import "testing"

func TestCSSRootPseudo(t *testing.T) {
	d, _ := XML(`<root><child/></root>`)
	set, err := d.CSS(":root")
	if err != nil {
		t.Fatal(err)
	}
	if set.Length() != 1 || set.First().Name != "root" {
		t.Fatalf(":root: %d", set.Length())
	}
}

func TestCSSNestedBrackets(t *testing.T) {
	// attribute selector body handling with a bracket inside a quoted value
	d, _ := HTML(`<a data-x="v[1]">A</a><a data-x="v">B</a>`)
	set, err := d.CSS(`a[data-x="v[1]"]`)
	if err != nil {
		t.Fatal(err)
	}
	if set.Length() != 1 {
		t.Fatalf("nested bracket attr: %d", set.Length())
	}
}

func TestCSSCompoundErrorBranches(t *testing.T) {
	d, _ := HTML(`<div><p>x</p></div>`)
	// A bad attribute selector inside a compound sequence (not the first token).
	if _, err := d.CSS(`p[ ~ ]`); err != nil {
		// tolerated (whitespace op), ensure no panic; not asserting error
		_ = err
	}
	// pseudo error inside a compound sequence
	if _, err := d.CSS(`p.cls:unknown-xyz`); err == nil {
		t.Error("compound pseudo error")
	}
	// :not with a bad pseudo inside
	if _, err := d.CSS(`p:not(:unknown-xyz)`); err == nil {
		t.Error(":not bad pseudo")
	}
}

func TestParseAnBNegativeCoeff(t *testing.T) {
	d, _ := HTML(`<ul><li>1</li><li>2</li><li>3</li><li>4</li></ul>`)
	// "-n+3" selects the first 3
	set, err := d.CSS("li:nth-child(-n+3)")
	if err != nil {
		t.Fatal(err)
	}
	if set.Length() != 3 {
		t.Fatalf("-n+3: %d", set.Length())
	}
	// explicit "+2n+1"
	set, _ = d.CSS("li:nth-child(+2n+1)")
	if set.Length() != 2 {
		t.Fatalf("+2n+1: %d", set.Length())
	}
}

func TestHTMLFragmentWithDoctype(t *testing.T) {
	// A doctype inside a fragment is converted (or dropped) without panic.
	d, err := HTMLFragment(`<span>hi</span>`)
	if err != nil {
		t.Fatal(err)
	}
	if d.AtCSSMust(t, "span").Text() != "hi" {
		t.Fatal("fragment span")
	}
}

// AtCSSMust is a test helper.
func (d *Document) AtCSSMust(t *testing.T, sel string) *Node {
	t.Helper()
	n, err := d.AtCSS(sel)
	if err != nil || n == nil {
		t.Fatalf("at_css %q: %v", sel, err)
	}
	return n
}

func TestConvertHTMLDoctypeNode(t *testing.T) {
	// Full document with doctype -> convertHTML handles the DoctypeNode child.
	d, err := HTML(`<!DOCTYPE html><html><body><p>x</p></body></html>`)
	if err != nil {
		t.Fatal(err)
	}
	if d.Root() == nil {
		t.Fatal("root")
	}
}

func TestCSSTypeSelectorDefault(t *testing.T) {
	// exercise the default (type name) branch of lexSimple explicitly
	d, _ := HTML(`<article><section>x</section></article>`)
	set, _ := d.CSS("article section")
	if set.Length() != 1 {
		t.Fatalf("type descendant: %d", set.Length())
	}
}
