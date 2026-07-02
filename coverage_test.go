// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

import "testing"

func TestNodeSetMethods(t *testing.T) {
	d, _ := XML(`<r><a>1</a><a>2</a><a>3</a></r>`)
	set, _ := d.XPath("//a")
	if set.Empty() {
		t.Fatal("not empty")
	}
	empty, _ := d.XPath("//zzz")
	if !empty.Empty() {
		t.Fatal("empty")
	}
	if set.At(0).Text() != "1" || set.At(-1).Text() != "3" {
		t.Fatal("At positive/negative")
	}
	if set.At(99) != nil || set.At(-99) != nil {
		t.Fatal("At out of range")
	}
	if set.First().Text() != "1" || set.Last().Text() != "3" {
		t.Fatal("First/Last")
	}
	if empty.First() != nil || empty.Last() != nil {
		t.Fatal("First/Last empty")
	}
	if len(set.Nodes()) != 3 {
		t.Fatal("Nodes")
	}
	var count int
	set.Each(func(*Node) { count++ })
	if count != 3 {
		t.Fatal("Each")
	}
	if set.Text() != "123" {
		t.Fatalf("NodeSet.Text = %q", set.Text())
	}
}

func TestNodeSetQueries(t *testing.T) {
	d, _ := XML(`<r><sec><x id="1"/></sec><sec><x id="2"/></sec></r>`)
	secs, _ := d.CSS("sec")
	xs, err := secs.CSS("x")
	if err != nil || xs.Length() != 2 {
		t.Fatalf("nodeset css: %v %d", err, xs.Length())
	}
	xs2, err := secs.XPath(".//x")
	if err != nil || xs2.Length() != 2 {
		t.Fatalf("nodeset xpath: %v %d", err, xs2.Length())
	}
}

func TestNodeQueryNsMap(t *testing.T) {
	d, _ := XML(`<root xmlns:a="urn:a"><a:item>x</a:item></root>`)
	ns := map[string]string{"a": "urn:a"}
	set, err := d.Node.XPath("//a:item", ns)
	if err != nil || set.Length() != 1 {
		t.Fatalf("ns xpath: %v %d", err, set.Length())
	}
	n, _ := d.Node.AtXPath("//a:item", ns)
	if n == nil || n.Text() != "x" {
		t.Fatal("at_xpath ns")
	}
	// wrong ns prefix falls back to literal prefix match
	set, _ = d.Node.XPath("//a:item", nil)
	if set.Length() != 1 {
		t.Fatal("literal prefix fallback")
	}
	// at_css
	got, _ := d.Node.AtCSS("item", nil)
	if got == nil {
		t.Fatal("at_css")
	}
}

func TestXPathScalarResults(t *testing.T) {
	d, _ := XML(`<r><a>1</a></r>`)
	// EvalXPath returns *NodeSet for node-set results
	v, _ := d.Node.EvalXPath("//a", nil)
	if _, ok := v.(*NodeSet); !ok {
		t.Fatalf("expected NodeSet, got %T", v)
	}
	// XPath on a scalar-producing expr returns an empty set gracefully
	set, err := d.XPath("count(//a)")
	if err != nil || !set.Empty() {
		t.Fatalf("scalar via XPath: %v %v", err, set)
	}
	// at_xpath on scalar
	n, _ := d.AtXPath("string(//a)")
	if n != nil {
		t.Fatal("at_xpath scalar nil")
	}
}

func TestHTMLFragmentParse(t *testing.T) {
	d, err := HTMLFragment(`<p>one</p><p>two</p><!--c-->`)
	if err != nil {
		t.Fatal(err)
	}
	set, _ := d.CSS("p")
	if set.Length() != 2 {
		t.Fatalf("fragment p: %d", set.Length())
	}
	if d.InnerHTML() != `<p>one</p><p>two</p><!--c-->` {
		t.Fatalf("fragment html: %q", d.InnerHTML())
	}
}

func TestXPathErrors(t *testing.T) {
	d, _ := XML(`<r/>`)
	bad := []string{
		"///",
		"1 +",
		"@",
		"(",
		"'unterminated",
		"unknownfunc()",
		"count()",
		"$undefined_var",
		"1 2",
		"foo::bar",
		"1 @ 2",
	}
	for _, xp := range bad {
		if _, err := d.XPath(xp); err == nil {
			t.Errorf("%q: expected error", xp)
		}
	}
}

func TestXPathVariables(t *testing.T) {
	d, _ := XML(`<r><a id="x"/><a id="y"/></r>`)
	vars := map[string]xpValue{"want": "x"}
	v, err := evalXPath("//a[@id=$want]", &d.Node, vars, nil)
	if err != nil {
		t.Fatal(err)
	}
	if nl, ok := v.(*nodeList); !ok || len(nl.nodes) != 1 {
		t.Fatalf("var: %v", v)
	}
}

func TestXPathAbsoluteRoot(t *testing.T) {
	d, _ := XML(`<r><a/></r>`)
	// "/" selects the root node-set; "/r" the root element
	set, _ := d.XPath("/r/a")
	if set.Length() != 1 {
		t.Fatalf("absolute: %d", set.Length())
	}
	// filter expr with predicate then path
	set, _ = d.XPath("(//a)[1]")
	if set.Length() != 1 {
		t.Fatalf("filter predicate: %d", set.Length())
	}
}

func TestXPathTextAndComment(t *testing.T) {
	d, _ := XML(`<r>hi<!--c--><a/></r>`)
	set, _ := d.XPath("//text()")
	if set.Length() != 1 || set.First().Text() != "hi" {
		t.Fatalf("text(): %d", set.Length())
	}
	set, _ = d.XPath("//comment()")
	if set.Length() != 1 {
		t.Fatalf("comment(): %d", set.Length())
	}
	set, _ = d.XPath("//node()")
	if set.Length() < 3 {
		t.Fatalf("node(): %d", set.Length())
	}
}

func TestXPathProcessingInstruction(t *testing.T) {
	d, _ := XML(`<r><?php echo?><?other x?></r>`)
	set, _ := d.XPath("//processing-instruction()")
	if set.Length() != 2 {
		t.Fatalf("pi(): %d", set.Length())
	}
	set, _ = d.XPath("//processing-instruction('php')")
	if set.Length() != 1 {
		t.Fatalf("pi(php): %d", set.Length())
	}
}

func TestXMLErrors(t *testing.T) {
	bad := []string{
		`<r><a></r>`,      // mismatched
		`<r>`,             // unclosed
		`</r>`,            // stray end
		`<r a="unclosed>`, // malformed
	}
	for _, s := range bad {
		if _, err := XML(s); err == nil {
			t.Errorf("%q: expected parse error", s)
		}
	}
}

func TestXMLDirective(t *testing.T) {
	d, err := XML(`<!DOCTYPE r><r/>`)
	if err != nil {
		t.Fatal(err)
	}
	// doctype node precedes the root element
	if d.FirstChild().Type != DoctypeNode || d.FirstChild().Name != "DOCTYPE" {
		t.Fatalf("doctype: %v", d.FirstChild())
	}
}

func TestSubstringEdgeCases(t *testing.T) {
	d, _ := XML(`<r/>`)
	if evalStr(t, d, "substring('hello', 0)") != "hello" {
		t.Error("substring 0")
	}
	if evalStr(t, d, "substring('hello', -2, 4)") != "h" {
		t.Errorf("substring negative: %q", evalStr(t, d, "substring('hello', -2, 4)"))
	}
	if evalStr(t, d, "substring('hello', 10)") != "" {
		t.Error("substring past end")
	}
	if evalStr(t, d, "substring('hello', 1, 100)") != "hello" {
		t.Error("substring long len")
	}
}

func TestEscapeSpecial(t *testing.T) {
	n := &Node{Type: TextNode, content: "plain text no specials"}
	if n.ToXML() != "plain text no specials" {
		t.Fatal("plain text")
	}
}

func TestUnquoteCSSNoQuotes(t *testing.T) {
	// bare (unquoted) attribute value
	d, _ := HTML(`<a data-n=5>x</a>`)
	set, err := d.CSS(`a[data-n=5]`)
	if err != nil || set.Length() != 1 {
		t.Fatalf("bare attr value: %v %d", err, set.Length())
	}
}

func TestParseAnBForms(t *testing.T) {
	d, _ := HTML(`<ul><li>1</li><li>2</li><li>3</li></ul>`)
	// "+n" and "-n" coefficient forms, bare integer
	cases := []struct {
		sel  string
		want int
	}{
		{"li:nth-child(n)", 3},
		{"li:nth-child(1)", 1},
		{"li:nth-child(0n+2)", 1},
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
	// invalid An+B forms
	for _, bad := range []string{"li:nth-child()", "li:nth-child(2x)", "li:nth-child(n+z)"} {
		if _, err := d.CSS(bad); err == nil {
			t.Errorf("%q: expected error", bad)
		}
	}
}
