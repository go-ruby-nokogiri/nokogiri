// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

import (
	"fmt"
	"strconv"
)

// xpathParser is a recursive-descent parser for XPath 1.0 following the grammar's
// precedence: Or > And > Equality > Relational > Additive > Multiplicative >
// Unary > Union > Path.
type xpathParser struct {
	toks []token
	pos  int
}

// parseXPath compiles an XPath 1.0 expression into an AST.
func parseXPath(expr string) (e expr, err error) {
	toks, err := lexXPath(expr)
	if err != nil {
		return nil, err
	}
	p := &xpathParser{toks: toks}
	defer func() {
		if r := recover(); r != nil {
			if pe, ok := r.(parseError); ok {
				e, err = nil, error(pe)
				return
			}
			panic(r)
		}
	}()
	e = p.parseExpr()
	if p.pos != len(p.toks) {
		return nil, fmt.Errorf("xpath: trailing tokens near %q", p.cur().val)
	}
	return e, nil
}

type parseError string

func (e parseError) Error() string { return string(e) }

func (p *xpathParser) fail(format string, a ...any) {
	panic(parseError("xpath: " + fmt.Sprintf(format, a...)))
}

func (p *xpathParser) cur() token {
	if p.pos >= len(p.toks) {
		return token{tEOF, ""}
	}
	return p.toks[p.pos]
}

func (p *xpathParser) isOp(v string) bool {
	t := p.cur()
	return t.kind == tOp && t.val == v
}

func (p *xpathParser) eatOp(v string) {
	if !p.isOp(v) {
		p.fail("expected %q, got %q", v, p.cur().val)
	}
	p.pos++
}

func (p *xpathParser) parseExpr() expr { return p.parseOr() }

func (p *xpathParser) parseOr() expr {
	l := p.parseAnd()
	for p.cur().kind == tOp && p.cur().val == "or" {
		p.pos++
		l = &binaryExpr{"or", l, p.parseAnd()}
	}
	return l
}

func (p *xpathParser) parseAnd() expr {
	l := p.parseEquality()
	for p.cur().kind == tOp && p.cur().val == "and" {
		p.pos++
		l = &binaryExpr{"and", l, p.parseEquality()}
	}
	return l
}

func (p *xpathParser) parseEquality() expr {
	l := p.parseRelational()
	for p.isOp("=") || p.isOp("!=") {
		op := p.cur().val
		p.pos++
		l = &binaryExpr{op, l, p.parseRelational()}
	}
	return l
}

func (p *xpathParser) parseRelational() expr {
	l := p.parseAdditive()
	for p.isOp("<") || p.isOp(">") || p.isOp("<=") || p.isOp(">=") {
		op := p.cur().val
		p.pos++
		l = &binaryExpr{op, l, p.parseAdditive()}
	}
	return l
}

func (p *xpathParser) parseAdditive() expr {
	l := p.parseMultiplicative()
	for p.isOp("+") || p.isOp("-") {
		op := p.cur().val
		p.pos++
		l = &binaryExpr{op, l, p.parseMultiplicative()}
	}
	return l
}

func (p *xpathParser) parseMultiplicative() expr {
	l := p.parseUnary()
	for p.isOp("*") || (p.cur().kind == tOp && (p.cur().val == "div" || p.cur().val == "mod")) {
		op := p.cur().val
		p.pos++
		l = &binaryExpr{op, l, p.parseUnary()}
	}
	return l
}

func (p *xpathParser) parseUnary() expr {
	if p.isOp("-") {
		p.pos++
		return &unaryExpr{p.parseUnary()}
	}
	return p.parseUnion()
}

func (p *xpathParser) parseUnion() expr {
	l := p.parsePath()
	for p.isOp("|") {
		p.pos++
		l = &binaryExpr{"|", l, p.parsePath()}
	}
	return l
}

// parsePath parses a location path or a filter expression followed by a relative
// path.
func (p *xpathParser) parsePath() expr {
	// Absolute paths.
	if p.isOp("/") || p.isOp("//") {
		return p.parseLocationPath(true)
	}
	// A primary expression that is not a name/axis/@ begins a FilterExpr.
	t := p.cur()
	if t.kind == tLiteral || t.kind == tNumber || t.kind == tVar || t.kind == tFunc || p.isOp("(") {
		prim := p.parsePrimary()
		// Predicates directly on the primary.
		var preds []expr
		for p.isOp("[") {
			p.pos++
			preds = append(preds, p.parseExpr())
			p.eatOp("]")
		}
		if len(preds) > 0 {
			prim = &pathExpr{filter: prim, steps: []step{{axis: axSelf, test: ntNode, predicates: preds}}}
		}
		if p.isOp("/") || p.isOp("//") {
			pe := p.parseLocationPath(false)
			lp := pe.(*pathExpr)
			lp.filter = prim
			return lp
		}
		return prim
	}
	return p.parseLocationPath(false)
}

// parseLocationPath parses a (relative or absolute) sequence of steps.
func (p *xpathParser) parseLocationPath(absolute bool) expr {
	lp := &pathExpr{rooted: absolute}
	if absolute {
		if p.isOp("//") {
			p.pos++
			lp.steps = append(lp.steps, step{axis: axDescendantOrSelf, test: ntNode})
			lp.steps = append(lp.steps, p.parseStep())
		} else {
			p.pos++ // "/"
			// "/" alone is the root node-set; a step may follow.
			if p.startsStep() {
				lp.steps = append(lp.steps, p.parseStep())
			}
		}
	} else {
		lp.steps = append(lp.steps, p.parseStep())
	}
	for {
		if p.isOp("//") {
			p.pos++
			lp.steps = append(lp.steps, step{axis: axDescendantOrSelf, test: ntNode})
			lp.steps = append(lp.steps, p.parseStep())
		} else if p.isOp("/") {
			p.pos++
			lp.steps = append(lp.steps, p.parseStep())
		} else {
			break
		}
	}
	return lp
}

// startsStep reports whether the current token can begin a step.
func (p *xpathParser) startsStep() bool {
	t := p.cur()
	switch t.kind {
	case tName, tAxis, tFunc:
		return true
	case tOp:
		return t.val == "@" || t.val == "." || t.val == ".." || t.val == "*"
	}
	return false
}

func (p *xpathParser) parseStep() step {
	var s step
	s.axis = axChild

	switch {
	case p.isOp("."):
		p.pos++
		s.axis = axSelf
		s.test = ntNode
		p.parsePredicates(&s)
		return s
	case p.isOp(".."):
		p.pos++
		s.axis = axParent
		s.test = ntNode
		p.parsePredicates(&s)
		return s
	case p.isOp("@"):
		p.pos++
		s.axis = axAttribute
	case p.cur().kind == tAxis:
		s.axis = axisByName(p.cur().val, p)
		p.pos++
	}

	p.parseNodeTest(&s)
	p.parsePredicates(&s)
	return s
}

func (p *xpathParser) parseNodeTest(s *step) {
	t := p.cur()
	switch {
	case t.kind == tFunc:
		// node type tests
		switch t.val {
		case "node":
			s.test = ntNode
		case "text":
			s.test = ntText
		case "comment":
			s.test = ntComment
		case "processing-instruction":
			s.test = ntPI
		default:
			p.fail("unexpected function %q in node test", t.val)
		}
		p.pos++
		p.eatOp("(")
		if s.test == ntPI && p.cur().kind == tLiteral {
			s.name = p.cur().val
			p.pos++
		}
		p.eatOp(")")
	case p.isOp("*"):
		s.test = ntAny
		p.pos++
	case t.kind == tName:
		s.test = ntName
		prefix, local := splitQName(t.val)
		s.prefix, s.name = prefix, local
		p.pos++
	default:
		p.fail("expected node test, got %q", t.val)
	}
}

func (p *xpathParser) parsePredicates(s *step) {
	for p.isOp("[") {
		p.pos++
		s.predicates = append(s.predicates, p.parseExpr())
		p.eatOp("]")
	}
}

func (p *xpathParser) parsePrimary() expr {
	t := p.cur()
	switch t.kind {
	case tLiteral:
		p.pos++
		return &stringLit{t.val}
	case tNumber:
		p.pos++
		f, err := strconv.ParseFloat(t.val, 64)
		if err != nil {
			p.fail("bad number %q", t.val)
		}
		return &numberLit{f}
	case tVar:
		p.pos++
		return &varRef{t.val}
	case tFunc:
		return p.parseFuncCall()
	case tOp:
		if t.val == "(" {
			p.pos++
			e := p.parseExpr()
			p.eatOp(")")
			return e
		}
	}
	p.fail("unexpected token %q", t.val)
	return nil
}

func (p *xpathParser) parseFuncCall() expr {
	name := p.cur().val
	p.pos++
	p.eatOp("(")
	fc := &funcCall{name: name}
	if !p.isOp(")") {
		fc.args = append(fc.args, p.parseExpr())
		for p.isOp(",") {
			p.pos++
			fc.args = append(fc.args, p.parseExpr())
		}
	}
	p.eatOp(")")
	return fc
}

func axisByName(name string, p *xpathParser) axis {
	switch name {
	case "child":
		return axChild
	case "descendant":
		return axDescendant
	case "descendant-or-self":
		return axDescendantOrSelf
	case "self":
		return axSelf
	case "parent":
		return axParent
	case "ancestor":
		return axAncestor
	case "ancestor-or-self":
		return axAncestorOrSelf
	case "following-sibling":
		return axFollowingSibling
	case "preceding-sibling":
		return axPrecedingSibling
	case "following":
		return axFollowing
	case "preceding":
		return axPreceding
	case "attribute":
		return axAttribute
	case "namespace":
		return axNamespace
	default:
		p.fail("unknown axis %q", name)
		return axChild
	}
}
