// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

// Get returns the value of the named attribute and whether it was present. The
// lookup matches on the qualified name as written (prefix:local) first, then on
// the bare local name, mirroring Nokogiri::XML::Node#[].
func (n *Node) Get(name string) (string, bool) {
	for _, a := range n.Attrs {
		if a.qualified() == name || a.Name == name {
			return a.Value, true
		}
	}
	return "", false
}

// Attribute returns the value of the named attribute, or "" if absent. This is
// the ergonomic form of Nokogiri#[]/#attr.
func (n *Node) Attribute(name string) string {
	v, _ := n.Get(name)
	return v
}

// HasAttribute reports whether the named attribute is present.
func (n *Node) HasAttribute(name string) bool {
	_, ok := n.Get(name)
	return ok
}

// SetAttribute sets (or creates) the named attribute, matching
// Nokogiri::XML::Node#set_attribute / #[]=.
func (n *Node) SetAttribute(name, value string) {
	for _, a := range n.Attrs {
		if a.qualified() == name || a.Name == name {
			a.Value = value
			return
		}
	}
	prefix, local := splitQName(name)
	n.Attrs = append(n.Attrs, &Attr{Name: local, Prefix: prefix, Value: value})
}

// RemoveAttribute deletes the named attribute if present (Nokogiri#remove_attribute).
func (n *Node) RemoveAttribute(name string) {
	for i, a := range n.Attrs {
		if a.qualified() == name || a.Name == name {
			n.Attrs = append(n.Attrs[:i], n.Attrs[i+1:]...)
			return
		}
	}
}

// Attributes returns the node's attributes keyed by their qualified name,
// matching Nokogiri::XML::Node#attributes (best effort; last write wins on a
// duplicate qualified name).
func (n *Node) Attributes() map[string]*Attr {
	m := make(map[string]*Attr, len(n.Attrs))
	for _, a := range n.Attrs {
		m[a.qualified()] = a
	}
	return m
}

// NamespaceDeclarations returns the xmlns declarations introduced on this node.
func (n *Node) NamespaceDeclarations() []*Namespace { return n.nsDecls }

// qualified returns the attribute name as written (prefix:local or local).
func (a *Attr) qualified() string {
	if a.Prefix != "" {
		return a.Prefix + ":" + a.Name
	}
	return a.Name
}

// splitQName splits "prefix:local" into its parts; a name with no colon has an
// empty prefix.
func splitQName(name string) (prefix, local string) {
	for i := 0; i < len(name); i++ {
		if name[i] == ':' {
			return name[:i], name[i+1:]
		}
	}
	return "", name
}
