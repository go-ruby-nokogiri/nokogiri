// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

import "testing"

const badXP = "1 +"     // invalid XPath
const badCSS = "div:@(" // invalid CSS

func TestPublicErrorPaths(t *testing.T) {
	d, _ := XML(`<r><a/></r>`)
	root := d.Root()

	// Node.XPath / AtXPath / EvalXPath error
	if _, err := root.XPath(badXP, nil); err == nil {
		t.Error("Node.XPath")
	}
	if _, err := root.AtXPath(badXP, nil); err == nil {
		t.Error("Node.AtXPath")
	}
	if _, err := root.EvalXPath(badXP, nil); err == nil {
		t.Error("Node.EvalXPath")
	}
	// Node.CSS / AtCSS error
	if _, err := root.CSS(badCSS, nil); err == nil {
		t.Error("Node.CSS")
	}
	if _, err := root.AtCSS(badCSS, nil); err == nil {
		t.Error("Node.AtCSS")
	}
	// Document wrappers error
	if _, err := d.CSS(badCSS); err == nil {
		t.Error("Document.CSS")
	}
	if _, err := d.AtCSS(badCSS); err == nil {
		t.Error("Document.AtCSS")
	}
	if _, err := d.XPath(badXP); err == nil {
		t.Error("Document.XPath")
	}
	if _, err := d.AtXPath(badXP); err == nil {
		t.Error("Document.AtXPath")
	}
}

func TestNodeSetQueryErrorPaths(t *testing.T) {
	d, _ := XML(`<r><a/><a/></r>`)
	set, _ := d.XPath("//a")
	if _, err := set.CSS(badCSS); err == nil {
		t.Error("NodeSet.CSS error")
	}
	if _, err := set.XPath(badXP); err == nil {
		t.Error("NodeSet.XPath error")
	}
}

func TestNodeSetQueryErrorMidIteration(t *testing.T) {
	// The second member also errors; ensures the loop error return is exercised
	// with a non-empty prefix.
	d, _ := XML(`<r><a/></r>`)
	set := &NodeSet{nodes: []*Node{d.Root(), d.Root()}}
	if _, err := set.XPath("@"); err == nil {
		t.Error("mid-iteration xpath error")
	}
	if _, err := set.CSS(":"); err == nil {
		t.Error("mid-iteration css error")
	}
}

func TestParsePrimaryUnexpected(t *testing.T) {
	d, _ := XML(`<r/>`)
	// A bare "]" cannot start a primary.
	if _, err := d.XPath("//a[]"); err == nil {
		t.Error("empty predicate")
	}
	// operator with nothing after
	if _, err := d.XPath("//a[1 =]"); err == nil {
		t.Error("dangling equality")
	}
}
