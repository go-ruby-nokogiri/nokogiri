// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

import "testing"

const scrapeHTML = `<!DOCTYPE html>
<html><body>
  <div id="main" class="wrap">
    <ul class="list">
      <li class="item first" data-x="1">Alpha</li>
      <li class="item" data-x="2">Beta</li>
      <li class="item last" data-x="3">Gamma</li>
    </ul>
    <a href="https://example.com/a">A</a>
    <a href="http://example.org/b">B</a>
    <p>Para <span>inner</span> tail</p>
  </div>
</body></html>`

func mustHTML(t *testing.T, s string) *Document {
	t.Helper()
	d, err := HTML(s)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestCSSBasic(t *testing.T) {
	d := mustHTML(t, scrapeHTML)
	cases := []struct {
		sel  string
		want int
	}{
		{"li", 3},
		{"li.item", 3},
		{".first", 1},
		{"ul.list li", 3},
		{"#main", 1},
		{"div#main ul li.last", 1},
		{"a[href]", 2},
		{`a[href^="https"]`, 1},
		{`a[href$="/b"]`, 1},
		{`a[href*="example"]`, 2},
		{`li[data-x="2"]`, 1},
		{"ul > li", 3},
		{"li:first-child", 1},
		{"li:last-child", 1},
		{"li:nth-child(2)", 1},
		{"li:nth-child(odd)", 2},
		{"li:not(.first)", 2},
		{"span, a", 3},
		{"p > span", 1},
	}
	for _, c := range cases {
		set, err := d.CSS(c.sel)
		if err != nil {
			t.Errorf("%q: error %v", c.sel, err)
			continue
		}
		if set.Length() != c.want {
			t.Errorf("%q: got %d want %d", c.sel, set.Length(), c.want)
		}
	}
}

func TestCSSText(t *testing.T) {
	d := mustHTML(t, scrapeHTML)
	n, err := d.AtCSS("li.first")
	if err != nil || n == nil {
		t.Fatalf("at_css: %v %v", n, err)
	}
	if n.Text() != "Alpha" {
		t.Errorf("text = %q", n.Text())
	}
	if v := n.Attribute("data-x"); v != "1" {
		t.Errorf("data-x = %q", v)
	}
}

func TestCSSAdjacentSibling(t *testing.T) {
	d := mustHTML(t, scrapeHTML)
	set, err := d.CSS("li.first + li")
	if err != nil {
		t.Fatal(err)
	}
	if set.Length() != 1 || set.First().Text() != "Beta" {
		t.Fatalf("adjacent got %d %q", set.Length(), set.Text())
	}
}

func TestCSSGeneralSibling(t *testing.T) {
	d := mustHTML(t, scrapeHTML)
	set, err := d.CSS("li.first ~ li")
	if err != nil {
		t.Fatal(err)
	}
	if set.Length() != 2 {
		t.Fatalf("general sibling got %d", set.Length())
	}
}

const xmlDoc = `<catalog>
  <book id="b1" genre="fiction"><title>Alpha</title><price>10</price></book>
  <book id="b2" genre="tech"><title>Beta</title><price>20</price></book>
  <book id="b3" genre="fiction"><title>Gamma</title><price>30</price></book>
</catalog>`

func TestXPathBasic(t *testing.T) {
	d, err := XML(xmlDoc)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		xp   string
		want int
	}{
		{"//book", 3},
		{"//book[@genre='fiction']", 2},
		{"/catalog/book", 3},
		{"//book[2]", 1},
		{"//book[price>15]", 2},
		{"//title", 3},
		{"//book[last()]", 1},
		{"//book[position()=1]", 1},
		{"//*[@id]", 3},
		{"//book[@genre='fiction']/title", 2},
	}
	for _, c := range cases {
		set, err := d.XPath(c.xp)
		if err != nil {
			t.Errorf("%q: error %v", c.xp, err)
			continue
		}
		if set.Length() != c.want {
			t.Errorf("%q: got %d want %d", c.xp, set.Length(), c.want)
		}
	}
}

func TestXPathFunctions(t *testing.T) {
	d, err := XML(xmlDoc)
	if err != nil {
		t.Fatal(err)
	}
	v, err := d.Node.EvalXPath("count(//book)", nil)
	if err != nil {
		t.Fatal(err)
	}
	if f, ok := v.(float64); !ok || f != 3 {
		t.Errorf("count = %v", v)
	}
	v, err = d.Node.EvalXPath("sum(//price)", nil)
	if err != nil {
		t.Fatal(err)
	}
	if f := v.(float64); f != 60 {
		t.Errorf("sum = %v", f)
	}
	v, err = d.Node.EvalXPath("string(//title[1])", nil)
	if err != nil {
		t.Fatal(err)
	}
	if s := v.(string); s != "Alpha" {
		t.Errorf("string = %q", s)
	}
}

func TestXPathAtAndAttrs(t *testing.T) {
	d, err := XML(xmlDoc)
	if err != nil {
		t.Fatal(err)
	}
	n, err := d.AtXPath("//book[@id='b2']/title")
	if err != nil || n == nil {
		t.Fatalf("at_xpath: %v %v", n, err)
	}
	if n.Text() != "Beta" {
		t.Errorf("title = %q", n.Text())
	}
	set, err := d.XPath("//book/@genre")
	if err != nil {
		t.Fatal(err)
	}
	if set.Length() != 3 {
		t.Errorf("attr axis got %d", set.Length())
	}
	if set.First().Text() != "fiction" {
		t.Errorf("attr text = %q", set.First().Text())
	}
}
