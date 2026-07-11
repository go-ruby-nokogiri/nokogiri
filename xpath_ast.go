// Copyright (c) the go-ruby-nokogiri/nokogiri authors
//
// SPDX-License-Identifier: BSD-3-Clause

package nokogiri

// The XPath 1.0 abstract syntax tree. An expr evaluates against a context to an
// object (node-set, string, number, or boolean).

// axis identifies an XPath axis.
type axis int

const (
	axChild axis = iota
	axDescendant
	axDescendantOrSelf
	axSelf
	axParent
	axAncestor
	axAncestorOrSelf
	axFollowingSibling
	axPrecedingSibling
	axFollowing
	axPreceding
	axAttribute
	axNamespace
)

// nodeTestKind identifies the kind of node test in a step.
type nodeTestKind int

const (
	ntName    nodeTestKind = iota // name test (possibly with prefix / *)
	ntNode                        // node()
	ntText                        // text()
	ntComment                     // comment()
	ntPI                          // processing-instruction(...)
	ntAny                         // *
)

// expr is any XPath expression node.
type expr interface{}

// pathExpr is a location path: an ordered list of steps, optionally rooted at the
// document ("/"). filter, if non-nil, is a primary expression whose node-set the
// path is applied to (e.g. (//a)[1]/b or key(...)/x).
type pathExpr struct {
	rooted bool
	filter expr // nil for a plain location path
	steps  []step
}

// step is one location step: axis, node test, and predicates.
type step struct {
	axis       axis
	test       nodeTestKind
	prefix     string // for ntName with a prefix
	name       string // for ntName / ntPI target
	predicates []expr
}

// binaryExpr is an infix operation (and/or/=/!=/relational/additive/mult/|).
type binaryExpr struct {
	op   string
	l, r expr
}

// unaryExpr is arithmetic negation.
type unaryExpr struct{ x expr }

// numberLit is a numeric literal.
type numberLit struct{ v float64 }

// stringLit is a string literal.
type stringLit struct{ v string }

// funcCall is a core-library (or registered) function call.
type funcCall struct {
	name string
	args []expr
}

// varRef is a $variable reference.
type varRef struct{ name string }
