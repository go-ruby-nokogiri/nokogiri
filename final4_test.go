// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

import "testing"

func TestEqualityAllBranches(t *testing.T) {
	d, _ := XML(`<r><a>1</a><b>2</b></r>`)
	// nodeset = nodeset with no equal pair -> false
	if evalBool(t, d, "//a = //b") {
		t.Error("distinct nodeset equality should be false")
	}
	// nodeset != nodeset with a differing pair -> true
	if !evalBool(t, d, "//a != //b") {
		t.Error("distinct nodeset inequality should be true")
	}
	// scalar on the LEFT, nodeset on the RIGHT (rIsNode branch)
	if !evalBool(t, d, "'1' = //a") {
		t.Error("string = nodeset (right)")
	}
	if !evalBool(t, d, "1 = //a") {
		t.Error("number = nodeset (right)")
	}
	if !evalBool(t, d, "true() = //a") {
		t.Error("bool = nodeset (right)")
	}
	// nodeset = string with no match -> false
	if evalBool(t, d, "//a = 'zzz'") {
		t.Error("nodeset = nonmatching string")
	}
	// string vs number scalar coercion
	if !evalBool(t, d, "'3' = 3") {
		t.Error("string = number coercion")
	}
	// plain string vs string inequality
	if !evalBool(t, d, "'a' != 'b'") {
		t.Error("string inequality")
	}
}

func TestRelationalNodeBothSides(t *testing.T) {
	d, _ := XML(`<r><a>3</a><b>5</b></r>`)
	// nodeset < nodeset
	if !evalBool(t, d, "//a < //b") {
		t.Error("nodeset < nodeset")
	}
	if evalBool(t, d, "//b < //a") {
		t.Error("nodeset < nodeset false")
	}
	// all four relational operators against a scalar
	if !evalBool(t, d, "//a > 2") || !evalBool(t, d, "//a >= 3") ||
		!evalBool(t, d, "//a < 4") || !evalBool(t, d, "//a <= 3") {
		t.Error("relational ops")
	}
}

func TestPrecedingSiblingAxisExplicit(t *testing.T) {
	d, _ := XML(`<r><a/><b/><c/></r>`)
	set, _ := d.XPath("//c/preceding-sibling::*")
	if set.Length() != 2 {
		t.Fatalf("preceding-sibling: %d", set.Length())
	}
	// following-sibling too
	set, _ = d.XPath("//a/following-sibling::*")
	if set.Length() != 2 {
		t.Fatalf("following-sibling: %d", set.Length())
	}
}

func TestFollowingAxisExplicit(t *testing.T) {
	d, _ := XML(`<r><a><x/></a><b><y/></b></r>`)
	set, _ := d.XPath("//x/following::*")
	names := map[string]bool{}
	set.Each(func(n *Node) { names[n.Name] = true })
	if !names["b"] || !names["y"] {
		t.Fatalf("following: %v", names)
	}
}

func TestDocOrderOrphanUnion(t *testing.T) {
	d, _ := XML(`<r><a/></r>`)
	// Detached node with no parent, unioned with an in-tree node, forces the
	// docOrder fallback for an unindexed, parent-less node.
	orphan := d.NewElement("orphan")
	vars := map[string]xpValue{"o": &nodeList{nodes: []*Node{orphan}}}
	v, err := evalXPath("//a | $o", &d.Node, vars, nil)
	if err != nil {
		t.Fatal(err)
	}
	nl := v.(*nodeList)
	if len(nl.nodes) != 2 {
		t.Fatalf("orphan union: %d", len(nl.nodes))
	}
}
