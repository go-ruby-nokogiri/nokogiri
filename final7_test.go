// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

import "testing"

func TestSiblingInsertMiddle(t *testing.T) {
	d, _ := XML(`<r><a/><b/></r>`)
	root := d.Root()
	a := root.FirstChild()
	// insert after 'a', which HAS a next sibling ('b') -> rewires b.prev
	a.AddNextSibling(d.NewElement("x"))
	if root.ToXML() != `<r><a/><x/><b/></r>` {
		t.Fatalf("insert middle next: %q", root.ToXML())
	}
	// insert before 'b', which HAS a prev sibling -> rewires prev.next
	b := root.LastChild()
	b.AddPreviousSibling(d.NewElement("y"))
	if root.ToXML() != `<r><a/><x/><y/><b/></r>` {
		t.Fatalf("insert middle prev: %q", root.ToXML())
	}
}

func TestNestedPseudoArgParens(t *testing.T) {
	// a pseudo whose argument itself contains parentheses exercises the nested
	// depth counter in the CSS lexer.
	d, _ := HTML(`<ul><li>1</li><li>2</li></ul>`)
	// :not(:nth-child(1)) -> the argument ":nth-child(1)" has inner parens
	set, err := d.CSS("li:not(:nth-child(1))")
	if err != nil {
		t.Fatal(err)
	}
	if set.Length() != 1 {
		t.Fatalf(":not(:nth-child(1)): %d", set.Length())
	}
}

func TestCSSUnexpectedCharacter(t *testing.T) {
	d, _ := HTML(`<p>x</p>`)
	// a selector that starts with a character that cannot begin a simple selector
	if _, err := d.CSS("%bad"); err == nil {
		t.Error("expected unexpected-character error")
	}
	if _, err := d.CSS("p %bad"); err == nil {
		t.Error("expected error after combinator")
	}
}
