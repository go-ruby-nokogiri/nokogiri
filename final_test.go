// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

import "testing"

func TestPseudoRemaining(t *testing.T) {
	d, _ := HTML(`<ul><li>1</li><li>2</li><li>3</li></ul><ol><li>x</li></ol>`)
	cases := []struct {
		sel  string
		want int
	}{
		{"li:only-child", 1},        // the sole <li> in <ol>
		{"li:nth-last-child(1)", 2}, // last li of each list
		{"li:last-of-type", 2},
		{"li:only-of-type", 1},
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

func TestNotArgPseudoAndAttr(t *testing.T) {
	d, _ := HTML(`<ul><li class="x">1</li><li>2</li><li>3</li></ul>`)
	// :not with a pseudo inside
	set, _ := d.CSS("li:not(:first-child)")
	if set.Length() != 2 {
		t.Errorf(":not(:first-child): %d", set.Length())
	}
	// :not with attribute op inside
	set, _ = d.CSS(`li:not([class~="x"])`)
	if set.Length() != 2 {
		t.Errorf(":not([class~=x]): %d", set.Length())
	}
	// :not(*) -> matches nothing (universal)
	set, _ = d.CSS("li:not(*)")
	if set.Length() != 0 {
		t.Errorf(":not(*): %d", set.Length())
	}
}

func TestSetDocRecursiveOnSubtree(t *testing.T) {
	d, _ := XML(`<r/>`)
	// build a detached subtree, then attach: setDocRecursive must stamp children
	parent := d.NewElement("p")
	child := d.NewElement("c")
	parent.AddChild(child)
	d.Root().AddChild(parent)
	if child.Document() != d {
		t.Fatal("child doc not stamped")
	}
	// query reaches the grandchild
	set, _ := d.CSS("p c")
	if set.Length() != 1 {
		t.Fatalf("grandchild query: %d", set.Length())
	}
}

func TestSiblingInsertionWithoutParent(t *testing.T) {
	d, _ := XML(`<r/>`)
	// a detached node: AddNextSibling/AddPreviousSibling with nil parent
	orphan := d.NewElement("o")
	sib := d.NewElement("s")
	orphan.AddNextSibling(sib)
	if sib.Parent() != nil {
		t.Fatal("next sibling of orphan has no parent")
	}
	orphan.AddPreviousSibling(d.NewElement("p"))
}

func TestHTMLDoctypeConversion(t *testing.T) {
	d, err := HTML(`<!DOCTYPE html><html><head></head><body>x</body></html>`)
	if err != nil {
		t.Fatal(err)
	}
	// doctype node exists among document children
	var hasDoctype bool
	for c := d.FirstChild(); c != nil; c = c.Next() {
		if c.Type == DoctypeNode {
			hasDoctype = true
		}
	}
	if !hasDoctype {
		t.Fatal("no doctype node")
	}
}

func TestNestedNamespaceResolution(t *testing.T) {
	d, _ := XML(`<root xmlns:a="urn:a"><a:outer xmlns:b="urn:b"><b:inner a:attr="v"/></a:outer></root>`)
	// inner element resolves b; attribute resolves a from outer scope
	set, _ := d.Node.XPath("//b:inner", map[string]string{"b": "urn:b"})
	if set.Length() != 1 {
		t.Fatalf("nested ns: %d", set.Length())
	}
	inner := set.First()
	if inner.NsURI != "urn:b" {
		t.Fatalf("inner nsuri: %q", inner.NsURI)
	}
	// the a:attr attribute got its namespace resolved from the outer scope
	var found bool
	for _, at := range inner.Attrs {
		if at.Prefix == "a" && at.Namespace == "urn:a" {
			found = true
		}
	}
	if !found {
		t.Fatal("attribute namespace not resolved")
	}
}

func TestFnIDWithNodeSetArg(t *testing.T) {
	d, _ := XML(`<r><ref>t1</ref><a id="t1">A</a><a id="t2">B</a></r>`)
	// id(//ref) uses the node-set's string value ("t1")
	set, err := d.XPath("id(//ref)")
	if err != nil {
		t.Fatal(err)
	}
	if set.Length() != 1 || set.First().Text() != "A" {
		t.Fatalf("id(nodeset): %d", set.Length())
	}
}

func TestTranslateDuplicateFrom(t *testing.T) {
	d, _ := XML(`<r/>`)
	// duplicate chars in the 'from' arg: first mapping wins
	if evalStr(t, d, "translate('abc', 'aa', 'XY')") != "Xbc" {
		t.Errorf("translate dup: %q", evalStr(t, d, "translate('abc', 'aa', 'XY')"))
	}
}

func TestConcatNoArgsAndNumberNodeSet(t *testing.T) {
	d, _ := XML(`<r><n>7</n></r>`)
	// concat of two node-set string values
	if evalStr(t, d, "concat(//n, //n)") != "77" {
		t.Error("concat nodeset")
	}
	// number() of a node-set
	if evalNum(t, d, "number(//n)") != 7 {
		t.Error("number nodeset")
	}
	// arithmetic forcing node-set -> number
	if evalNum(t, d, "//n + 1") != 8 {
		t.Error("nodeset arithmetic")
	}
}

func TestDocOrderForCreatedNodes(t *testing.T) {
	d, _ := XML(`<r><a/></r>`)
	// build new nodes then union with existing -> exercises docOrder fallback
	root := d.Root()
	nw := d.NewElement("z")
	root.AddChild(nw)
	set, _ := d.XPath("//a | //z")
	if set.Length() != 2 {
		t.Fatalf("created node union: %d", set.Length())
	}
}

func TestSubstringRounding(t *testing.T) {
	d, _ := XML(`<r/>`)
	// XPath rounds the position/length
	if evalStr(t, d, "substring('12345', 1.5, 2.6)") != "234" {
		t.Errorf("substring round: %q", evalStr(t, d, "substring('12345', 1.5, 2.6)"))
	}
	// NaN start -> empty
	if evalStr(t, d, "substring('12345', number('x'))") != "" {
		t.Error("substring NaN")
	}
}
