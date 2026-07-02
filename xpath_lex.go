// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// tokKind enumerates XPath token categories.
type tokKind int

const (
	tEOF tokKind = iota
	tName
	tNumber
	tLiteral // quoted string
	tOp      // operator / punctuation
	tAxis    // NCName followed by "::"
	tFunc    // NCName followed by "("
	tVar     // $name
)

// token is a single lexed XPath token.
type token struct {
	kind tokKind
	val  string
}

// xpathLexer turns an XPath expression string into a token slice. XPath lexing is
// context sensitive: a name is a function call if immediately followed by "(",
// an axis if followed by "::", and the operator keywords (and/or/div/mod/*) are
// only operators when an operator is expected. We resolve those with a small
// look-behind on the previous significant token.
type xpathLexer struct {
	src  string
	pos  int
	toks []token
	err  error
}

// lexXPath tokenizes expr, returning the tokens or an error.
func lexXPath(expr string) ([]token, error) {
	l := &xpathLexer{src: expr}
	l.run()
	if l.err != nil {
		return nil, l.err
	}
	return l.toks, nil
}

func (l *xpathLexer) run() {
	for l.err == nil {
		l.skipSpace()
		if l.pos >= len(l.src) {
			return
		}
		c := l.src[l.pos]
		switch {
		case c == '"' || c == '\'':
			l.lexString(c)
		case c >= '0' && c <= '9':
			l.lexNumber()
		case c == '.' && l.pos+1 < len(l.src) && l.src[l.pos+1] >= '0' && l.src[l.pos+1] <= '9':
			l.lexNumber()
		case c == '$':
			l.lexVar()
		case isNameStart(rune(c)) || c >= 0x80:
			l.lexName()
		default:
			l.lexOp()
		}
	}
}

func (l *xpathLexer) skipSpace() {
	for l.pos < len(l.src) {
		switch l.src[l.pos] {
		case ' ', '\t', '\n', '\r':
			l.pos++
		default:
			return
		}
	}
}

func (l *xpathLexer) lexString(q byte) {
	l.pos++ // opening quote
	start := l.pos
	for l.pos < len(l.src) && l.src[l.pos] != q {
		l.pos++
	}
	if l.pos >= len(l.src) {
		l.err = fmt.Errorf("xpath: unterminated string literal")
		return
	}
	l.toks = append(l.toks, token{tLiteral, l.src[start:l.pos]})
	l.pos++ // closing quote
}

func (l *xpathLexer) lexNumber() {
	start := l.pos
	for l.pos < len(l.src) && ((l.src[l.pos] >= '0' && l.src[l.pos] <= '9') || l.src[l.pos] == '.') {
		l.pos++
	}
	l.toks = append(l.toks, token{tNumber, l.src[start:l.pos]})
}

func (l *xpathLexer) lexVar() {
	l.pos++ // $
	start := l.pos
	for l.pos < len(l.src) && isNameChar(rune(l.src[l.pos])) {
		l.pos++
	}
	l.toks = append(l.toks, token{tVar, l.src[start:l.pos]})
}

func (l *xpathLexer) lexName() {
	start := l.pos
	for l.pos < len(l.src) {
		r, sz := utf8.DecodeRuneInString(l.src[l.pos:])
		if r == ':' {
			// A single ':' joins a QName prefix to its local part, but "::"
			// terminates the name (it introduces an axis specifier).
			if l.pos+1 < len(l.src) && l.src[l.pos+1] == ':' {
				break
			}
			l.pos += sz
			continue
		}
		if isNameChar(r) {
			l.pos += sz
			continue
		}
		break
	}
	name := l.src[start:l.pos]
	// Look ahead past spaces for "::" (axis) or "(" (function).
	save := l.pos
	l.skipSpace()
	rest := l.src[l.pos:]
	switch {
	case strings.HasPrefix(rest, "::"):
		l.pos += 2
		l.toks = append(l.toks, token{tAxis, name})
		return
	case strings.HasPrefix(rest, "(") && isFunctionName(name):
		// leave "(" for the op lexer; classify name as function
		l.pos = save
		l.toks = append(l.toks, token{tFunc, name})
		return
	default:
		l.pos = save
	}
	// Operator keyword when an operator is expected.
	if l.opExpected() && (name == "and" || name == "or" || name == "div" || name == "mod") {
		l.toks = append(l.toks, token{tOp, name})
		return
	}
	l.toks = append(l.toks, token{tName, name})
}

// isFunctionName rejects the node-test keywords that look like calls but are node
// tests (node(), text(), comment(), processing-instruction()); those are still
// lexed as functions here and disambiguated by the parser via context.
func isFunctionName(string) bool { return true }

// opExpected reports whether the previous significant token means the next name
// keyword is an operator (per the XPath lexer disambiguation rules).
func (l *xpathLexer) opExpected() bool {
	if len(l.toks) == 0 {
		return false
	}
	prev := l.toks[len(l.toks)-1]
	switch prev.kind {
	case tName, tNumber, tLiteral, tVar:
		return true
	case tOp:
		switch prev.val {
		case ")", "]", "*":
			return true
		}
	}
	return false
}

func (l *xpathLexer) lexOp() {
	rest := l.src[l.pos:]
	for _, op := range []string{"//", "!=", "<=", ">=", "::", ".."} {
		if strings.HasPrefix(rest, op) {
			l.toks = append(l.toks, token{tOp, op})
			l.pos += len(op)
			return
		}
	}
	c := l.src[l.pos]
	switch c {
	case '/', '(', ')', '[', ']', '@', ',', '.', '|', '+', '-', '=', '<', '>', '*':
		l.toks = append(l.toks, token{tOp, string(c)})
		l.pos++
	default:
		l.err = fmt.Errorf("xpath: unexpected character %q", string(c))
	}
}

func isNameStart(r rune) bool {
	return r == '_' || unicode.IsLetter(r)
}

func isNameChar(r rune) bool {
	return r == '_' || r == '-' || r == '.' || unicode.IsLetter(r) || unicode.IsDigit(r)
}
