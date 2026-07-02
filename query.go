// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

// XPath evaluates an XPath 1.0 expression against n and returns the matching
// nodes as a NodeSet (Nokogiri::XML::Node#xpath). Namespaces registered with
// nsMap (prefix->URI) are visible to prefixed name tests.
func (n *Node) XPath(expr string, nsMap map[string]string) (*NodeSet, error) {
	v, err := evalXPath(expr, n, nil, nsMap)
	if err != nil {
		return nil, err
	}
	nl, ok := v.(*nodeList)
	if !ok {
		return &NodeSet{}, nil
	}
	return &NodeSet{nodes: nl.nodes}, nil
}

// AtXPath returns the first node matching the XPath expression, or nil
// (Nokogiri#at_xpath).
func (n *Node) AtXPath(expr string, nsMap map[string]string) (*Node, error) {
	set, err := n.XPath(expr, nsMap)
	if err != nil {
		return nil, err
	}
	return set.First(), nil
}

// EvalXPath evaluates expr and returns the raw XPath object (a *NodeSet, string,
// float64, or bool), matching how Nokogiri returns scalar results for functions
// like count()/string(). Node-sets are wrapped as *NodeSet.
func (n *Node) EvalXPath(expr string, nsMap map[string]string) (any, error) {
	v, err := evalXPath(expr, n, nil, nsMap)
	if err != nil {
		return nil, err
	}
	if nl, ok := v.(*nodeList); ok {
		return &NodeSet{nodes: nl.nodes}, nil
	}
	return v, nil
}

// CSS evaluates one or more comma-separated CSS selectors against n and returns
// the matching nodes (Nokogiri::XML::Node#css). Selectors are translated to XPath
// and evaluated over the descendant axis rooted at n.
func (n *Node) CSS(selector string, nsMap map[string]string) (*NodeSet, error) {
	xp, err := cssToXPath(selector, ".//")
	if err != nil {
		return nil, err
	}
	return n.XPath(xp, nsMap)
}

// AtCSS returns the first node matching the CSS selector, or nil (Nokogiri#at_css).
func (n *Node) AtCSS(selector string, nsMap map[string]string) (*Node, error) {
	set, err := n.CSS(selector, nsMap)
	if err != nil {
		return nil, err
	}
	return set.First(), nil
}

// --- Document convenience wrappers -----------------------------------------

// CSS runs a CSS query against the document root (Nokogiri::XML::Document#css).
func (d *Document) CSS(selector string) (*NodeSet, error) {
	return d.Node.CSS(selector, nil)
}

// AtCSS runs a CSS query and returns the first match.
func (d *Document) AtCSS(selector string) (*Node, error) {
	return d.Node.AtCSS(selector, nil)
}

// XPath runs an XPath query against the document (Nokogiri::XML::Document#xpath).
func (d *Document) XPath(expr string) (*NodeSet, error) {
	return d.Node.XPath(expr, nil)
}

// AtXPath runs an XPath query and returns the first match.
func (d *Document) AtXPath(expr string) (*Node, error) {
	return d.Node.AtXPath(expr, nil)
}

// --- NodeSet query wrappers (union across members) -------------------------

// CSS runs a CSS query against every node in the set and unions the results.
func (s *NodeSet) CSS(selector string) (*NodeSet, error) {
	var all []*Node
	for _, n := range s.nodes {
		sub, err := n.CSS(selector, nil)
		if err != nil {
			return nil, err
		}
		all = append(all, sub.nodes...)
	}
	return &NodeSet{nodes: dedupInDocOrder(all)}, nil
}

// XPath runs an XPath query against every node in the set and unions the results.
func (s *NodeSet) XPath(expr string) (*NodeSet, error) {
	var all []*Node
	for _, n := range s.nodes {
		sub, err := n.XPath(expr, nil)
		if err != nil {
			return nil, err
		}
		all = append(all, sub.nodes...)
	}
	return &NodeSet{nodes: dedupInDocOrder(all)}, nil
}
