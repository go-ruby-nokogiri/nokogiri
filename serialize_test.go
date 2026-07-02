// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

import "testing"

func TestSerializeEscaping(t *testing.T) {
	d, _ := XML(`<r a="&lt;&amp;&gt;&quot;">a &lt; b &amp; c</r>`)
	root := d.Root()
	if root.Attribute("a") != `<&>"` {
		t.Fatalf("attr unescape: %q", root.Attribute("a"))
	}
	out := root.ToXML()
	if out != `<r a="&lt;&amp;&gt;&quot;">a &lt; b &amp; c</r>` {
		t.Fatalf("escape roundtrip: %q", out)
	}
}

func TestSerializeHTMLVoid(t *testing.T) {
	d, err := HTML(`<html><body><br><img src="x.png"><p>hi</p></body></html>`)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := d.AtCSS("body")
	out := body.InnerHTML()
	if out != `<br><img src="x.png"><p>hi</p>` {
		t.Fatalf("void html: %q", out)
	}
	// XML serialization self-closes empties
	br, _ := d.AtCSS("br")
	if br.ToXML() != `<br/>` {
		t.Fatalf("br xml: %q", br.ToXML())
	}
}

func TestSerializeVariants(t *testing.T) {
	d, _ := XML(`<r><a>1</a><b/></r>`)
	root := d.Root()
	if root.InnerXML() != `<a>1</a><b/>` {
		t.Fatalf("innerxml: %q", root.InnerXML())
	}
	// ToS uses document default (XML here)
	if root.ToS() != root.ToXML() {
		t.Fatal("ToS xml")
	}
	h, _ := HTML(`<html><body><p>x</p></body></html>`)
	p, _ := h.AtCSS("p")
	if p.ToS() != p.ToHTML() {
		t.Fatal("ToS html")
	}
	// detached node ToS defaults to XML
	det := &Node{Type: ElementNode, Name: "z"}
	if det.ToS() != `<z/>` {
		t.Fatalf("detached ToS: %q", det.ToS())
	}
}

func TestSerializeCDATAAndComment(t *testing.T) {
	d, _ := XML(`<r><![CDATA[<raw>]]><!--note--><?pi data?></r>`)
	out := d.Root().ToXML()
	if out != `<r><![CDATA[<raw>]]><!--note--><?pi data?></r>` {
		t.Fatalf("cdata/comment/pi: %q", out)
	}
}

func TestSerializeNamespaceDecls(t *testing.T) {
	d, _ := XML(`<r xmlns="urn:d" xmlns:x="urn:x"><x:c/></r>`)
	out := d.Root().ToXML()
	if out != `<r xmlns="urn:d" xmlns:x="urn:x"><x:c/></r>` {
		t.Fatalf("ns decls: %q", out)
	}
}

func TestSerializeDoctype(t *testing.T) {
	d, err := HTML(`<!DOCTYPE html><html><body>x</body></html>`)
	if err != nil {
		t.Fatal(err)
	}
	out := d.ToHTML()
	if len(out) < 15 || out[:15] != "<!DOCTYPE html>" {
		t.Fatalf("doctype html: %q", out)
	}
}

func TestSerializeDocumentNode(t *testing.T) {
	d, _ := XML(`<r><a/></r>`)
	if d.Node.ToXML() != `<r><a/></r>` {
		t.Fatalf("document serialize: %q", d.Node.ToXML())
	}
}

func TestProcInstWithoutData(t *testing.T) {
	pi := &Node{Type: ProcessingInstructionNode, Name: "go"}
	if pi.ToXML() != `<?go?>` {
		t.Fatalf("pi no data: %q", pi.ToXML())
	}
}
