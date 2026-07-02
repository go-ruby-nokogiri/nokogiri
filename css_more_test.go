// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

import "testing"

const cssMoreHTML = `<div>
  <ul>
    <li lang="en-US">a</li>
    <li lang="en">b</li>
    <li lang="fr">c</li>
    <li>d</li>
    <li></li>
  </ul>
  <input type="text" required>
  <p class="x y z">p1</p>
  <p class='has "quote"'>p2</p>
  <span title="it's here">s</span>
</div>`

func cssDoc(t *testing.T) *Document {
	t.Helper()
	d, err := HTML(cssMoreHTML)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestCSSAttrOperators(t *testing.T) {
	d := cssDoc(t)
	cases := []struct {
		sel  string
		want int
	}{
		{`[lang|="en"]`, 2},   // en-US and en
		{`li[lang~="en"]`, 1}, // whitespace list contains en (only "en")
		{`input[required]`, 1},
		{`input[type="text"]`, 1},
		{`p.x`, 1},
		{`p.y.z`, 1},
		{`*[title]`, 1},
		{":empty", 3},          // the implied empty <head>, the empty <li>, and void <input>
		{"li:only-child", 0},   // none
		{"li:nth-child(3)", 1}, // c
		{"li:nth-child(2n)", 2},
		{"li:nth-last-child(1)", 1},
		{"li:not([lang])", 2}, // d and empty
	}
	for _, c := range cases {
		set, err := d.CSS(c.sel)
		if err != nil {
			t.Errorf("%q: %v", c.sel, err)
			continue
		}
		if set.Length() != c.want {
			t.Errorf("%q: got %d want %d", c.sel, set.Length(), c.want)
		}
	}
}

func TestCSSQuotedValues(t *testing.T) {
	d := cssDoc(t)
	// value containing double quotes -> XPath uses single-quote literal
	set, err := d.CSS(`p[class='has "quote"']`)
	if err != nil {
		t.Fatal(err)
	}
	if set.Length() != 1 {
		t.Errorf("double-quote value: %d", set.Length())
	}
	// value containing single quote -> XPath must use concat()
	set, err = d.CSS(`span[title="it's here"]`)
	if err != nil {
		t.Fatal(err)
	}
	if set.Length() != 1 {
		t.Errorf("single-quote value: %d", set.Length())
	}
}

func TestCSSToXPathBothQuotes(t *testing.T) {
	// exercise xpStr concat() branch directly
	got := xpStr(`a'b"c`)
	// should be a concat expression
	if got == `'a'b"c'` {
		t.Fatal("should not be a naive literal")
	}
	d, _ := XML(`<r><a x="a'b&quot;c"/></r>`)
	set, err := d.XPath(`//a[@x=` + got + `]`)
	if err != nil {
		t.Fatal(err)
	}
	if set.Length() != 1 {
		t.Errorf("mixed-quote match: %d", set.Length())
	}
}

func TestCSSNthVariants(t *testing.T) {
	d, _ := HTML(`<ul><li>1</li><li>2</li><li>3</li><li>4</li><li>5</li></ul>`)
	cases := []struct {
		sel  string
		want int
	}{
		{"li:nth-child(odd)", 3},
		{"li:nth-child(even)", 2},
		{"li:nth-child(2n+1)", 3},
		{"li:nth-child(3n)", 1},
		{"li:nth-child(-n+2)", 2},
		{"li:first-of-type", 1},
	}
	for _, c := range cases {
		set, err := d.CSS(c.sel)
		if err != nil {
			t.Errorf("%q: %v", c.sel, err)
			continue
		}
		if set.Length() != c.want {
			t.Errorf("%q: got %d want %d", c.sel, set.Length(), c.want)
		}
	}
}

func TestCSSErrors(t *testing.T) {
	d := cssDoc(t)
	bad := []string{
		"",
		"div,,p",
		"li:unknown-pseudo",
		"li:nth-child(bad)",
		"#",
		".",
		":",
		"[unterminated",
		"li:not(a > b)",
	}
	for _, sel := range bad {
		if _, err := d.CSS(sel); err == nil {
			t.Errorf("%q: expected error", sel)
		}
	}
}
