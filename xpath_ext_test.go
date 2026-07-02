// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

import "testing"

// TestEvalXPathCtxNil confirms a nil context behaves like EvalXPath.
func TestEvalXPathCtxNil(t *testing.T) {
	doc, err := XML(`<r><a>1</a><a>2</a></r>`)
	if err != nil {
		t.Fatal(err)
	}
	v, err := doc.Node.EvalXPathCtx("count(//a)", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if f, ok := v.(float64); !ok || f != 2 {
		t.Fatalf("count(//a) = %v, want 2", v)
	}
}

// TestEvalXPathCtxVars exercises every boundary variable kind.
func TestEvalXPathCtxVars(t *testing.T) {
	doc, err := XML(`<r><a>x</a><a>y</a></r>`)
	if err != nil {
		t.Fatal(err)
	}
	set, err := doc.XPath("//a")
	if err != nil {
		t.Fatal(err)
	}
	first := set.First()
	cases := []struct {
		expr string
		vars map[string]any
		want any
	}{
		{"$s", map[string]any{"s": "hello"}, "hello"},
		{"$n + 1", map[string]any{"n": float64(41)}, float64(42)},
		{"$i + 1", map[string]any{"i": 10}, float64(11)},
		{"$j + 1", map[string]any{"j": int64(10)}, float64(11)},
		{"$b", map[string]any{"b": true}, true},
		{"count($ns)", map[string]any{"ns": set}, float64(2)},
		{"string($one)", map[string]any{"one": first}, "x"},
		{"count($sl)", map[string]any{"sl": []*Node{first}}, float64(1)},
		{"string($nilv)", map[string]any{"nilv": nil}, ""},
		{"count($nilnode)", map[string]any{"nilnode": (*Node)(nil)}, float64(0)},
		{"string($bad)", map[string]any{"bad": struct{}{}}, ""},
	}
	for _, c := range cases {
		got, err := doc.Node.EvalXPathCtx(c.expr, nil, &XPathContext{Vars: c.vars})
		if err != nil {
			t.Fatalf("%s: %v", c.expr, err)
		}
		if got != c.want {
			t.Errorf("%s = %v, want %v", c.expr, got, c.want)
		}
	}
}

// TestEvalXPathCtxVarsNodeSetResult checks a node-set variable returned as a set.
func TestEvalXPathCtxVarsNodeSetResult(t *testing.T) {
	doc, err := XML(`<r><a/><a/></r>`)
	if err != nil {
		t.Fatal(err)
	}
	set, _ := doc.XPath("//a")
	got, err := doc.Node.EvalXPathCtx("$ns", nil, &XPathContext{Vars: map[string]any{"ns": set}})
	if err != nil {
		t.Fatal(err)
	}
	ns, ok := got.(*NodeSet)
	if !ok || ns.Length() != 2 {
		t.Fatalf("$ns = %v, want 2-node set", got)
	}
}

// TestEvalXPathCtxFuncHook drives the extension-function seam, both the resolved
// and the fall-through (ok=false) paths.
func TestEvalXPathCtxFuncHook(t *testing.T) {
	doc, err := XML(`<r/>`)
	if err != nil {
		t.Fatal(err)
	}
	ctx := &XPathContext{
		ResolveFunc: func(name string, args []any) (any, bool) {
			switch name {
			case "my:double":
				return args[0].(float64) * 2, true
			case "my:echo":
				return args[0].(string), true
			case "my:nodes":
				return NewNodeSet(nil), true
			}
			return nil, false
		},
	}
	got, err := doc.Node.EvalXPathCtx("my:double(21)", nil, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got != float64(42) {
		t.Fatalf("my:double(21) = %v, want 42", got)
	}
	got, _ = doc.Node.EvalXPathCtx("my:echo('hi')", nil, ctx)
	if got != "hi" {
		t.Fatalf("my:echo = %v", got)
	}
	got, _ = doc.Node.EvalXPathCtx("count(my:nodes())", nil, ctx)
	if got != float64(0) {
		t.Fatalf("count(my:nodes()) = %v", got)
	}
	// Fall-through: an unresolved function still errors.
	if _, err := doc.Node.EvalXPathCtx("no:such()", nil, ctx); err == nil {
		t.Fatal("expected error for unresolved function")
	}
}

// TestEvalXPathCtxCurrentOverride checks current() honours the override, including
// inside a predicate.
func TestEvalXPathCtxCurrentOverride(t *testing.T) {
	doc, err := XML(`<r><a id="1"/><a id="2"/></r>`)
	if err != nil {
		t.Fatal(err)
	}
	set, _ := doc.XPath("//a")
	second := set.At(1)
	// With current() forced to the 2nd <a>, //a[@id=current()/@id] selects it.
	got, err := doc.Node.EvalXPathCtx("//a[@id=current()/@id]", nil, &XPathContext{Current: second})
	if err != nil {
		t.Fatal(err)
	}
	ns, ok := got.(*NodeSet)
	if !ok || ns.Length() != 1 || ns.First().Attribute("id") != "2" {
		t.Fatalf("current-override predicate = %v", got)
	}
}

// TestEvalXPathCtxParseError propagates a parse error.
func TestEvalXPathCtxParseError(t *testing.T) {
	doc, _ := XML(`<r/>`)
	if _, err := doc.Node.EvalXPathCtx("///", nil, &XPathContext{}); err == nil {
		t.Fatal("expected parse error")
	}
}

// TestNewDocumentAndNodes covers the result-tree constructors.
func TestNewDocumentAndNodes(t *testing.T) {
	d := NewDocument()
	if d.Type != DocumentNode {
		t.Fatalf("NewDocument type = %v", d.Type)
	}
	el := d.NewElement("x")
	el.AddChild(d.NewText("hi"))
	pi := d.NewPI("php", "echo 1;")
	el.AddChild(pi)
	d.AddChild(el)
	got := d.ToXML()
	want := `<x>hi<?php echo 1;?></x>`
	if got != want {
		t.Fatalf("ToXML = %q, want %q", got, want)
	}
	if d.NewPI("t", "").ToXML() != "<?t?>" {
		t.Fatalf("empty PI = %q", d.NewPI("t", "").ToXML())
	}
}
