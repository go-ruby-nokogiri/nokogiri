// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

import "testing"

func evalStr(t *testing.T, d *Document, xp string) string {
	t.Helper()
	v, err := d.Node.EvalXPath(xp, nil)
	if err != nil {
		t.Fatalf("%q: %v", xp, err)
	}
	s, ok := v.(string)
	if !ok {
		t.Fatalf("%q: not a string: %T", xp, v)
	}
	return s
}

func evalNum(t *testing.T, d *Document, xp string) float64 {
	t.Helper()
	v, err := d.Node.EvalXPath(xp, nil)
	if err != nil {
		t.Fatalf("%q: %v", xp, err)
	}
	return v.(float64)
}

func TestStringFunctions(t *testing.T) {
	d, _ := XML(`<r x="Hello World"><a>foo</a><b>bar</b></r>`)
	if s := evalStr(t, d, "concat('a', 'b', 'c')"); s != "abc" {
		t.Errorf("concat: %q", s)
	}
	if evalStr(t, d, "substring('hello', 2, 3)") != "ell" {
		t.Error("substring")
	}
	if evalStr(t, d, "substring('hello', 3)") != "llo" {
		t.Error("substring 2-arg")
	}
	if evalStr(t, d, "substring-before('a-b-c', '-')") != "a" {
		t.Error("substring-before")
	}
	if evalStr(t, d, "substring-after('a-b-c', '-')") != "b-c" {
		t.Error("substring-after")
	}
	if evalStr(t, d, "substring-before('abc', 'z')") != "" {
		t.Error("substring-before missing")
	}
	if evalStr(t, d, "substring-after('abc', 'z')") != "" {
		t.Error("substring-after missing")
	}
	if evalStr(t, d, "normalize-space('  a   b  ')") != "a b" {
		t.Error("normalize-space")
	}
	if evalStr(t, d, "translate('bar', 'abc', 'ABC')") != "BAr" {
		t.Error("translate")
	}
	if evalStr(t, d, "translate('hello', 'l', '')") != "heo" {
		t.Error("translate delete")
	}
	if !evalBool(t, d, "starts-with('hello', 'he')") {
		t.Error("starts-with")
	}
	if !evalBool(t, d, "contains('hello', 'ell')") {
		t.Error("contains")
	}
	if evalNum(t, d, "string-length('hello')") != 5 {
		t.Error("string-length")
	}
	if evalNum(t, d, "string-length(//r/@x)") != 11 {
		t.Error("string-length attr arg")
	}
	// string() with no arg uses context node
	sub, _ := d.AtXPath("//a")
	v, _ := sub.EvalXPath("string()", nil)
	if v.(string) != "foo" {
		t.Errorf("string() no arg: %v", v)
	}
	// normalize-space and string-length with no arg
	v, _ = sub.EvalXPath("normalize-space()", nil)
	if v.(string) != "foo" {
		t.Error("normalize-space no arg")
	}
	v, _ = sub.EvalXPath("string-length()", nil)
	if v.(float64) != 3 {
		t.Error("string-length no arg")
	}
}

func evalBool(t *testing.T, d *Document, xp string) bool {
	t.Helper()
	v, err := d.Node.EvalXPath(xp, nil)
	if err != nil {
		t.Fatalf("%q: %v", xp, err)
	}
	return v.(bool)
}

func TestNumberFunctions(t *testing.T) {
	d, _ := XML(`<r><n>1</n><n>2</n><n>3</n></r>`)
	if evalNum(t, d, "sum(//n)") != 6 {
		t.Error("sum")
	}
	if evalNum(t, d, "floor(3.7)") != 3 {
		t.Error("floor")
	}
	if evalNum(t, d, "ceiling(3.2)") != 4 {
		t.Error("ceiling")
	}
	if evalNum(t, d, "round(3.5)") != 4 {
		t.Error("round")
	}
	if evalNum(t, d, "number('42')") != 42 {
		t.Error("number")
	}
	if evalNum(t, d, "count(//n)") != 3 {
		t.Error("count")
	}
	// number() no arg
	sub, _ := d.AtXPath("//n[1]")
	v, _ := sub.EvalXPath("number()", nil)
	if v.(float64) != 1 {
		t.Error("number no arg")
	}
}

func TestNameFunctions(t *testing.T) {
	d, _ := XML(`<root xmlns:x="urn:x"><x:item id="i1"/></root>`)
	if evalStr(t, d, "local-name(//*[@id='i1'])") != "item" {
		t.Error("local-name")
	}
	if evalStr(t, d, "name(//*[@id='i1'])") != "x:item" {
		t.Error("name")
	}
	if evalStr(t, d, "namespace-uri(//*[@id='i1'])") != "urn:x" {
		t.Errorf("namespace-uri: %q", evalStr(t, d, "namespace-uri(//*[@id='i1'])"))
	}
	// no-arg forms use context node
	sub, _ := d.Node.AtXPath("//x:item", map[string]string{"x": "urn:x"})
	v, _ := sub.EvalXPath("local-name()", nil)
	if v.(string) != "item" {
		t.Error("local-name no arg")
	}
	v, _ = sub.EvalXPath("name()", nil)
	if v.(string) != "x:item" {
		t.Error("name no arg")
	}
	v, _ = sub.EvalXPath("namespace-uri()", nil)
	if v.(string) != "urn:x" {
		t.Error("namespace-uri no arg")
	}
	// empty node-set arg
	if evalStr(t, d, "local-name(//zzz)") != "" {
		t.Error("local-name empty")
	}
	if evalStr(t, d, "name(//zzz)") != "" {
		t.Error("name empty")
	}
	if evalStr(t, d, "namespace-uri(//zzz)") != "" {
		t.Error("namespace-uri empty")
	}
}

func TestIDAndLangFunctions(t *testing.T) {
	d, _ := XML(`<root><a id="x1">A</a><b id="x2" xml:lang="en">B</b><c xml:lang="fr-CA">C</c></root>`)
	set, err := d.XPath("id('x1 x2')")
	if err != nil {
		t.Fatal(err)
	}
	if set.Length() != 2 {
		t.Errorf("id: %d", set.Length())
	}
	// lang()
	sub, _ := d.AtXPath("//b")
	v, _ := sub.EvalXPath("lang('en')", nil)
	if !v.(bool) {
		t.Error("lang en")
	}
	sub, _ = d.AtXPath("//c")
	v, _ = sub.EvalXPath("lang('fr')", nil)
	if !v.(bool) {
		t.Error("lang fr-CA matches fr")
	}
	sub, _ = d.AtXPath("//a")
	v, _ = sub.EvalXPath("lang('en')", nil)
	if v.(bool) {
		t.Error("lang no xml:lang")
	}
}

func TestPositionLastCurrent(t *testing.T) {
	d, _ := XML(`<r><i>a</i><i>b</i><i>c</i></r>`)
	set, _ := d.XPath("//i[position()=2]")
	if set.First().Text() != "b" {
		t.Error("position")
	}
	set, _ = d.XPath("//i[last()]")
	if set.First().Text() != "c" {
		t.Error("last")
	}
	set, _ = d.XPath("//i[position() mod 2 = 1]")
	if set.Length() != 2 {
		t.Error("position mod")
	}
}

func TestNumberFormatting(t *testing.T) {
	d, _ := XML(`<r/>`)
	if evalStr(t, d, "string(0.5)") != "0.5" {
		t.Error("frac")
	}
	if evalStr(t, d, "string(3)") != "3" {
		t.Error("int")
	}
	if evalStr(t, d, "string(1 div 0)") != "Infinity" {
		t.Error("inf")
	}
	if evalStr(t, d, "string(-1 div 0)") != "-Infinity" {
		t.Error("-inf")
	}
	if evalStr(t, d, "string(0 div 0)") != "NaN" {
		t.Error("nan")
	}
	if evalStr(t, d, "string(true())") != "true" {
		t.Error("bool string")
	}
	if evalNum(t, d, "number('abc')") == evalNum(t, d, "number('abc')") {
		// NaN != NaN, so this is only true if not NaN
		t.Error("number of non-numeric should be NaN")
	}
}
