// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

import (
	"fmt"
	"strconv"
	"strings"
)

// cssTokKind classifies a CSS compound-selector token.
type cssTokKind int

const (
	cssType cssTokKind = iota
	cssUniversal
	cssID
	cssClass
	cssAttr   // val holds the body between [ ]
	cssPseudo // val holds the pseudo name plus any (arg)
	cssCombinator
)

// cssToken is a lexed CSS token.
type cssToken struct {
	kind cssTokKind
	val  string
}

// tokenizeCSS lexes one complex selector into a flat token list where compound
// pieces are adjacent and combinators separate them.
func tokenizeCSS(sel string) ([]cssToken, error) {
	var toks []cssToken
	i := 0
	n := len(sel)
	pendingSpace := false
	emit := func(t cssToken) { toks = append(toks, t) }

	for i < n {
		c := sel[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			pendingSpace = true
			i++
		case c == '>' || c == '+' || c == '~':
			emit(cssToken{cssCombinator, string(c)})
			pendingSpace = false
			i++
		default:
			if pendingSpace && len(toks) > 0 && toks[len(toks)-1].kind != cssCombinator {
				emit(cssToken{cssCombinator, " "})
			}
			pendingSpace = false
			adv, err := lexSimple(sel, i, emit)
			if err != nil {
				return nil, err
			}
			i = adv
		}
	}
	return toks, nil
}

// lexSimple lexes a single simple selector at position i and returns the new
// position.
func lexSimple(sel string, i int, emit func(cssToken)) (int, error) {
	c := sel[i]
	switch c {
	case '*':
		emit(cssToken{cssUniversal, "*"})
		return i + 1, nil
	case '#':
		id, j := readIdent(sel, i+1)
		if id == "" {
			return 0, fmt.Errorf("css: empty id at %q", sel[i:])
		}
		emit(cssToken{cssID, id})
		return j, nil
	case '.':
		cl, j := readIdent(sel, i+1)
		if cl == "" {
			return 0, fmt.Errorf("css: empty class at %q", sel[i:])
		}
		emit(cssToken{cssClass, cl})
		return j, nil
	case '[':
		j := i + 1
		depth := 1
		for j < len(sel) && depth > 0 {
			if sel[j] == '[' {
				depth++
			} else if sel[j] == ']' {
				depth--
				if depth == 0 {
					break
				}
			}
			j++
		}
		if depth != 0 {
			return 0, fmt.Errorf("css: unterminated attribute selector")
		}
		emit(cssToken{cssAttr, sel[i+1 : j]})
		return j + 1, nil
	case ':':
		start := i + 1
		if start < len(sel) && sel[start] == ':' {
			start++ // treat ::pseudo-element leniently as a pseudo
		}
		name, j := readIdent(sel, start)
		if name == "" {
			return 0, fmt.Errorf("css: empty pseudo at %q", sel[i:])
		}
		// capture (...) argument
		if j < len(sel) && sel[j] == '(' {
			depth := 1
			k := j + 1
			for k < len(sel) && depth > 0 {
				switch sel[k] {
				case '(':
					depth++
				case ')':
					depth--
				}
				k++
			}
			emit(cssToken{cssPseudo, name + sel[j:k]})
			return k, nil
		}
		emit(cssToken{cssPseudo, name})
		return j, nil
	default:
		name, j := readIdent(sel, i)
		if name == "" {
			return 0, fmt.Errorf("css: unexpected character %q", string(c))
		}
		emit(cssToken{cssType, name})
		return j, nil
	}
}

// readIdent reads a CSS identifier (letters, digits, -, _) starting at i. A
// namespace prefix in CSS uses "|" (handled separately), not ":".
func readIdent(sel string, i int) (string, int) {
	start := i
	for i < len(sel) {
		c := sel[i]
		if c == '-' || c == '_' ||
			(c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c >= 0x80 {
			i++
			continue
		}
		break
	}
	return sel[start:i], i
}

// unquoteCSS strips surrounding single or double quotes from an attribute value.
func unquoteCSS(s string) string {
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') && s[len(s)-1] == s[0] {
		return s[1 : len(s)-1]
	}
	return s
}

// splitTopLevel splits s on sep at bracket/paren depth zero.
func splitTopLevel(s string, sep byte) []string {
	var out []string
	depth := 0
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '[', '(':
			depth++
		case ']', ')':
			depth--
		case sep:
			if depth == 0 {
				out = append(out, s[start:i])
				start = i + 1
			}
		}
	}
	out = append(out, s[start:])
	return out
}

// parseAnB parses a CSS An+B microsyntax argument into (a, b).
func parseAnB(arg string) (a, b int, ok bool) {
	arg = strings.ReplaceAll(arg, " ", "")
	if arg == "" {
		return 0, 0, false
	}
	if !strings.Contains(arg, "n") {
		v, err := strconv.Atoi(arg)
		if err != nil {
			return 0, 0, false
		}
		return 0, v, true
	}
	parts := strings.SplitN(arg, "n", 2)
	coeff := parts[0]
	switch coeff {
	case "", "+":
		a = 1
	case "-":
		a = -1
	default:
		v, err := strconv.Atoi(coeff)
		if err != nil {
			return 0, 0, false
		}
		a = v
	}
	if parts[1] == "" {
		return a, 0, true
	}
	v, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, false
	}
	return a, v, true
}
