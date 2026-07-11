// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

// These differential oracles pin the newer surface — libxml2-style to_xml
// indentation, the SAX event stream, and #namespaces — against the real Nokogiri
// gem. They are skip-gated exactly like the oracle in oracle_test.go: when a
// usable ruby with nokogiri (>= 4.0) is absent they are skipped, and every case
// also has a deterministic golden test elsewhere so the ruby-free lanes are green.

// gemEval runs a one-liner that prints an inspected value and returns the raw
// stdout (trailing newline trimmed).
func gemEval(t *testing.T, bin, script string, args ...string) string {
	t.Helper()
	cmd := exec.Command(bin, append([]string{"-e", script}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gem error: %v\n%s", err, out)
	}
	return strings.TrimRight(string(out), "\n")
}

func TestOracleXMLSerialization(t *testing.T) {
	bin := oracleRuby(t)
	// Node-level to_xml (no declaration) across the formatting-sensitive shapes.
	nodeDocs := []string{
		`<a><b/><c/></a>`,
		`<a><b><c/></b></a>`,
		`<catalog><book id="b1"><title>Alpha</title><price>10</price></book></catalog>`,
		`<p>Para <span>inner</span> tail</p>`,
		`<a><!--c--><b/></a>`,
		`<r xmlns="urn:d" xmlns:x="urn:x"><x:c/></r>`,
		`<a><![CDATA[<raw>]]></a>`,
	}
	for _, doc := range nodeDocs {
		got, err := XML(doc)
		if err != nil {
			t.Fatalf("go parse %q: %v", doc, err)
		}
		script := `$stdout.binmode; require "nokogiri"; print Nokogiri::XML(ARGV[0]).root.to_xml`
		want := gemEval(t, bin, script, doc)
		if g := got.Root().ToXML(); g != want {
			t.Errorf("node to_xml %q:\n go=%q\ngem=%q", doc, g, want)
		}
	}
	// Document-level to_xml (declaration + trailing newline). A sentinel guards the
	// trailing newline, which a naive trim would otherwise swallow.
	for _, doc := range []string{`<r><a/></r>`, `<!DOCTYPE greeting SYSTEM "x.dtd"><greeting/>`} {
		got, _ := XML(doc)
		script := `$stdout.binmode; require "nokogiri"; print Nokogiri::XML(ARGV[0]).to_xml + "\x00"`
		want := strings.TrimSuffix(gemEval(t, bin, script, doc), "\x00")
		if g := got.Node.ToXML(); g != want {
			t.Errorf("doc to_xml %q:\n go=%q\ngem=%q", doc, g, want)
		}
	}
}

func TestOracleHTML5Serialization(t *testing.T) {
	bin := oracleRuby(t)
	docs := []string{
		`<div><p>a<br>b</p><img src="x"><script>1<2</script></div>`,
		`<div><style>a>b{c:1}</style></div>`,
		`<ul><li>1<li>2</ul>`,
	}
	for _, doc := range docs {
		got, _ := HTML5(doc)
		div, _ := got.AtCSS("div, ul")
		script := `$stdout.binmode; require "nokogiri"
d = Nokogiri::HTML5(ARGV[0])
print (d.at_css("div") || d.at_css("ul")).to_html`
		want := gemEval(t, bin, script, doc)
		if g := div.ToHTML(); g != want {
			t.Errorf("html5 to_html %q:\n go=%q\ngem=%q", doc, g, want)
		}
	}
}

// gemNamespaces returns the gem's #namespaces hash for the node selected by xp,
// rendered as sorted "key=value" lines for order-independent comparison.
func gemNamespaces(t *testing.T, bin, doc, xp string) []string {
	t.Helper()
	script := `require "nokogiri"
doc = Nokogiri::XML(STDIN.read)
n = doc.at_xpath(ARGV[0])
n.namespaces.sort.each { |k, v| puts "#{k}=#{v}" }`
	cmd := exec.Command(bin, "-e", script, xp)
	cmd.Stdin = strings.NewReader(doc)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gem namespaces error: %v\n%s", err, out)
	}
	s := strings.TrimRight(string(out), "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func TestOracleNamespaces(t *testing.T) {
	bin := oracleRuby(t)
	doc := `<root xmlns="urn:def" xmlns:a="urn:a"><a:child xmlns:b="urn:b"><a:g/></a:child></root>`
	for _, xp := range []string{"//*[local-name()='g']", "//*[local-name()='child']", "/*"} {
		d, _ := XML(doc)
		n, _ := d.Node.AtXPath(xp, nil)
		var goLines []string
		for k, v := range n.Namespaces() {
			goLines = append(goLines, fmt.Sprintf("%s=%s", k, v))
		}
		want := gemNamespaces(t, bin, doc, xp)
		if !eqStringSet(goLines, want) {
			t.Errorf("namespaces %q: go=%v gem=%v", xp, goLines, want)
		}
	}
}

func TestOracleSAX(t *testing.T) {
	bin := oracleRuby(t)
	doc := `<?xml version="1.0"?><r xmlns:a="urn:a" a:x="1" b="2"><c>hi</c><!--k--><![CDATA[<raw>]]><?pi go?></r>`
	// Gem SAX event stream.
	script := `require "nokogiri"
class Cb < Nokogiri::XML::SAX::Document
  def start_document; puts "start_doc"; end
  def end_document; puts "end_doc"; end
  def start_element(n, a); puts "start #{n}" + a.map { |k, v| " #{k}=#{v}" }.join; end
  def end_element(n); puts "end #{n}"; end
  def characters(s); puts "chars #{s}"; end
  def comment(s); puts "comment #{s}"; end
  def cdata_block(s); puts "cdata #{s}"; end
  def processing_instruction(t, c); puts "pi #{t} #{c}"; end
end
Nokogiri::XML::SAX::Parser.new(Cb.new).parse(STDIN.read)`
	cmd := exec.Command(bin, "-e", script)
	cmd.Stdin = strings.NewReader(doc)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gem sax error: %v\n%s", err, out)
	}
	want := strings.TrimRight(string(out), "\n")

	rec := &plainRecorder{}
	if err := NewSAXParser(rec).Parse(doc); err != nil {
		t.Fatalf("go sax: %v", err)
	}
	got := strings.Join(rec.lines, "\n")
	if got != want {
		t.Errorf("sax stream:\n go=%q\ngem=%q", got, want)
	}
}

// plainRecorder renders events in the same textual shape as the Ruby oracle
// script (values unquoted), so the two streams compare directly.
type plainRecorder struct {
	SAXDocument
	lines []string
}

func (r *plainRecorder) StartDocument() { r.lines = append(r.lines, "start_doc") }
func (r *plainRecorder) EndDocument()   { r.lines = append(r.lines, "end_doc") }
func (r *plainRecorder) StartElement(name string, attrs []*Attr) {
	s := "start " + name
	for _, a := range attrs {
		s += " " + a.qualified() + "=" + a.Value
	}
	r.lines = append(r.lines, s)
}
func (r *plainRecorder) EndElement(name string) { r.lines = append(r.lines, "end "+name) }
func (r *plainRecorder) Characters(t string)    { r.lines = append(r.lines, "chars "+t) }
func (r *plainRecorder) Comment(t string)       { r.lines = append(r.lines, "comment "+t) }
func (r *plainRecorder) CdataBlock(t string)    { r.lines = append(r.lines, "cdata "+t) }
func (r *plainRecorder) ProcessingInstruction(target, data string) {
	r.lines = append(r.lines, "pi "+target+" "+data)
}

func eqStringSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	m := map[string]int{}
	for _, s := range a {
		m[s]++
	}
	for _, s := range b {
		m[s]--
	}
	for _, v := range m {
		if v != 0 {
			return false
		}
	}
	return true
}
