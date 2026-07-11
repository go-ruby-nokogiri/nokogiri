// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// recorder records every SAX event as a line of text for golden comparison.
type recorder struct {
	SAXDocument
	lines []string
}

func (r *recorder) StartDocument() { r.lines = append(r.lines, "start_doc") }
func (r *recorder) EndDocument()   { r.lines = append(r.lines, "end_doc") }
func (r *recorder) StartElement(name string, attrs []*Attr) {
	s := "start " + name
	for _, a := range attrs {
		s += fmt.Sprintf(" %s=%q", a.qualified(), a.Value)
	}
	r.lines = append(r.lines, s)
}
func (r *recorder) EndElement(name string) { r.lines = append(r.lines, "end "+name) }
func (r *recorder) Characters(t string)    { r.lines = append(r.lines, fmt.Sprintf("chars %q", t)) }
func (r *recorder) Comment(t string)       { r.lines = append(r.lines, fmt.Sprintf("comment %q", t)) }
func (r *recorder) CdataBlock(t string)    { r.lines = append(r.lines, fmt.Sprintf("cdata %q", t)) }
func (r *recorder) ProcessingInstruction(target, data string) {
	r.lines = append(r.lines, fmt.Sprintf("pi %s %q", target, data))
}
func (r *recorder) Error(m string) { r.lines = append(r.lines, "error") }

func TestSAXFullStream(t *testing.T) {
	rec := &recorder{}
	err := NewSAXParser(rec).Parse(
		`<?xml version="1.0"?><r xmlns:a="urn:a" a:x="1" b="2"><c>hi</c><!--k--><![CDATA[<raw>]]><?pi go?></r>`)
	if err != nil {
		t.Fatalf("parse err: %v", err)
	}
	want := []string{
		"start_doc",
		`start r xmlns:a="urn:a" a:x="1" b="2"`,
		"start c",
		`chars "hi"`,
		"end c",
		`comment "k"`,
		`cdata "<raw>"`,
		`pi pi "go"`,
		"end r",
		"end_doc",
	}
	if strings.Join(rec.lines, "\n") != strings.Join(want, "\n") {
		t.Fatalf("events:\n got %v\nwant %v", rec.lines, want)
	}
}

func TestSAXEndTagMismatch(t *testing.T) {
	rec := &recorder{}
	err := NewSAXParser(rec).Parse(`<r><a></b></r>`)
	if err == nil {
		t.Fatal("expected mismatch error")
	}
	if rec.lines[len(rec.lines)-1] != "error" {
		t.Fatalf("last event = %q, want error", rec.lines[len(rec.lines)-1])
	}
}

func TestSAXUnexpectedEndTag(t *testing.T) {
	rec := &recorder{}
	// An end tag with nothing open trips the empty-stack branch.
	if err := NewSAXParser(rec).Parse(`</r>`); err == nil {
		t.Fatal("expected error for stray end tag")
	}
}

func TestSAXUnclosed(t *testing.T) {
	rec := &recorder{}
	err := NewSAXParser(rec).Parse(`<r><a></a>`)
	if err == nil {
		t.Fatal("expected unclosed error")
	}
	if rec.lines[len(rec.lines)-1] != "error" {
		t.Fatalf("last event = %q, want error", rec.lines[len(rec.lines)-1])
	}
}

func TestSAXDecoderError(t *testing.T) {
	rec := &recorder{}
	// An undefined entity makes the tokenizer itself fail.
	if err := NewSAXParser(rec).Parse(`<r>&nope;</r>`); err == nil {
		t.Fatal("expected tokenizer error")
	}
}

func TestSAXParseReader(t *testing.T) {
	rec := &recorder{}
	if err := NewSAXParser(rec).ParseReader(strings.NewReader(`<r/>`)); err != nil {
		t.Fatal(err)
	}
	want := []string{"start_doc", "start r", "end r", "end_doc"}
	if strings.Join(rec.lines, "\n") != strings.Join(want, "\n") {
		t.Fatalf("reader events = %v", rec.lines)
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

func TestSAXParseReaderError(t *testing.T) {
	rec := &recorder{}
	if err := NewSAXParser(rec).ParseReader(errReader{}); err == nil {
		t.Fatal("expected read error")
	}
}

// bare embeds SAXDocument without overriding anything, exercising the no-op
// default callbacks (including Error).
type bare struct{ SAXDocument }

func TestSAXDefaultNoOps(t *testing.T) {
	// A well-formed doc with every node type drives the no-op callbacks.
	if err := NewSAXParser(bare{}).Parse(
		`<r a="1"><c>hi</c><!--k--><![CDATA[x]]><?pi go?></r>`); err != nil {
		t.Fatal(err)
	}
	// A malformed doc drives the no-op Error callback.
	if err := NewSAXParser(bare{}).Parse(`<r></s>`); err == nil {
		t.Fatal("expected error")
	}
	// A directive/doctype is skipped silently.
	if err := NewSAXParser(bare{}).Parse(`<!DOCTYPE r><r/>`); err != nil {
		t.Fatal(err)
	}
}

func TestSAXDocumentNoOpMethods(t *testing.T) {
	// Call each no-op default directly so its (empty) body is exercised even for a
	// handler that overrides everything else.
	var s SAXDocument
	s.StartDocument()
	s.EndDocument()
	s.StartElement("x", nil)
	s.EndElement("x")
	s.Characters("t")
	s.Comment("c")
	s.CdataBlock("d")
	s.ProcessingInstruction("t", "v")
	s.Error("e")
}

func TestSAXPrefixedElement(t *testing.T) {
	rec := &recorder{}
	if err := NewSAXParser(rec).Parse(`<a:r xmlns:a="urn:a"><a:c/></a:r>`); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"start_doc",
		`start a:r xmlns:a="urn:a"`,
		"start a:c",
		"end a:c",
		"end a:r",
		"end_doc",
	}
	if strings.Join(rec.lines, "\n") != strings.Join(want, "\n") {
		t.Fatalf("prefixed events = %v", rec.lines)
	}
}
