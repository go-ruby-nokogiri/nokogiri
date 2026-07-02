// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

import (
	"math"
	"strings"
)

// callFunc dispatches an XPath core-library function call.
func callFunc(fc *funcCall, ctx *evalContext) xpValue {
	args := fc.args
	switch fc.name {
	// --- node-set functions ---
	case "last":
		return float64(ctx.size)
	case "position":
		return float64(ctx.pos)
	case "current":
		// current() is always bound (to the eval root, then to the filtered node
		// inside a predicate), so it yields a single-node set.
		return &nodeList{nodes: []*Node{ctx.current}}
	case "count":
		return float64(len(toNodeList(eval(arg(args, 0, fc), ctx)).nodes))
	case "id":
		return ctx.fnID(eval(arg(args, 0, fc), ctx))
	case "local-name":
		n := ctx.nodeArgOrContext(args, ctx)
		if n == nil {
			return ""
		}
		return n.Name
	case "name":
		n := ctx.nodeArgOrContext(args, ctx)
		if n == nil {
			return ""
		}
		return n.NodeName()
	case "namespace-uri":
		n := ctx.nodeArgOrContext(args, ctx)
		if n == nil {
			return ""
		}
		return n.NsURI

	// --- string functions ---
	case "string":
		if len(args) == 0 {
			return stringValue(ctx.node)
		}
		return toString(eval(args[0], ctx))
	case "concat":
		var b strings.Builder
		for _, a := range args {
			b.WriteString(toString(eval(a, ctx)))
		}
		return b.String()
	case "starts-with":
		return strings.HasPrefix(toString(eval(arg(args, 0, fc), ctx)), toString(eval(arg(args, 1, fc), ctx)))
	case "contains":
		return strings.Contains(toString(eval(arg(args, 0, fc), ctx)), toString(eval(arg(args, 1, fc), ctx)))
	case "substring-before":
		s := toString(eval(arg(args, 0, fc), ctx))
		sub := toString(eval(arg(args, 1, fc), ctx))
		if i := strings.Index(s, sub); i >= 0 {
			return s[:i]
		}
		return ""
	case "substring-after":
		s := toString(eval(arg(args, 0, fc), ctx))
		sub := toString(eval(arg(args, 1, fc), ctx))
		if i := strings.Index(s, sub); i >= 0 {
			return s[i+len(sub):]
		}
		return ""
	case "substring":
		return fnSubstring(args, ctx, fc)
	case "string-length":
		var s string
		if len(args) == 0 {
			s = stringValue(ctx.node)
		} else {
			s = toString(eval(args[0], ctx))
		}
		return float64(len([]rune(s)))
	case "normalize-space":
		var s string
		if len(args) == 0 {
			s = stringValue(ctx.node)
		} else {
			s = toString(eval(args[0], ctx))
		}
		return strings.Join(strings.Fields(s), " ")
	case "translate":
		return fnTranslate(args, ctx, fc)

	// --- boolean functions ---
	case "boolean":
		return toBool(eval(arg(args, 0, fc), ctx))
	case "not":
		return !toBool(eval(arg(args, 0, fc), ctx))
	case "true":
		return true
	case "false":
		return false
	case "lang":
		return ctx.fnLang(toString(eval(arg(args, 0, fc), ctx)))

	// --- number functions ---
	case "number":
		if len(args) == 0 {
			return toNumber(stringValue(ctx.node))
		}
		return toNumber(eval(args[0], ctx))
	case "sum":
		nl := toNodeList(eval(arg(args, 0, fc), ctx))
		var total float64
		for _, n := range nl.nodes {
			total += toNumber(stringValue(n))
		}
		return total
	case "floor":
		return math.Floor(toNumber(eval(arg(args, 0, fc), ctx)))
	case "ceiling":
		return math.Ceil(toNumber(eval(arg(args, 0, fc), ctx)))
	case "round":
		f := toNumber(eval(arg(args, 0, fc), ctx))
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return f
		}
		return math.Floor(f + 0.5)
	}
	panic(evalError("xpath: unknown function " + fc.name + "()"))
}

func arg(args []expr, i int, fc *funcCall) expr {
	if i >= len(args) {
		panic(evalError("xpath: " + fc.name + "() missing argument"))
	}
	return args[i]
}

// nodeArgOrContext returns the first node of the node-set argument, or the context
// node when no argument is given.
func (ctx *evalContext) nodeArgOrContext(args []expr, c *evalContext) *Node {
	if len(args) == 0 {
		return ctx.node
	}
	nl := toNodeList(eval(args[0], c))
	if len(nl.nodes) == 0 {
		return nil
	}
	return nl.nodes[0]
}

func fnSubstring(args []expr, ctx *evalContext, fc *funcCall) xpValue {
	s := []rune(toString(eval(arg(args, 0, fc), ctx)))
	start := toNumber(eval(arg(args, 1, fc), ctx))
	var length float64 = math.Inf(1)
	if len(args) >= 3 {
		length = toNumber(eval(args[2], ctx))
	}
	if math.IsNaN(start) {
		return ""
	}
	// XPath is 1-based with rounding.
	from := math.Round(start)
	to := from + length
	if !math.IsInf(length, 1) {
		to = math.Round(start) + math.Round(length)
	}
	lo := int(math.Max(from, 1))
	var hi int
	if math.IsInf(to, 1) {
		hi = len(s) + 1
	} else {
		hi = int(to)
	}
	if hi > len(s)+1 {
		hi = len(s) + 1
	}
	// lo is already clamped to at least 1 by the math.Max above.
	if hi <= lo {
		return ""
	}
	return string(s[lo-1 : hi-1])
}

func fnTranslate(args []expr, ctx *evalContext, fc *funcCall) xpValue {
	s := toString(eval(arg(args, 0, fc), ctx))
	from := []rune(toString(eval(arg(args, 1, fc), ctx)))
	to := []rune(toString(eval(arg(args, 2, fc), ctx)))
	m := make(map[rune]rune, len(from))
	del := make(map[rune]bool)
	for i, r := range from {
		if _, seen := m[r]; seen || del[r] {
			continue
		}
		if i < len(to) {
			m[r] = to[i]
		} else {
			del[r] = true
		}
	}
	var b strings.Builder
	for _, r := range s {
		if del[r] {
			continue
		}
		if nr, ok := m[r]; ok {
			b.WriteRune(nr)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// fnID resolves space-separated id tokens to elements whose id attribute matches.
func (ctx *evalContext) fnID(v xpValue) xpValue {
	var ids []string
	if nl, ok := v.(*nodeList); ok {
		for _, n := range nl.nodes {
			ids = append(ids, strings.Fields(stringValue(n))...)
		}
	} else {
		ids = strings.Fields(toString(v))
	}
	want := map[string]bool{}
	for _, id := range ids {
		want[id] = true
	}
	var out []*Node
	root := ctx.node
	if root.doc != nil {
		root = &root.doc.Node
	}
	for _, n := range descendants(root, true) {
		if n.Type == ElementNode {
			if idv, ok := n.Get("id"); ok && want[idv] {
				out = append(out, n)
			}
		}
	}
	return &nodeList{nodes: out}
}

// fnLang implements the lang() function against xml:lang in scope.
func (ctx *evalContext) fnLang(want string) bool {
	want = strings.ToLower(want)
	for n := ctx.node; n != nil; n = n.parent {
		if n.Type != ElementNode {
			continue
		}
		if v, ok := n.Get("xml:lang"); ok {
			v = strings.ToLower(v)
			return v == want || strings.HasPrefix(v, want+"-")
		}
	}
	return false
}
