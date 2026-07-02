// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

import (
	"math"
	"sort"
	"strconv"
	"strings"
)

// xpValue is an XPath object: a *nodeList, string, float64, or bool.
type xpValue interface{}

// nodeList is an ordered set of nodes with a document-order flag used during
// evaluation. It is converted to a NodeSet for the public API.
type nodeList struct{ nodes []*Node }

// evalContext carries the current node, position, size, variable bindings, and
// namespace map through evaluation.
type evalContext struct {
	node    *Node
	current *Node // the current() node: the context of the outermost expression
	root    *Node // the node the whole expression is evaluated against
	pos     int   // 1-based
	size    int
	vars    map[string]xpValue
	ns      map[string]string // prefix -> URI (registered via NamespaceContext)
	docid   map[*Node]int     // document-order index cache

	// funcHook, when set, resolves function calls the built-in library does not
	// implement (the XSLT extension seam; see xpath_ext.go). curOverride, when
	// non-nil, is the node current() reports instead of ctx.current.
	funcHook    func(name string, args []xpValue) (xpValue, bool)
	curOverride *Node
}

// evalError is a runtime XPath error.
type evalError string

func (e evalError) Error() string { return string(e) }

// recoverEvalError classifies a recovered panic value: an evalError becomes the
// returned error, and any other value is re-raised (a genuine bug, not an XPath
// diagnostic). It returns nil when there was no panic.
func recoverEvalError(r any) error {
	if r == nil {
		return nil
	}
	if ee, ok := r.(evalError); ok {
		return ee
	}
	panic(r)
}

// evalXPath compiles and evaluates expr against ctxNode, returning the raw value.
func evalXPath(expr string, ctxNode *Node, vars map[string]xpValue, ns map[string]string) (v xpValue, err error) {
	ast, err := parseXPath(expr)
	if err != nil {
		return nil, err
	}
	defer func() {
		if e := recoverEvalError(recover()); e != nil {
			v, err = nil, e
		}
	}()
	ctx := &evalContext{node: ctxNode, current: ctxNode, root: ctxNode, pos: 1, size: 1, vars: vars, ns: ns, docid: map[*Node]int{}}
	ctx.indexDoc(ctxNode)
	return eval(ast, ctx), nil
}

// evalXPathExt is evalXPath with the two extra seams the XSLT processor drives
// (see xpath_ext.go): an extension-function resolver consulted before the
// "unknown function" error, and an override for the current() node.
func evalXPathExt(expr string, ctxNode *Node, vars map[string]xpValue, ns map[string]string, hook func(string, []xpValue) (xpValue, bool), current *Node, pos, size int) (v xpValue, err error) {
	ast, err := parseXPath(expr)
	if err != nil {
		return nil, err
	}
	defer func() {
		if e := recoverEvalError(recover()); e != nil {
			v, err = nil, e
		}
	}()
	cur := ctxNode
	if current != nil {
		cur = current
	}
	ctx := &evalContext{node: ctxNode, current: cur, root: ctxNode, pos: pos, size: size, vars: vars, ns: ns, docid: map[*Node]int{}, funcHook: hook, curOverride: current}
	ctx.indexDoc(ctxNode)
	return eval(ast, ctx), nil
}

// indexDoc numbers every node of the owning document in document order so we can
// sort/dedup node-sets cheaply.
func (c *evalContext) indexDoc(from *Node) {
	root := from
	if from.doc != nil {
		root = &from.doc.Node
	}
	i := 0
	var walk func(*Node)
	walk = func(n *Node) {
		c.docid[n] = i
		i++
		for _, a := range n.Attrs {
			_ = a
		}
		for ch := n.firstChild; ch != nil; ch = ch.next {
			walk(ch)
		}
	}
	walk(root)
}

func eval(e expr, ctx *evalContext) xpValue {
	switch ex := e.(type) {
	case *numberLit:
		return ex.v
	case *stringLit:
		return ex.v
	case *varRef:
		if ctx.vars != nil {
			if v, ok := ctx.vars[ex.name]; ok {
				return v
			}
		}
		panic(evalError("xpath: undefined variable $" + ex.name))
	case *unaryExpr:
		return -toNumber(eval(ex.x, ctx))
	case *funcCall:
		return callFunc(ex, ctx)
	case *binaryExpr:
		return evalBinary(ex, ctx)
	default:
		// The parser only ever produces the expr kinds above; *pathExpr is the
		// remaining one.
		return evalPath(e.(*pathExpr), ctx)
	}
}

func evalBinary(ex *binaryExpr, ctx *evalContext) xpValue {
	switch ex.op {
	case "or":
		return toBool(eval(ex.l, ctx)) || toBool(eval(ex.r, ctx))
	case "and":
		return toBool(eval(ex.l, ctx)) && toBool(eval(ex.r, ctx))
	case "|":
		l := toNodeList(eval(ex.l, ctx))
		r := toNodeList(eval(ex.r, ctx))
		merged := append(append([]*Node{}, l.nodes...), r.nodes...)
		return ctx.sortUnique(merged)
	case "=", "!=":
		return compareEquality(ex.op, eval(ex.l, ctx), eval(ex.r, ctx))
	case "<", ">", "<=", ">=":
		return compareRelational(ex.op, eval(ex.l, ctx), eval(ex.r, ctx))
	case "+":
		return toNumber(eval(ex.l, ctx)) + toNumber(eval(ex.r, ctx))
	case "-":
		return toNumber(eval(ex.l, ctx)) - toNumber(eval(ex.r, ctx))
	case "*":
		return toNumber(eval(ex.l, ctx)) * toNumber(eval(ex.r, ctx))
	case "div":
		return toNumber(eval(ex.l, ctx)) / toNumber(eval(ex.r, ctx))
	default:
		// The only remaining operator the parser emits is "mod".
		return math.Mod(toNumber(eval(ex.l, ctx)), toNumber(eval(ex.r, ctx)))
	}
}

// sortUnique returns a nodeList sorted in document order with duplicates removed.
func (c *evalContext) sortUnique(nodes []*Node) *nodeList {
	nodes = dedupInDocOrder(nodes)
	sort.SliceStable(nodes, func(i, j int) bool {
		return c.docOrder(nodes[i]) < c.docOrder(nodes[j])
	})
	return &nodeList{nodes: nodes}
}

// docOrder returns a node's document-order index, indexing lazily for attribute
// nodes and freshly built nodes not present in the initial scan.
func (c *evalContext) docOrder(n *Node) int {
	if idx, ok := c.docid[n]; ok {
		return idx
	}
	// Attribute nodes and dynamically created nodes: order by parent then append.
	if n.parent != nil {
		return c.docOrder(n.parent)*4096 + 1
	}
	return math.MaxInt / 2
}

// --- conversions -----------------------------------------------------------

func toNodeList(v xpValue) *nodeList {
	if nl, ok := v.(*nodeList); ok {
		return nl
	}
	panic(evalError("xpath: expected a node-set"))
}

func toBool(v xpValue) bool {
	switch t := v.(type) {
	case bool:
		return t
	case float64:
		return t != 0 && !math.IsNaN(t)
	case string:
		return t != ""
	case *nodeList:
		return len(t.nodes) > 0
	}
	return false
}

func toNumber(v xpValue) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case bool:
		if t {
			return 1
		}
		return 0
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(t), 64)
		if err != nil {
			return math.NaN()
		}
		return f
	case *nodeList:
		return toNumber(toString(v))
	}
	return math.NaN()
}

func toString(v xpValue) string {
	switch t := v.(type) {
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64:
		return formatNumber(t)
	case *nodeList:
		if len(t.nodes) == 0 {
			return ""
		}
		return stringValue(t.nodes[0])
	}
	return ""
}

// stringValue returns the XPath string-value of a node.
func stringValue(n *Node) string {
	switch n.Type {
	case AttributeNode:
		return n.content
	case TextNode, CDATANode, CommentNode:
		return n.content
	case ProcessingInstructionNode:
		return n.content
	default:
		return n.Text()
	}
}

// formatNumber renders a float per XPath number->string rules.
func formatNumber(f float64) string {
	if math.IsNaN(f) {
		return "NaN"
	}
	if math.IsInf(f, 1) {
		return "Infinity"
	}
	if math.IsInf(f, -1) {
		return "-Infinity"
	}
	if f == math.Trunc(f) && math.Abs(f) < 1e21 {
		return strconv.FormatFloat(f, 'f', -1, 64)
	}
	return strconv.FormatFloat(f, 'g', -1, 64)
}

// --- comparisons -----------------------------------------------------------

func compareEquality(op string, l, r xpValue) bool {
	ln, lIsNode := l.(*nodeList)
	rn, rIsNode := r.(*nodeList)
	switch {
	case lIsNode && rIsNode:
		for _, a := range ln.nodes {
			for _, b := range rn.nodes {
				if (stringValue(a) == stringValue(b)) == (op == "=") {
					return true
				}
			}
		}
		return false
	case lIsNode || rIsNode:
		var nl *nodeList
		var other xpValue
		if lIsNode {
			nl, other = ln, r
		} else {
			nl, other = rn, l
		}
		return nodeSetCompareScalar(op, nl, other)
	}
	// scalar vs scalar
	switch {
	case isBool(l) || isBool(r):
		return eqResult(op, toBool(l) == toBool(r))
	case isNum(l) || isNum(r):
		return eqResult(op, toNumber(l) == toNumber(r))
	default:
		return eqResult(op, toString(l) == toString(r))
	}
}

func nodeSetCompareScalar(op string, nl *nodeList, other xpValue) bool {
	switch other.(type) {
	case bool:
		want := toBool(other)
		return eqResult(op, toBool(nl) == want)
	case float64:
		for _, n := range nl.nodes {
			if eqResult(op, toNumber(stringValue(n)) == toNumber(other)) {
				return true
			}
		}
		return false
	default:
		s := toString(other)
		for _, n := range nl.nodes {
			if eqResult(op, stringValue(n) == s) {
				return true
			}
		}
		return false
	}
}

func eqResult(op string, equal bool) bool {
	if op == "=" {
		return equal
	}
	return !equal
}

func compareRelational(op string, l, r xpValue) bool {
	ln, lIsNode := l.(*nodeList)
	rn, rIsNode := r.(*nodeList)
	if lIsNode || rIsNode {
		lvals := numbersOf(l, ln, lIsNode)
		rvals := numbersOf(r, rn, rIsNode)
		for _, a := range lvals {
			for _, b := range rvals {
				if relCmp(op, a, b) {
					return true
				}
			}
		}
		return false
	}
	return relCmp(op, toNumber(l), toNumber(r))
}

func numbersOf(v xpValue, nl *nodeList, isNode bool) []float64 {
	if isNode {
		out := make([]float64, 0, len(nl.nodes))
		for _, n := range nl.nodes {
			out = append(out, toNumber(stringValue(n)))
		}
		return out
	}
	return []float64{toNumber(v)}
}

func relCmp(op string, a, b float64) bool {
	switch op {
	case "<":
		return a < b
	case ">":
		return a > b
	case "<=":
		return a <= b
	default:
		// The only remaining relational operator is ">=".
		return a >= b
	}
}

func isBool(v xpValue) bool { _, ok := v.(bool); return ok }
func isNum(v xpValue) bool  { _, ok := v.(float64); return ok }
