// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

// This file exposes the extension seam that the XSLT 1.0 processor
// (github.com/go-ruby-xslt/xslt) builds on. XSLT drives the same XPath 1.0 engine
// Nokogiri exposes, but it additionally needs three things the plain
// Node.XPath/EvalXPath entry points do not surface:
//
//   - variable bindings ($name), for xsl:variable / xsl:param / xsl:with-param;
//   - an extension-function resolver, for the XSLT function library
//     (key(), format-number(), generate-id(), current(), document(),
//     system-property(), element-available(), function-available());
//   - control over the current() node, which in XSLT is the node the pattern is
//     being evaluated against rather than the innermost predicate context.
//
// The engine already carries all three internally (evalContext.vars/current and
// the callFunc switch); XPathContext is the public projection of that seam. Values
// crossing the boundary use the same object model EvalXPath returns: *NodeSet,
// string, float64 and bool. Node-lists are wrapped as *NodeSet on the way out and
// unwrapped on the way in, so callers never see the unexported nodeList.

// XPathContext supplies variable bindings and an extension-function resolver to an
// XPath evaluation. A nil *XPathContext behaves exactly like Node.EvalXPath.
type XPathContext struct {
	// Vars binds $name references. A value may be a *NodeSet, string, float64 or
	// bool; other numeric/int kinds are coerced to float64 by NewXPathValue.
	Vars map[string]any

	// ResolveFunc, when non-nil, is consulted for any function call the built-in
	// XPath library does not implement. It receives the function's already-evaluated
	// arguments (each a *NodeSet, string, float64 or bool) and returns the result
	// plus ok=true, or ok=false to fall through to the built-in "unknown function"
	// error. This is where XSLT plugs key(), format-number(), and friends.
	ResolveFunc func(name string, args []any) (result any, ok bool)

	// Current overrides the node returned by current(). When nil, current() yields
	// the context node the expression is evaluated against (the default behaviour).
	Current *Node
}

// NewNodeSet builds a *NodeSet from a slice of nodes. XSLT uses it to hand
// node-set variable values and extension-function results back to the engine.
func NewNodeSet(nodes []*Node) *NodeSet { return &NodeSet{nodes: nodes} }

// toXPValue converts a public boundary value into the engine's internal value.
func toXPValue(v any) xpValue {
	switch t := v.(type) {
	case nil:
		return &nodeList{}
	case *NodeSet:
		return &nodeList{nodes: t.nodes}
	case *Node:
		if t == nil {
			return &nodeList{}
		}
		return &nodeList{nodes: []*Node{t}}
	case []*Node:
		return &nodeList{nodes: t}
	case string:
		return t
	case bool:
		return t
	case float64:
		return t
	case int:
		return float64(t)
	case int64:
		return float64(t)
	default:
		return &nodeList{}
	}
}

// fromXPValue converts an engine value into a public boundary value.
func fromXPValue(v xpValue) any {
	if nl, ok := v.(*nodeList); ok {
		return &NodeSet{nodes: nl.nodes}
	}
	return v
}

// EvalXPathCtx evaluates expr against n with the variable bindings and
// extension-function resolver carried by ctx (which may be nil). Node-sets are
// returned as *NodeSet; scalar results as string, float64 or bool — the same
// object model as EvalXPath.
func (n *Node) EvalXPathCtx(expr string, nsMap map[string]string, ctx *XPathContext) (any, error) {
	var vars map[string]xpValue
	var hook func(string, []xpValue) (xpValue, bool)
	var current *Node
	if ctx != nil {
		if len(ctx.Vars) > 0 {
			vars = make(map[string]xpValue, len(ctx.Vars))
			for k, val := range ctx.Vars {
				vars[k] = toXPValue(val)
			}
		}
		if ctx.ResolveFunc != nil {
			rf := ctx.ResolveFunc
			hook = func(name string, args []xpValue) (xpValue, bool) {
				pub := make([]any, len(args))
				for i, a := range args {
					pub[i] = fromXPValue(a)
				}
				res, ok := rf(name, pub)
				if !ok {
					return nil, false
				}
				return toXPValue(res), true
			}
		}
		current = ctx.Current
	}
	v, err := evalXPathExt(expr, n, vars, nsMap, hook, current)
	if err != nil {
		return nil, err
	}
	return fromXPValue(v), nil
}
