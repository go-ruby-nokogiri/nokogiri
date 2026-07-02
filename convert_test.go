// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

import (
	"math"
	"testing"
)

func TestConversions(t *testing.T) {
	// toBool over each type
	if !toBool(true) || toBool(false) {
		t.Fatal("toBool bool")
	}
	if toBool(0.0) || !toBool(1.0) || toBool(math.NaN()) {
		t.Fatal("toBool number")
	}
	if toBool("") || !toBool("x") {
		t.Fatal("toBool string")
	}
	if toBool(&nodeList{}) || !toBool(&nodeList{nodes: []*Node{{}}}) {
		t.Fatal("toBool nodelist")
	}
	if toBool(nil) {
		t.Fatal("toBool nil")
	}

	// toNumber over each type
	if toNumber(3.0) != 3 || toNumber(true) != 1 || toNumber(false) != 0 {
		t.Fatal("toNumber scalar")
	}
	if !math.IsNaN(toNumber("abc")) || toNumber("42") != 42 {
		t.Fatal("toNumber string")
	}
	if !math.IsNaN(toNumber(nil)) {
		t.Fatal("toNumber nil")
	}

	// toString over each type
	if toString("s") != "s" || toString(true) != "true" || toString(false) != "false" {
		t.Fatal("toString scalar")
	}
	if toString(2.0) != "2" {
		t.Fatal("toString number")
	}
	if toString(&nodeList{}) != "" {
		t.Fatal("toString empty nodelist")
	}
	if toString(nil) != "" {
		t.Fatal("toString nil")
	}
}

func TestStringValueNodeKinds(t *testing.T) {
	d, _ := XML(`<r a="av">text<!--cm--><![CDATA[cd]]><?pi pd?></r>`)
	root := d.Root()
	// attribute node string value
	attrs := attrNodes(root)
	if stringValue(attrs[0]) != "av" {
		t.Fatal("attr string value")
	}
	// text/comment/cdata/pi string values
	c := root.FirstChild()
	if stringValue(c) != "text" {
		t.Fatal("text sv")
	}
	if stringValue(c.Next()) != "cm" {
		t.Fatal("comment sv")
	}
	if stringValue(c.Next().Next()) != "cd" {
		t.Fatal("cdata sv")
	}
	if stringValue(c.Next().Next().Next()) != "pd" {
		t.Fatal("pi sv")
	}
	// element string value is descendant text
	if stringValue(root) != "textcd" {
		t.Fatalf("element sv: %q", stringValue(root))
	}
}

func TestEqualityCombinations(t *testing.T) {
	d, _ := XML(`<r><n>5</n></r>`)
	// number = number, number != number
	if !evalBool(t, d, "3 = 3") || evalBool(t, d, "3 = 4") {
		t.Fatal("num eq")
	}
	// bool = bool
	if !evalBool(t, d, "true() = true()") || evalBool(t, d, "true() = false()") {
		t.Fatal("bool eq")
	}
	// string = string
	if !evalBool(t, d, "'a' = 'a'") {
		t.Fatal("str eq")
	}
	// nodeset = bool
	if !evalBool(t, d, "//n = true()") {
		t.Fatal("nodeset = bool")
	}
	// nodeset = number
	if !evalBool(t, d, "//n = 5") {
		t.Fatal("nodeset = number")
	}
	// nodeset != number
	if !evalBool(t, d, "//n != 6") {
		t.Fatal("nodeset != number")
	}
	// nodeset = nodeset
	if !evalBool(t, d, "//n = //n") {
		t.Fatal("nodeset = nodeset")
	}
	// empty nodeset = string is false
	if evalBool(t, d, "//zzz = 'x'") {
		t.Fatal("empty nodeset eq")
	}
	// mixing bool with number/string coerces to bool
	if !evalBool(t, d, "true() = 1") {
		t.Fatal("bool = number")
	}
}

func TestRelationalCombinations(t *testing.T) {
	d, _ := XML(`<r><n>5</n><n>10</n></r>`)
	if !evalBool(t, d, "3 < 5") || !evalBool(t, d, "5 > 3") {
		t.Fatal("num rel")
	}
	if !evalBool(t, d, "5 <= 5") || !evalBool(t, d, "5 >= 5") {
		t.Fatal("num rel eq")
	}
	// nodeset relational both sides
	if !evalBool(t, d, "//n < 20") {
		t.Fatal("nodeset < scalar")
	}
	// scalar vs nodeset
	if !evalBool(t, d, "3 < //n") {
		t.Fatal("scalar < nodeset")
	}
}

func TestUnionDocumentOrder(t *testing.T) {
	d, _ := XML(`<r><a/><b/><c/></r>`)
	// union should produce document order regardless of operand order
	set, _ := d.XPath("//c | //a | //b")
	if set.Length() != 3 {
		t.Fatal("union count")
	}
	if set.At(0).Name != "a" || set.At(2).Name != "c" {
		t.Fatalf("union order: %s..%s", set.At(0).Name, set.At(2).Name)
	}
}
