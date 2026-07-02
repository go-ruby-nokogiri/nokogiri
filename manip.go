// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

// unlink detaches n from its current parent and sibling links without clearing
// its own child pointers, so the caller can re-insert it elsewhere.
func (n *Node) unlink() {
	if n.parent != nil {
		if n.parent.firstChild == n {
			n.parent.firstChild = n.next
		}
		if n.parent.lastChild == n {
			n.parent.lastChild = n.prev
		}
	}
	if n.prev != nil {
		n.prev.next = n.next
	}
	if n.next != nil {
		n.next.prev = n.prev
	}
	n.parent = nil
	n.prev = nil
	n.next = nil
}

// setDocRecursive stamps doc onto n and every descendant.
func (n *Node) setDocRecursive(doc *Document) {
	n.doc = doc
	for c := n.firstChild; c != nil; c = c.next {
		c.setDocRecursive(doc)
	}
}

// AddChild appends child as the last child of n (Nokogiri::XML::Node#add_child /
// #<<). If child already has a parent it is detached first. Returns child.
func (n *Node) AddChild(child *Node) *Node {
	child.unlink()
	child.parent = n
	child.setDocRecursive(n.doc)
	if n.lastChild == nil {
		n.firstChild = child
		n.lastChild = child
		return child
	}
	child.prev = n.lastChild
	n.lastChild.next = child
	n.lastChild = child
	return child
}

// Prepend inserts child as the first child of n (Nokogiri#prepend_child).
func (n *Node) Prepend(child *Node) *Node {
	child.unlink()
	child.parent = n
	child.setDocRecursive(n.doc)
	if n.firstChild == nil {
		n.firstChild = child
		n.lastChild = child
		return child
	}
	child.next = n.firstChild
	n.firstChild.prev = child
	n.firstChild = child
	return child
}

// AddNextSibling inserts sib immediately after n (Nokogiri#add_next_sibling).
func (n *Node) AddNextSibling(sib *Node) *Node {
	sib.unlink()
	sib.parent = n.parent
	sib.setDocRecursive(n.doc)
	sib.prev = n
	sib.next = n.next
	if n.next != nil {
		n.next.prev = sib
	} else if n.parent != nil {
		n.parent.lastChild = sib
	}
	n.next = sib
	return sib
}

// AddPreviousSibling inserts sib immediately before n (Nokogiri#add_previous_sibling).
func (n *Node) AddPreviousSibling(sib *Node) *Node {
	sib.unlink()
	sib.parent = n.parent
	sib.setDocRecursive(n.doc)
	sib.next = n
	sib.prev = n.prev
	if n.prev != nil {
		n.prev.next = sib
	} else if n.parent != nil {
		n.parent.firstChild = sib
	}
	n.prev = sib
	return sib
}

// Remove detaches n from its tree (Nokogiri::XML::Node#remove / #unlink).
func (n *Node) Remove() { n.unlink() }

// Replace swaps n out for repl in the tree (Nokogiri::XML::Node#replace).
func (n *Node) Replace(repl *Node) *Node {
	repl.unlink()
	repl.parent = n.parent
	repl.setDocRecursive(n.doc)
	repl.prev = n.prev
	repl.next = n.next
	if n.prev != nil {
		n.prev.next = repl
	} else if n.parent != nil {
		n.parent.firstChild = repl
	}
	if n.next != nil {
		n.next.prev = repl
	} else if n.parent != nil {
		n.parent.lastChild = repl
	}
	// Detach n directly (do NOT call unlink here: n's sibling links have already
	// been repurposed to point at repl, so re-running the sibling rewiring would
	// clobber repl's placement).
	n.parent = nil
	n.prev = nil
	n.next = nil
	return repl
}

// SetContent replaces all children of n with a single text node holding s
// (Nokogiri::XML::Node#content=).
func (n *Node) SetContent(s string) {
	n.firstChild = nil
	n.lastChild = nil
	if n.Type == TextNode || n.Type == CommentNode || n.Type == CDATANode {
		n.content = s
		return
	}
	t := &Node{Type: TextNode, content: s, doc: n.doc}
	n.AddChild(t)
}

// NewElement creates a detached element node named name owned by the document,
// mirroring Nokogiri::XML::Document#create_element.
func (d *Document) NewElement(name string) *Node {
	prefix, local := splitQName(name)
	return &Node{Type: ElementNode, Name: local, Prefix: prefix, doc: d}
}

// NewText creates a detached text node (Nokogiri::XML::Document#create_text_node).
func (d *Document) NewText(s string) *Node {
	return &Node{Type: TextNode, content: s, doc: d}
}

// NewCDATA creates a detached CDATA node (Nokogiri::XML::Document#create_cdata).
func (d *Document) NewCDATA(s string) *Node {
	return &Node{Type: CDATANode, content: s, doc: d}
}

// NewComment creates a detached comment node (Nokogiri::XML::Document#create_comment).
func (d *Document) NewComment(s string) *Node {
	return &Node{Type: CommentNode, content: s, doc: d}
}
