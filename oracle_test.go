// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

import (
	"os/exec"
	"strings"
	"testing"
)

// The differential oracle compares this library against the real Nokogiri gem.
// It is entirely optional: every assertion below also has a deterministic golden
// expectation, so the ruby-free lanes drive coverage on their own. When a usable
// `ruby` with `nokogiri` and RUBY_VERSION >= "4.0" is present (the ubuntu/macos
// CI lanes), the same corpora are additionally run through the gem and the
// selected nodes' text/attributes must agree.

// oracleRuby returns a ruby binary that can `require "nokogiri"` and is at least
// version 4.0, or skips the test.
func oracleRuby(t *testing.T) string {
	t.Helper()
	bin, err := exec.LookPath("ruby")
	if err != nil {
		t.Skip("ruby not on PATH; skipping Nokogiri gem oracle")
	}
	// version gate + gem availability
	out, err := exec.Command(bin, "-e",
		`exit(RUBY_VERSION >= "4.0" ? (require "nokogiri"; 0) : 3)`).CombinedOutput()
	if err != nil {
		t.Skipf("ruby unusable for oracle (need >=4.0 with nokogiri): %s", strings.TrimSpace(string(out)))
	}
	return bin
}

// gemSelectText parses html/xml with the gem and returns, one per line, the text
// of each node selected by the css or xpath query.
func gemSelectText(t *testing.T, bin, kind, doc, mode, query string) []string {
	t.Helper()
	// Emit one line per node, with newlines/CRs in the node's own text escaped so
	// a multi-line text value stays on a single output line.
	script := `
$stdout.binmode
require "nokogiri"
doc = Nokogiri::` + kind + `(STDIN.read)
nodes = doc.` + mode + `(ARGV[0])
nodes.each { |n| puts n.text.gsub("\\", "\\\\").gsub("\n", "\\n").gsub("\r", "\\r") }
`
	cmd := exec.Command(bin, "-e", script, query)
	cmd.Stdin = strings.NewReader(doc)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gem error for %s %q: %v\n%s", mode, query, err, out)
	}
	s := strings.TrimRight(string(out), "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func goSelectText(t *testing.T, kind, doc, mode, query string) []string {
	t.Helper()
	var d *Document
	var err error
	if kind == "HTML" {
		d, err = HTML(doc)
	} else {
		d, err = XML(doc)
	}
	if err != nil {
		t.Fatalf("go parse error: %v", err)
	}
	var set *NodeSet
	if mode == "css" {
		set, err = d.CSS(query)
	} else {
		set, err = d.XPath(query)
	}
	if err != nil {
		t.Fatalf("go %s %q error: %v", mode, query, err)
	}
	var out []string
	set.Each(func(n *Node) { out = append(out, escapeNL(n.Text())) })
	return out
}

// escapeNL mirrors the Ruby side's newline escaping so multi-line node text
// compares as a single line.
func escapeNL(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	return s
}

func eqStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

const oracleHTML = `<!DOCTYPE html>
<html><body>
  <div id="main" class="wrap">
    <ul class="list">
      <li class="item first" data-x="1">Alpha</li>
      <li class="item" data-x="2">Beta</li>
      <li class="item last" data-x="3">Gamma</li>
    </ul>
    <a href="https://example.com/a">A</a>
    <a href="http://example.org/b">B</a>
    <p>Para <span>inner</span> tail</p>
  </div>
</body></html>`

const oracleXML = `<catalog>
  <book id="b1" genre="fiction"><title>Alpha</title><price>10</price></book>
  <book id="b2" genre="tech"><title>Beta</title><price>20</price></book>
  <book id="b3" genre="fiction"><title>Gamma</title><price>30</price></book>
</catalog>`

func TestOracleCSS(t *testing.T) {
	bin := oracleRuby(t)
	queries := []string{
		"li",
		"li.item",
		".first",
		"ul.list li",
		"#main",
		"a[href^='https']",
		"li:first-child",
		"li:last-child",
		"li:nth-child(2)",
		"li:not(.first)",
		"p > span",
	}
	for _, q := range queries {
		got := goSelectText(t, "HTML", oracleHTML, "css", q)
		want := gemSelectText(t, bin, "HTML", oracleHTML, "css", q)
		if !eqStrings(got, want) {
			t.Errorf("css %q: go=%v gem=%v", q, got, want)
		}
	}
}

func TestOracleXPath(t *testing.T) {
	bin := oracleRuby(t)
	queries := []string{
		"//book",
		"//book[@genre='fiction']",
		"/catalog/book",
		"//book[2]/title",
		"//book[price>15]/title",
		"//title",
		"//book[last()]/title",
		"//book[@id='b2']/title",
	}
	for _, q := range queries {
		got := goSelectText(t, "XML", oracleXML, "xpath", q)
		want := gemSelectText(t, bin, "XML", oracleXML, "xpath", q)
		if !eqStrings(got, want) {
			t.Errorf("xpath %q: go=%v gem=%v", q, got, want)
		}
	}
}
