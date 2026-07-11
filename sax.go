// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// SAXHandler receives streaming parse events, the Go analogue of a
// Nokogiri::XML::SAX::Document subclass. Every method is a callback; embed
// SAXDocument to get no-op defaults and override only what you need.
type SAXHandler interface {
	// StartDocument fires once before any other event.
	StartDocument()
	// EndDocument fires once after the last event of a successful parse.
	EndDocument()
	// StartElement fires at an element's start tag. name is the qualified name
	// (prefix:local) as written; attrs lists every attribute in source order,
	// including xmlns declarations, matching the non-namespaced start_element.
	StartElement(name string, attrs []*Attr)
	// EndElement fires at an element's end tag (or the implied end of an
	// empty-element tag).
	EndElement(name string)
	// Characters fires for a run of ordinary character data.
	Characters(text string)
	// Comment fires for a comment's content.
	Comment(text string)
	// CdataBlock fires for the content of a CDATA section.
	CdataBlock(text string)
	// ProcessingInstruction fires for a <?target data?> instruction (the XML
	// declaration is not reported).
	ProcessingInstruction(target, data string)
	// Error fires when the parse cannot continue; the parse then stops.
	Error(message string)
}

// SAXDocument provides no-op implementations of every SAXHandler callback, so a
// handler need only embed it and override the events it cares about — the Go
// equivalent of subclassing Nokogiri::XML::SAX::Document.
type SAXDocument struct{}

// StartDocument is a no-op default.
func (SAXDocument) StartDocument() {}

// EndDocument is a no-op default.
func (SAXDocument) EndDocument() {}

// StartElement is a no-op default.
func (SAXDocument) StartElement(string, []*Attr) {}

// EndElement is a no-op default.
func (SAXDocument) EndElement(string) {}

// Characters is a no-op default.
func (SAXDocument) Characters(string) {}

// Comment is a no-op default.
func (SAXDocument) Comment(string) {}

// CdataBlock is a no-op default.
func (SAXDocument) CdataBlock(string) {}

// ProcessingInstruction is a no-op default.
func (SAXDocument) ProcessingInstruction(string, string) {}

// Error is a no-op default.
func (SAXDocument) Error(string) {}

// SAXParser drives a SAXHandler over an XML source, mirroring
// Nokogiri::XML::SAX::Parser. Parsing is event-driven; no DOM is built.
type SAXParser struct {
	handler SAXHandler
}

// NewSAXParser returns a parser that dispatches events to h.
func NewSAXParser(h SAXHandler) *SAXParser { return &SAXParser{handler: h} }

// Parse streams s through the handler (Nokogiri::XML::SAX::Parser#parse).
func (p *SAXParser) Parse(s string) error { return p.run(s) }

// ParseReader reads all of r, then streams it through the handler. libxml2's SAX
// is incremental; the whole source is buffered here so CDATA sections can be told
// apart from ordinary text, which needs the raw bytes.
func (p *SAXParser) ParseReader(r io.Reader) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	return p.run(string(b))
}

// run tokenizes src and dispatches SAX events. It tracks the open-element stack so
// a mismatched or missing end tag is reported through Error (and stops the parse),
// as libxml2 does — though the exact diagnostic text is not reproduced.
func (p *SAXParser) run(src string) error {
	h := p.handler
	h.StartDocument()
	dec := xml.NewDecoder(strings.NewReader(src))
	dec.Strict = true
	var stack []string
	for {
		startOff := dec.InputOffset()
		tok, err := dec.RawToken()
		if err == io.EOF {
			break
		}
		if err != nil {
			h.Error(err.Error())
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			name := qname(t.Name)
			var attrs []*Attr
			for _, a := range t.Attr {
				attrs = append(attrs, &Attr{Name: a.Name.Local, Prefix: a.Name.Space, Value: a.Value})
			}
			h.StartElement(name, attrs)
			stack = append(stack, name)
		case xml.EndElement:
			name := qname(t.Name)
			if len(stack) == 0 || stack[len(stack)-1] != name {
				msg := fmt.Sprintf("tag mismatch: unexpected end tag </%s>", name)
				h.Error(msg)
				return &xml.SyntaxError{Msg: msg}
			}
			stack = stack[:len(stack)-1]
			h.EndElement(name)
		case xml.CharData:
			if isCDATASpan(src, int(startOff), int(dec.InputOffset())) {
				h.CdataBlock(string(t))
			} else {
				h.Characters(string(t))
			}
		case xml.Comment:
			h.Comment(string(t))
		case xml.ProcInst:
			if t.Target == "xml" {
				continue // the XML declaration is not a PI event
			}
			h.ProcessingInstruction(t.Target, string(t.Inst))
		case xml.Directive:
			// DTD/doctype declarations have no standard SAX text callback here.
		}
	}
	if len(stack) != 0 {
		msg := fmt.Sprintf("premature end of data: unclosed <%s>", stack[len(stack)-1])
		h.Error(msg)
		return &xml.SyntaxError{Msg: msg}
	}
	h.EndDocument()
	return nil
}

// qname renders an encoding/xml Name back into its written "prefix:local" form
// (in RawToken mode Name.Space carries the literal prefix).
func qname(n xml.Name) string {
	if n.Space != "" {
		return n.Space + ":" + n.Local
	}
	return n.Local
}
