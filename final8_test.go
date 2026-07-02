// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

import "testing"

func TestAbsoluteWildcardStep(t *testing.T) {
	d, _ := XML(`<r><a/></r>`)
	// "/*" -> "/" alone followed by a "*" step (startsStep sees an operator token)
	set, _ := d.XPath("/*")
	if set.Length() != 1 || set.First().Name != "r" {
		t.Fatalf("/*: %d", set.Length())
	}
	// "/." -> the "." self step after root
	set, _ = d.XPath("/.")
	if set.Length() != 1 {
		t.Fatalf("/.: %d", set.Length())
	}
	// "/@x" attribute step after root (operator start)
	d2, _ := XML(`<r x="1"/>`)
	set, _ = d2.XPath("/r")
	if set.Length() != 1 {
		t.Fatalf("/r: %d", set.Length())
	}
}

func TestParsePrimaryFailToken(t *testing.T) {
	d, _ := XML(`<r/>`)
	// "@" where a primary is expected (inside a parenthesised expression)
	if _, err := d.XPath("(@)"); err == nil {
		t.Error("expected primary error for @")
	}
	// a predicate whose expression is just an operator
	if _, err := d.XPath("//a[*=]"); err == nil {
		t.Error("expected error for dangling =")
	}
	// an empty parenthesised group reaches parsePrimary with a ")" token
	if _, err := d.XPath("()"); err == nil {
		t.Error("expected error for empty group")
	}
	// a parenthesised expression whose body starts with a stray operator
	if _, err := d.XPath("(,)"); err == nil {
		t.Error("expected error for comma in group")
	}
}

func TestNodeSetNumberComparisonNoMatch(t *testing.T) {
	d, _ := XML(`<r><n>5</n><n>10</n></r>`)
	// nodeset = number with no matching member -> false (float default branch)
	if evalBool(t, d, "//n = 999") {
		t.Error("nodeset = nonmatching number")
	}
	// nodeset != number where all differ -> true
	if !evalBool(t, d, "//n != 999") {
		t.Error("nodeset != number")
	}
}

func TestSubstringStartBelowOne(t *testing.T) {
	d, _ := XML(`<r/>`)
	// start below 1 with a length that reaches into the string
	if evalStr(t, d, "substring('12345', -1, 4)") != "12" {
		t.Errorf("substring below 1: %q", evalStr(t, d, "substring('12345', -1, 4)"))
	}
}
