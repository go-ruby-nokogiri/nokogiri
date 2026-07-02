// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

import "testing"

func TestParsePrimaryUnexpectedToken(t *testing.T) {
	d, _ := XML(`<r><a/></r>`)
	// a comma cannot begin a primary expression
	if _, err := d.XPath("//a[,]"); err == nil {
		t.Error("comma primary")
	}
	// a closing bracket cannot begin a primary
	if _, err := d.XPath("//a[]"); err == nil {
		t.Error("empty predicate")
	}
}

func TestParseMalformedNumber(t *testing.T) {
	d, _ := XML(`<r/>`)
	// the lexer accepts a run of digits and dots; ParseFloat then rejects it
	if _, err := d.XPath("1.2.3"); err == nil {
		t.Error("malformed number")
	}
}

func TestStartsStepFalse(t *testing.T) {
	d, _ := XML(`<r><a/></r>`)
	// "/" followed by a token that cannot start a step (a literal) is an error
	if _, err := d.XPath("/'x'"); err == nil {
		t.Error("root then non-step")
	}
}

func TestSubstringEmptyRange(t *testing.T) {
	d, _ := XML(`<r/>`)
	// hi <= lo -> empty string
	if evalStr(t, d, "substring('abc', 5, 1)") != "" {
		t.Error("substring hi<=lo")
	}
	if evalStr(t, d, "substring('abc', 2, 0)") != "" {
		t.Error("substring zero len")
	}
}

func TestAttributeAxisUnionDocOrder(t *testing.T) {
	d, _ := XML(`<r a="1" b="2"><c/></r>`)
	// union of two attribute node-sets exercises docOrder's parent-based fallback
	set, _ := d.XPath("//r/@a | //r/@b")
	if set.Length() != 2 {
		t.Fatalf("attr union: %d", set.Length())
	}
}

func TestParentAxisFromElement(t *testing.T) {
	d, _ := XML(`<r><child><grand/></child></r>`)
	// parent axis from a non-root element returns its element parent
	set, _ := d.XPath("//grand/parent::child")
	if set.Length() != 1 {
		t.Fatalf("parent axis: %d", set.Length())
	}
}

func TestNodeSetCompareScalarStringNoMatch(t *testing.T) {
	d, _ := XML(`<r><a>x</a><a>y</a></r>`)
	// nodeset = string where no member matches -> false (default branch)
	if evalBool(t, d, "//a = 'z'") {
		t.Error("nodeset string no match")
	}
	// but a match returns true
	if !evalBool(t, d, "//a = 'y'") {
		t.Error("nodeset string match")
	}
}

func TestRelCmpAllOps(t *testing.T) {
	d, _ := XML(`<r><n>5</n></r>`)
	// exercise each relCmp arm against a node-set (forces the per-op switch)
	if !evalBool(t, d, "//n > 4") {
		t.Error(">")
	}
	if !evalBool(t, d, "//n < 6") {
		t.Error("<")
	}
	if !evalBool(t, d, "//n >= 5") {
		t.Error(">=")
	}
	if !evalBool(t, d, "//n <= 5") {
		t.Error("<=")
	}
}
