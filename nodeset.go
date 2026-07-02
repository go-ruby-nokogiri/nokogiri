// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

import "strings"

// NodeSet is an ordered collection of nodes, the analogue of
// Nokogiri::XML::NodeSet returned by #css / #xpath / #children.
type NodeSet struct {
	nodes []*Node
}

// Length returns the number of nodes in the set (Nokogiri#length / #size).
func (s *NodeSet) Length() int { return len(s.nodes) }

// Empty reports whether the set has no nodes.
func (s *NodeSet) Empty() bool { return len(s.nodes) == 0 }

// At returns the node at index i, or nil if out of range. Negative indices count
// from the end, matching Ruby's NodeSet#[].
func (s *NodeSet) At(i int) *Node {
	if i < 0 {
		i += len(s.nodes)
	}
	if i < 0 || i >= len(s.nodes) {
		return nil
	}
	return s.nodes[i]
}

// First returns the first node, or nil.
func (s *NodeSet) First() *Node {
	if len(s.nodes) == 0 {
		return nil
	}
	return s.nodes[0]
}

// Last returns the last node, or nil.
func (s *NodeSet) Last() *Node {
	if len(s.nodes) == 0 {
		return nil
	}
	return s.nodes[len(s.nodes)-1]
}

// Nodes returns the underlying slice (a copy is not made; do not mutate).
func (s *NodeSet) Nodes() []*Node { return s.nodes }

// Each calls fn for every node in order.
func (s *NodeSet) Each(fn func(*Node)) {
	for _, n := range s.nodes {
		fn(n)
	}
}

// Text returns the concatenated text of every node in the set, matching
// Nokogiri::XML::NodeSet#text.
func (s *NodeSet) Text() string {
	var b strings.Builder
	for _, n := range s.nodes {
		b.WriteString(n.Text())
	}
	return b.String()
}

// dedupInDocOrder removes duplicate node pointers while keeping the first
// occurrence's order. XPath location paths can otherwise revisit a node.
func dedupInDocOrder(nodes []*Node) []*Node {
	seen := make(map[*Node]bool, len(nodes))
	out := nodes[:0:0]
	for _, n := range nodes {
		if !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	return out
}
