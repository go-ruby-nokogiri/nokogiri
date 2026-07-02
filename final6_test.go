// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

import "testing"

func TestNthOfType(t *testing.T) {
	d, _ := HTML(`<div><h1>t</h1><p>1</p><span>s</span><p>2</p><p>3</p></div>`)
	// second <p> among its siblings
	set, err := d.CSS("p:nth-of-type(2)")
	if err != nil {
		t.Fatal(err)
	}
	if set.Length() != 1 || set.First().Text() != "2" {
		t.Fatalf("nth-of-type(2): %d %q", set.Length(), set.Text())
	}
}

func TestParentAxisOfRoot(t *testing.T) {
	d, _ := XML(`<r/>`)
	// the root element's parent is the document node; the document node's parent
	// is nil, so parent::* from the document yields nothing.
	set, _ := d.Node.XPath("parent::*", nil)
	if set.Length() != 0 {
		t.Fatalf("parent of document: %d", set.Length())
	}
}

func TestNamespaceAxisEmpty(t *testing.T) {
	d, _ := XML(`<r xmlns:a="urn:a"><a:x/></r>`)
	set, _ := d.XPath("//r/namespace::*")
	if set.Length() != 0 {
		t.Fatalf("namespace axis: %d", set.Length())
	}
}

func TestAnBOverflow(t *testing.T) {
	d, _ := HTML(`<ul><li>1</li></ul>`)
	// a coefficient that overflows int makes parseAnB fail
	if _, err := d.CSS("li:nth-child(99999999999999999999n+1)"); err == nil {
		t.Error("expected overflow error")
	}
	// an offset that overflows int
	if _, err := d.CSS("li:nth-child(2n+99999999999999999999)"); err == nil {
		t.Error("expected offset overflow error")
	}
	// a bare integer that overflows
	if _, err := d.CSS("li:nth-child(99999999999999999999)"); err == nil {
		t.Error("expected bare overflow error")
	}
}

func TestPrecedingNodesReachN(t *testing.T) {
	// A tree where the target has preceding nodes AND the walk terminates at it.
	d, _ := XML(`<r><a/><b/><target/><c/></r>`)
	set, _ := d.XPath("//target/preceding::*")
	names := map[string]bool{}
	set.Each(func(n *Node) { names[n.Name] = true })
	if !names["a"] || !names["b"] || names["c"] {
		t.Fatalf("preceding: %v", names)
	}
}

func TestSubstringInfiniteLengthPastEnd(t *testing.T) {
	d, _ := XML(`<r/>`)
	// 2-arg substring whose start is within range returns to the end (hi capped)
	if evalStr(t, d, "substring('hello', 2)") != "ello" {
		t.Error("substring to end")
	}
}

func TestCSSPseudoNoArgEmit(t *testing.T) {
	// a pseudo with no parenthesized argument (exercises the no-arg emit path)
	d, _ := HTML(`<ul><li>a</li><li>b</li></ul>`)
	set, _ := d.CSS("li:first-child")
	if set.Length() != 1 {
		t.Fatalf("first-child: %d", set.Length())
	}
}
