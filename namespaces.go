// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

// Namespaces returns every xmlns declaration in scope at n — the node's own
// declarations plus those inherited from its ancestors, nearest first — keyed the
// way Nokogiri::XML::Node#namespaces keys them: "xmlns" for the default namespace
// and "xmlns:<prefix>" for a prefixed one. When a prefix is redeclared on a nearer
// ancestor, the nearer declaration wins (and appears in its position). The implicit
// "xml" namespace is not included, matching Nokogiri.
func (n *Node) Namespaces() map[string]string {
	out := make(map[string]string)
	seen := make(map[string]bool)
	for cur := n; cur != nil; cur = cur.parent {
		for _, d := range cur.nsDecls {
			key := "xmlns"
			if d.Prefix != "" {
				key = "xmlns:" + d.Prefix
			}
			if seen[key] {
				continue
			}
			seen[key] = true
			out[key] = d.URI
		}
	}
	return out
}

// Namespace returns the namespace this node itself belongs to (its resolved
// prefix and URI), or nil when the node is not in any namespace — mirroring
// Nokogiri::XML::Node#namespace.
func (n *Node) Namespace() *Namespace {
	if n.NsURI == "" {
		return nil
	}
	return &Namespace{Prefix: n.Prefix, URI: n.NsURI}
}

// AddNamespace declares an xmlns mapping on n (an empty prefix sets the default
// namespace) and returns it, mirroring
// Nokogiri::XML::Node#add_namespace_definition. If the same prefix is already
// declared on n its URI is updated in place. The declaration takes effect for the
// node's own resolution and for its descendants.
func (n *Node) AddNamespace(prefix, uri string) *Namespace {
	for _, d := range n.nsDecls {
		if d.Prefix == prefix {
			d.URI = uri
			n.refreshNamespaces()
			return d
		}
	}
	d := &Namespace{Prefix: prefix, URI: uri}
	n.nsDecls = append(n.nsDecls, d)
	n.refreshNamespaces()
	return d
}

// refreshNamespaces re-resolves NsURI on the subtree rooted at n, using the
// declarations in scope from n's ancestors down. It is called after a namespace
// declaration changes so queries and #namespace see the new binding.
func (n *Node) refreshNamespaces() {
	scope := nsScope{}
	// Seed from ancestors (root-most first so nearer ones override).
	var chain []*Node
	for cur := n.parent; cur != nil; cur = cur.parent {
		chain = append(chain, cur)
	}
	for i := len(chain) - 1; i >= 0; i-- {
		for _, d := range chain[i].nsDecls {
			scope[d.Prefix] = d.URI
		}
	}
	resolveNamespaces(n, scope)
}
