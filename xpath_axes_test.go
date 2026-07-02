// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

import "testing"

const axesXML = `<root>
  <section id="s1"><h>H1</h><p id="p1">one</p><p id="p2">two</p></section>
  <section id="s2"><p id="p3">three</p></section>
</root>`

func axesDoc(t *testing.T) *Document {
	t.Helper()
	d, err := XML(axesXML)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestAllAxes(t *testing.T) {
	d := axesDoc(t)
	cases := []struct {
		xp   string
		want int
	}{
		{"//p[@id='p1']/parent::section", 1},
		{"//p[@id='p1']/ancestor::root", 1},
		{"//p[@id='p1']/ancestor::*", 2},
		{"//p[@id='p1']/ancestor-or-self::*", 3},
		{"//p[@id='p1']/self::p", 1},
		{"//h/following-sibling::p", 2},
		{"//p[@id='p2']/preceding-sibling::p", 1},
		{"//p[@id='p2']/preceding-sibling::h", 1},
		{"//section[@id='s1']/child::p", 2},
		{"//section[@id='s1']/descendant::p", 2},
		{"//section[@id='s1']/descendant-or-self::section", 1},
		{"//h/following::p", 3},
		{"//p[@id='p3']/preceding::p", 2},
		{"//p[@id='p3']/preceding::h", 1},
		{"//section[@id='s1']/attribute::id", 1},
		{"//p/../..", 1},
		{"//p/.", 3},
	}
	for _, c := range cases {
		set, err := d.XPath(c.xp)
		if err != nil {
			t.Errorf("%q: %v", c.xp, err)
			continue
		}
		if set.Length() != c.want {
			t.Errorf("%q: got %d want %d", c.xp, set.Length(), c.want)
		}
	}
}

func TestXPathUnionAndArith(t *testing.T) {
	d := axesDoc(t)
	set, _ := d.XPath("//h | //p[@id='p3']")
	if set.Length() != 2 {
		t.Errorf("union: %d", set.Length())
	}
	v, _ := d.Node.EvalXPath("1 + 2 * 3 - 4 div 2 + 7 mod 3", nil)
	if v.(float64) != 6 {
		t.Errorf("arith: %v", v)
	}
	v, _ = d.Node.EvalXPath("-5 + 3", nil)
	if v.(float64) != -2 {
		t.Errorf("unary: %v", v)
	}
}

func TestXPathBooleanAndRelational(t *testing.T) {
	d := axesDoc(t)
	cases := []struct {
		xp   string
		want bool
	}{
		{"1 < 2 and 2 <= 2", true},
		{"3 > 4 or 1 = 1", true},
		{"2 != 3", true},
		{"count(//p) >= 3", true},
		{"count(//p) < 3", false},
		{"'a' = 'a'", true},
		{"true() and not(false())", true},
		{"//p[@id='p1'] = 'one'", true},
		{"//p = 'three'", true},
		{"//p != 'nope'", true},
		{"//nonexistent = 'x'", false},
		{"boolean(//p)", true},
		{"boolean(//zzz)", false},
	}
	for _, c := range cases {
		v, err := d.Node.EvalXPath(c.xp, nil)
		if err != nil {
			t.Errorf("%q: %v", c.xp, err)
			continue
		}
		if v.(bool) != c.want {
			t.Errorf("%q: got %v want %v", c.xp, v, c.want)
		}
	}
}

func TestXPathNodeComparisons(t *testing.T) {
	d := axesDoc(t)
	// node-set relational against node-set
	v, _ := d.Node.EvalXPath("//p[@id='p1'] < //p[@id='p2']", nil)
	if v.(bool) {
		t.Error("string 'one' < 'two' numerically is NaN, expect false")
	}
	// number node-set comparisons
	d2, _ := XML(`<r><n>5</n><n>10</n></r>`)
	v, _ = d2.Node.EvalXPath("//n > 7", nil)
	if !v.(bool) {
		t.Error("some n > 7")
	}
	v, _ = d2.Node.EvalXPath("//n >= 10", nil)
	if !v.(bool) {
		t.Error(">=")
	}
	v, _ = d2.Node.EvalXPath("//n <= 5", nil)
	if !v.(bool) {
		t.Error("<=")
	}
}
