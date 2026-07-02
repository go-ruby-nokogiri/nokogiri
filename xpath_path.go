// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

// evalPath evaluates a location path (optionally rooted at the document or based
// on a filter primary expression).
func evalPath(pe *pathExpr, ctx *evalContext) xpValue {
	var start []*Node
	switch {
	case pe.filter != nil:
		v := eval(pe.filter, ctx)
		start = toNodeList(v).nodes
	case pe.rooted:
		start = []*Node{&ctx.node.doc.Node}
	default:
		start = []*Node{ctx.node}
	}

	cur := start
	for _, s := range pe.steps {
		cur = ctx.evalStep(s, cur)
	}
	return ctx.sortUnique(cur)
}

// evalStep applies one step to every node in the input set and returns the union
// of results (deduped, unsorted; final sort happens once in evalPath).
func (ctx *evalContext) evalStep(s step, input []*Node) []*Node {
	var out []*Node
	seen := map[*Node]bool{}
	for _, n := range input {
		candidates := ctx.axisNodes(s, n)
		matched := candidates[:0:0]
		for _, c := range candidates {
			if ctx.nodeTestMatch(s, c) {
				matched = append(matched, c)
			}
		}
		matched = ctx.applyPredicates(s, matched)
		for _, m := range matched {
			if !seen[m] {
				seen[m] = true
				out = append(out, m)
			}
		}
	}
	return out
}

// axisNodes returns the nodes reachable along the step's axis from n, in the
// axis's natural order.
func (ctx *evalContext) axisNodes(s step, n *Node) []*Node {
	switch s.axis {
	case axChild:
		return childrenOf(n)
	case axSelf:
		return []*Node{n}
	case axParent:
		if n.parent != nil {
			return []*Node{n.parent}
		}
		return nil
	case axDescendant:
		return descendants(n, false)
	case axDescendantOrSelf:
		return descendants(n, true)
	case axAncestor:
		return ancestors(n, false)
	case axAncestorOrSelf:
		return ancestors(n, true)
	case axFollowingSibling:
		var out []*Node
		for s := n.next; s != nil; s = s.next {
			out = append(out, s)
		}
		return out
	case axPrecedingSibling:
		var out []*Node
		for s := n.prev; s != nil; s = s.prev {
			out = append(out, s)
		}
		return out
	case axFollowing:
		return followingNodes(n)
	case axPreceding:
		return precedingNodes(n)
	case axAttribute:
		return attrNodes(n)
	case axNamespace:
		return nil
	}
	return nil
}

func childrenOf(n *Node) []*Node {
	var out []*Node
	for c := n.firstChild; c != nil; c = c.next {
		out = append(out, c)
	}
	return out
}

func descendants(n *Node, includeSelf bool) []*Node {
	var out []*Node
	if includeSelf {
		out = append(out, n)
	}
	var walk func(*Node)
	walk = func(x *Node) {
		for c := x.firstChild; c != nil; c = c.next {
			out = append(out, c)
			walk(c)
		}
	}
	walk(n)
	return out
}

func ancestors(n *Node, includeSelf bool) []*Node {
	var out []*Node
	if includeSelf {
		out = append(out, n)
	}
	for p := n.parent; p != nil; p = p.parent {
		out = append(out, p)
	}
	return out
}

func attrNodes(n *Node) []*Node {
	if n.Type != ElementNode {
		return nil
	}
	out := make([]*Node, 0, len(n.Attrs))
	for _, a := range n.Attrs {
		out = append(out, &Node{
			Type: AttributeNode, Name: a.Name, Prefix: a.Prefix,
			NsURI: a.Namespace, content: a.Value, parent: n, doc: n.doc,
		})
	}
	return out
}

// followingNodes returns all nodes after n in document order that are not
// ancestors or descendants of n.
func followingNodes(n *Node) []*Node {
	var out []*Node
	for cur := n; cur != nil; {
		// move to next sibling, walking up when needed
		for cur != nil && cur.next == nil {
			cur = cur.parent
		}
		if cur == nil {
			break
		}
		cur = cur.next
		out = append(out, cur)
		out = append(out, descendants(cur, false)...)
	}
	return out
}

// precedingNodes returns all nodes before n in document order excluding ancestors.
func precedingNodes(n *Node) []*Node {
	anc := map[*Node]bool{}
	for p := n.parent; p != nil; p = p.parent {
		anc[p] = true
	}
	root := n
	if n.doc != nil {
		root = &n.doc.Node
	}
	var out []*Node
	stop := false
	var walk func(*Node)
	walk = func(x *Node) {
		if stop {
			return
		}
		if x == n {
			stop = true
			return
		}
		if !anc[x] && x != root {
			out = append(out, x)
		} else if x == root {
			// don't include the document node
		}
		for c := x.firstChild; c != nil && !stop; c = c.next {
			walk(c)
		}
	}
	walk(root)
	return out
}

// nodeTestMatch reports whether n satisfies the step's node test.
func (ctx *evalContext) nodeTestMatch(s step, n *Node) bool {
	switch s.test {
	case ntNode:
		return true
	case ntText:
		return n.Type == TextNode || n.Type == CDATANode
	case ntComment:
		return n.Type == CommentNode
	case ntPI:
		if n.Type != ProcessingInstructionNode {
			return false
		}
		return s.name == "" || n.Name == s.name
	case ntAny:
		if s.axis == axAttribute {
			return n.Type == AttributeNode
		}
		return n.Type == ElementNode
	case ntName:
		return ctx.nameTest(s, n)
	}
	return false
}

func (ctx *evalContext) nameTest(s step, n *Node) bool {
	if s.axis == axAttribute {
		if n.Type != AttributeNode {
			return false
		}
	} else if n.Type != ElementNode {
		return false
	}
	if n.Name != s.name {
		return false
	}
	if s.prefix == "" {
		return true
	}
	// Resolve the prefix against the registered namespace context.
	if ctx.ns != nil {
		if uri, ok := ctx.ns[s.prefix]; ok {
			return n.NsURI == uri
		}
	}
	// Fall back to matching the literal prefix as written.
	return n.Prefix == s.prefix
}

// applyPredicates filters nodes through the step's predicates, updating context
// position/size for each predicate pass.
func (ctx *evalContext) applyPredicates(s step, nodes []*Node) []*Node {
	for _, pred := range s.predicates {
		var kept []*Node
		size := len(nodes)
		for i, n := range nodes {
			// current() resolves to the node matched by the *outermost* location
			// step, so it changes only at the top-level predicate; a predicate
			// nested inside another (e.g. the [name()=name(current())] the CSS
			// :nth-of-type compiler emits) keeps the outer candidate. We detect the
			// outermost predicate by current still pointing at the eval root.
			cur := ctx.current
			if ctx.current == ctx.root {
				cur = n
			}
			sub := &evalContext{node: n, current: cur, root: ctx.root, pos: i + 1, size: size, vars: ctx.vars, ns: ctx.ns, docid: ctx.docid}
			v := eval(pred, sub)
			if num, ok := v.(float64); ok {
				if int(num) == i+1 {
					kept = append(kept, n)
				}
				continue
			}
			if toBool(v) {
				kept = append(kept, n)
			}
		}
		nodes = kept
	}
	return nodes
}
