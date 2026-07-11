<p align="center"><img src="https://raw.githubusercontent.com/go-ruby-nokogiri/brand/main/social/go-ruby-nokogiri-nokogiri.png" alt="go-ruby-nokogiri/nokogiri" width="720"></p>

# nokogiri — go-ruby-nokogiri

[![Docs](https://img.shields.io/badge/docs-mkdocs--material-DC2626)](https://go-ruby-nokogiri.github.io/docs/)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.26.4%2B-00ADD8)](https://go.dev/dl/)
[![Coverage](https://img.shields.io/badge/coverage-100%25-1a7f37)](#tests--coverage)

**A pure-Go (no cgo) reimplementation of the core of Ruby's
[Nokogiri](https://nokogiri.org/) HTML/XML toolkit.** Upstream Nokogiri is a C
extension over **libxml2** and **libxslt**; this library instead builds on the
pure-Go [`golang.org/x/net/html`](https://pkg.go.dev/golang.org/x/net/html)
tag-soup parser (for `Nokogiri::HTML`) and Go's `encoding/xml` (for
`Nokogiri::XML`), exposes a single mutable **Node** tree over both, and layers a
full **XPath 1.0** engine plus a **CSS-selector → XPath** compiler on top — so
`css` / `at_css` / `xpath` / `at_xpath` behave as Ruby programs expect, **with
`CGO_ENABLED=0`** on every supported platform.

It is the HTML/XML backend for
[go-embedded-ruby](https://github.com/go-embedded-ruby/ruby), but is a
**standalone, reusable** Go module with no dependency on the Ruby runtime — a
sibling of [go-ruby-rexml](https://github.com/go-ruby-rexml/rexml) (the pure-Ruby
REXML parser). Where REXML implements a small XML DOM with an XPath *subset*,
this library targets the *Nokogiri* API surface and a fuller XPath 1.0.

## Architecture

Four layers, deliberately kept independent:

1. **HTML5 parser** — `Nokogiri::HTML(str)` runs the lenient WHATWG HTML5
   tree-building algorithm via `x/net/html`. Real-world "tag soup" is recovered
   exactly as a browser would: implied `<html>/<head>/<body>`, inferred end tags,
   misnested-tag correction. This is Nokogiri's #1 use.
2. **XML parser** — `Nokogiri::XML(str)` parses well-formed XML via
   `encoding/xml` (RawToken mode), resolving namespace prefixes to URIs while
   preserving the literal prefix for round-tripping, and recovering CDATA node
   types that `encoding/xml` otherwise collapses into text.
3. **Shared `Node` tree** — one doubly-linked, mutable DOM (`Node` / `NodeSet` /
   `Attr`) produced by **both** parsers and consumed by everything above.
4. **XPath 1.0 + CSS** — a from-scratch XPath 1.0 engine (all 13 axes, node
   tests, predicates, and the core function library) and a CSS-selector→XPath
   translator, both operating on the shared tree.

## Features

- **Parse:** `Nokogiri::HTML` / `Nokogiri::HTML5` / `Nokogiri::XML` / HTML
  fragments → `Document`. XML declarations are recognised (version/encoding
  preserved for round-tripping; ASCII-compatible non-UTF-8 declarations parse
  structurally — see the deferred note on charset transcoding).
- **Navigate:** `children` / `element_children` / `parent` / `next` / `previous`
  / `next_element` / `previous_element` / `root`.
- **Query:** `css` / `at_css` / `xpath` / `at_xpath` on `Node`, `Document`, and
  `NodeSet`; raw scalar XPath results (`count()`, `string()`, …) via `EvalXPath`.
- **CSS selectors:** type / `*` / `#id` / `.class`; `[attr]` and
  `[attr=val]` with the `~= |= ^= $= *=` variants; descendant / child (`>`) /
  adjacent (`+`) / general-sibling (`~`) combinators; structural pseudo-classes
  `:first-child` `:last-child` `:only-child` `:empty` `:root`
  `:nth-child(An+B|odd|even)` `:nth-last-child` `:first-of-type` `:last-of-type`
  `:nth-of-type` `:only-of-type` `:not(...)`; selector lists (`a, b`).
- **XPath 1.0:** axes `child descendant descendant-or-self self parent ancestor
  ancestor-or-self following-sibling preceding-sibling following preceding
  attribute namespace`; node tests `node() text() comment()
  processing-instruction()` and name tests (with namespace prefixes); predicates
  with position/`last()`; the core library (`last position count id local-name
  name namespace-uri string concat starts-with contains substring-before
  substring-after substring string-length normalize-space translate boolean not
  true false lang number sum floor ceiling round`, plus `current()`); the full
  operator set (`or and = != < <= > >= + - * div mod |`).
- **Text & serialization:** `text` / `content` / `inner_html` / `inner_xml` /
  `to_html` / `to_xml` / `to_s`. `to_xml` reproduces **libxml2's exact
  pretty-printing** — the two-space indent, the XML declaration + trailing
  newline at the document level, and the *sticky-downward* rule that leaves a
  subtree inline once any level holds character data — all pinned byte-for-byte
  against the gem. `to_html` follows **WHATWG/HTML5 serialization** (void
  elements without `/`, raw-text `script`/`style` left unescaped, empty
  non-void elements as `<x></x>`, no added whitespace). Correct entity escaping;
  `[]` / `attribute` / `set_attribute` / `remove_attribute` / `name` /
  node-type predicates.
- **Namespaces:** `namespaces` (full in-scope map, nearest-wins),
  `namespace` (the node's own prefix/URI), and `add_namespace_definition`
  (`AddNamespace`) with live re-resolution of the affected subtree.
- **Streaming SAX:** `Nokogiri::XML::SAX::Parser` / `SAX::Document` — a
  `SAXHandler` interface (embed `SAXDocument` for no-op defaults) driven over
  `Parse`/`ParseReader`, firing `start_document` / `end_document` /
  `start_element` (attributes including `xmlns` in source order) / `end_element` /
  `characters` / `comment` / `cdata_block` / `processing_instruction` / `error`,
  with the event stream pinned against the gem's SAX parser.
- **Build & mutate:** `Nokogiri::XML::Builder`-style programmatic construction;
  `add_child` / `prepend` / `add_next_sibling` / `add_previous_sibling` /
  `remove` / `replace` / `wrap` / `content=`.

## What it is — and isn't (deferred, documented honestly)

The following are **not** implemented and are called out so nothing is a silent
gap. They are the parts of Nokogiri that go well beyond parse + query + serialize
+ stream, or that depend on libxslt/libxml2 subsystems:

- **XSLT** (`Nokogiri::XSLT`) — not built in here; a pure-Go XSLT 1.0 processor
  lives in the sibling module
  [go-ruby-xslt/xslt](https://github.com/go-ruby-xslt/xslt), which drives this
  library's XPath engine through the `XPathContext` extension seam
  (`Node.EvalXPathCtx`).
- **Schema validation** — no DTD / RelaxNG / XSD (`Nokogiri::XML::Schema`,
  `RelaxNG`) validation. These are large, self-contained validators that in the
  reference implementation *are* libxml2's schema engines; reproducing them in
  pure Go is out of scope for this module and is named, not half-shipped.
- **Pull `Reader`** — `Nokogiri::XML::Reader` (cursor-style pull parsing) is not
  provided; use the event-driven `SAX::Parser` (implemented) or the DOM.
- **HTML SAX** — `Nokogiri::HTML4::SAX::Parser` is not provided; SAX streaming is
  XML-only. (HTML is still fully supported via the DOM parser.)
- **Exact libxml2 error recovery / diagnostics** — malformed input is reported
  through the `error` callback / a returned error, but the *text* of libxml2's
  messages (e.g. "Opening and ending tag mismatch: … line N") and its precise
  fault-tolerant recovery of broken XML are genuinely libxml2 behaviour and are
  not reproduced.
- **Charset transcoding** — a non-UTF-8 XML declaration is honoured for
  round-tripping (the encoding is preserved on `to_xml`) and ASCII-compatible
  encodings parse structurally, but the byte stream is **not transcoded** to
  UTF-8 the way libxml2 does; genuinely non-ASCII-compatible encodings are not
  supported.

The focus is squarely **parse → CSS/XPath → navigate → serialize → stream (SAX)**,
which covers the overwhelming majority of scraping and document-processing code.

## Install

```sh
go get github.com/go-ruby-nokogiri/nokogiri
```

## Usage

```go
package main

import (
	"fmt"

	nokogiri "github.com/go-ruby-nokogiri/nokogiri"
)

func main() {
	doc, _ := nokogiri.HTML(`<html><body>
	  <ul class="list">
	    <li class="item">Alpha</li>
	    <li class="item">Beta</li>
	  </ul>
	  <a href="https://example.com">link</a>
	</body></html>`)

	// CSS
	doc.CSS("ul.list li.item")             // NodeSet of the two <li>
	first, _ := doc.AtCSS("li.item")       // -> <li>Alpha</li>
	fmt.Println(first.Text())              // "Alpha"

	// XPath 1.0
	set, _ := doc.XPath("//a[starts-with(@href,'https')]")
	fmt.Println(set.First().Attribute("href")) // "https://example.com"

	n, _ := doc.Node.EvalXPath("count(//li)", nil)
	fmt.Println(n) // 2

	// Build
	b := nokogiri.NewBuilder()
	b.Element("catalog", func(b *nokogiri.Builder) {
		b.Element("book", func(b *nokogiri.Builder) {
			b.Attr("id", "b1")
			b.ElementText("title", "Alpha")
		})
	})
	fmt.Println(b.ToXML())
	// <catalog><book id="b1"><title>Alpha</title></book></catalog>
}
```

## Tests & coverage

The suite holds **100.0% statement coverage** with **zero** dependency on a Ruby
runtime — the deterministic, golden-vector tests alone drive the gate, so the
Windows and cross-arch (qemu) CI lanes pass without `ruby` installed. A separate
differential **oracle** compares parse + `css`/`xpath` results, `to_xml`/`to_html`
serialized bytes, `namespaces`, and the `SAX` event stream against the real
**`nokogiri` gem** on the ubuntu/macos lanes (skipped where `ruby` is absent, and
version-gated to `RUBY_VERSION >= "4.0"`).

```sh
GOWORK=off go test -race -cover ./...
```

CI validates on **3 OSes** (Linux/macOS/Windows) and **6 64-bit architectures**
(amd64, arm64, riscv64, loong64, ppc64le, s390x). The host `-race` lane keeps
cgo enabled; the architecture lanes build with `CGO_ENABLED=0`.

## License

BSD-3-Clause © the go-ruby-nokogiri/nokogiri authors. See [LICENSE](LICENSE).

## WebAssembly

Being pure Go (CGO=0), this library also compiles to **WebAssembly** — both
`GOOS=js GOARCH=wasm` (browser / Node.js) and `GOOS=wasip1 GOARCH=wasm` (WASI).
CI builds both targets on every push, alongside the six 64-bit native/qemu arches.

```sh
GOOS=js     GOARCH=wasm go build ./...   # browser / Node
GOOS=wasip1 GOARCH=wasm go build ./...   # WASI (wasmtime, wasmer, wasmedge, …)
```
