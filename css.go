// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

import (
	"fmt"
	"strings"
)

// cssToXPath translates a (possibly comma-separated) CSS selector list into an
// equivalent XPath 1.0 expression. prefix is the axis prefix each selector is
// anchored to (".//" for a descendant search, "/" for an absolute one). The
// supported subset covers the Nokogiri scraping surface: type/universal
// selectors, #id, .class, [attr], [attr=val] and its ~=/^=/$=/*=/|= variants,
// descendant/child(>)/adjacent(+)/general-sibling(~) combinators, and the
// structural/positional pseudo-classes :first-child, :last-child, :nth-child(n),
// :nth-of-type(n), :only-child, :empty, :not(...), and :root.
func cssToXPath(selector, prefix string) (string, error) {
	groups := splitTopLevel(selector, ',')
	var parts []string
	for _, g := range groups {
		g = strings.TrimSpace(g)
		if g == "" {
			return "", fmt.Errorf("css: empty selector in %q", selector)
		}
		xp, err := compileComplex(g, prefix)
		if err != nil {
			return "", err
		}
		parts = append(parts, xp)
	}
	return strings.Join(parts, " | "), nil
}

// compileComplex compiles one complex selector (sequences joined by combinators).
func compileComplex(sel, prefix string) (string, error) {
	toks, err := tokenizeCSS(sel)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString(prefix)
	combinator := "" // pending combinator between sequences
	first := true
	i := 0
	for i < len(toks) {
		if toks[i].kind == cssCombinator {
			combinator = toks[i].val
			i++
			continue
		}
		// gather one compound sequence
		seqStart := i
		for i < len(toks) && toks[i].kind != cssCombinator {
			i++
		}
		seq := toks[seqStart:i]
		if !first {
			switch combinator {
			case ">":
				b.WriteString("/")
			case "+":
				b.WriteString("/following-sibling::*[1]/self::")
			case "~":
				b.WriteString("/following-sibling::")
			default: // descendant
				b.WriteString("//")
			}
		}
		step, err := compileSequence(seq)
		if err != nil {
			return "", err
		}
		b.WriteString(step)
		first = false
		combinator = ""
	}
	return b.String(), nil
}

// compileSequence compiles a compound selector (type + qualifiers) into an XPath
// step with predicates.
func compileSequence(toks []cssToken) (string, error) {
	elem := "*"
	var preds []string
	for _, t := range toks {
		switch t.kind {
		case cssType:
			elem = t.val
		case cssUniversal:
			elem = "*"
		case cssID:
			preds = append(preds, fmt.Sprintf("@id=%s", xpStr(t.val)))
		case cssClass:
			preds = append(preds, fmt.Sprintf("contains(concat(' ', normalize-space(@class), ' '), %s)", xpStr(" "+t.val+" ")))
		case cssAttr:
			p, err := compileAttr(t.val)
			if err != nil {
				return "", err
			}
			preds = append(preds, p)
		case cssPseudo:
			p, err := compilePseudo(t.val)
			if err != nil {
				return "", err
			}
			if p != "" {
				preds = append(preds, p)
			}
		}
	}
	s := elem
	for _, p := range preds {
		s += "[" + p + "]"
	}
	return s, nil
}

// compileAttr compiles an [attr...] qualifier body (without the brackets).
func compileAttr(body string) (string, error) {
	body = strings.TrimSpace(body)
	for _, op := range []string{"~=", "|=", "^=", "$=", "*=", "="} {
		if idx := strings.Index(body, op); idx >= 0 {
			name := strings.TrimSpace(body[:idx])
			val := unquoteCSS(strings.TrimSpace(body[idx+len(op):]))
			at := "@" + name
			switch op {
			case "=":
				return fmt.Sprintf("%s=%s", at, xpStr(val)), nil
			case "~=":
				return fmt.Sprintf("contains(concat(' ', normalize-space(%s), ' '), %s)", at, xpStr(" "+val+" ")), nil
			case "^=":
				return fmt.Sprintf("starts-with(%s, %s)", at, xpStr(val)), nil
			case "$=":
				return fmt.Sprintf("substring(%s, string-length(%s) - %d) = %s", at, at, len(val)-1, xpStr(val)), nil
			case "*=":
				return fmt.Sprintf("contains(%s, %s)", at, xpStr(val)), nil
			case "|=":
				return fmt.Sprintf("(%s=%s or starts-with(%s, %s))", at, xpStr(val), at, xpStr(val+"-")), nil
			}
		}
	}
	// bare [attr] presence test
	return "@" + strings.TrimSpace(body), nil
}

// compilePseudo compiles a :pseudo-class selector.
func compilePseudo(p string) (string, error) {
	name := p
	arg := ""
	if i := strings.IndexByte(p, '('); i >= 0 {
		name = p[:i]
		arg = strings.TrimSuffix(p[i+1:], ")")
	}
	switch name {
	case "first-child":
		return "count(preceding-sibling::*)=0", nil
	case "last-child":
		return "count(following-sibling::*)=0", nil
	case "only-child":
		return "count(preceding-sibling::*)=0 and count(following-sibling::*)=0", nil
	case "empty":
		return "not(node())", nil
	case "root":
		return "not(parent::*)", nil
	case "nth-child":
		return nthExpr("preceding-sibling::*", arg)
	case "nth-of-type":
		// positional counting only same-name preceding siblings
		return nthExpr("preceding-sibling::*[name()=name(current())]", arg)
	case "nth-last-child":
		return nthExpr("following-sibling::*", arg)
	case "first-of-type":
		return "count(preceding-sibling::*[name()=name(current())])=0", nil
	case "last-of-type":
		return "count(following-sibling::*[name()=name(current())])=0", nil
	case "only-of-type":
		return "count(preceding-sibling::*[name()=name(current())])=0 and count(following-sibling::*[name()=name(current())])=0", nil
	case "not":
		inner, err := compileNotArg(arg)
		if err != nil {
			return "", err
		}
		return "not(" + inner + ")", nil
	default:
		return "", fmt.Errorf("css: unsupported pseudo-class :%s", name)
	}
}

// compileNotArg compiles the argument of :not(...) into a boolean predicate that
// is true when the context node matches the inner simple selector.
func compileNotArg(arg string) (string, error) {
	toks, err := tokenizeCSS(strings.TrimSpace(arg))
	if err != nil {
		return "", err
	}
	elem := ""
	var preds []string
	for _, t := range toks {
		switch t.kind {
		case cssType:
			elem = t.val
		case cssUniversal:
			elem = "*"
		case cssID:
			preds = append(preds, fmt.Sprintf("@id=%s", xpStr(t.val)))
		case cssClass:
			preds = append(preds, fmt.Sprintf("contains(concat(' ', normalize-space(@class), ' '), %s)", xpStr(" "+t.val+" ")))
		case cssAttr:
			p, err := compileAttr(t.val)
			if err != nil {
				return "", err
			}
			preds = append(preds, p)
		case cssPseudo:
			p, err := compilePseudo(t.val)
			if err != nil {
				return "", err
			}
			if p != "" {
				preds = append(preds, p)
			}
		default:
			return "", fmt.Errorf("css: :not() does not support combinators")
		}
	}
	if elem != "" && elem != "*" {
		preds = append([]string{"self::" + elem}, preds...)
	}
	if len(preds) == 0 {
		return "true()", nil
	}
	return "(" + strings.Join(preds, " and ") + ")", nil
}

// nthExpr builds an XPath predicate for :nth-child style arguments (an+b, odd,
// even, or a literal integer), counting via the given preceding-sibling axis.
func nthExpr(axis, arg string) (string, error) {
	arg = strings.TrimSpace(arg)
	pos := "count(" + axis + ")+1"
	switch arg {
	case "odd":
		return "(" + pos + ") mod 2 = 1", nil
	case "even":
		return "(" + pos + ") mod 2 = 0", nil
	}
	a, b, ok := parseAnB(arg)
	if !ok {
		return "", fmt.Errorf("css: bad nth argument %q", arg)
	}
	if a == 0 {
		return fmt.Sprintf("(%s) = %d", pos, b), nil
	}
	// (pos - b) is divisible by a and same sign
	return fmt.Sprintf("((%s) - %d) mod %d = 0 and ((%s) - %d) div %d >= 0", pos, b, a, pos, b, a), nil
}

// xpStr renders a Go string as an XPath string literal, using concat() when the
// value contains both quote kinds.
func xpStr(s string) string {
	if !strings.Contains(s, "'") {
		return "'" + s + "'"
	}
	if !strings.Contains(s, `"`) {
		return `"` + s + `"`
	}
	parts := strings.Split(s, "'")
	var b strings.Builder
	b.WriteString("concat(")
	for i, p := range parts {
		if i > 0 {
			b.WriteString(`, "'", `)
		}
		b.WriteString("'" + p + "'")
	}
	b.WriteString(")")
	return b.String()
}
