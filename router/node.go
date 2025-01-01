package main

import "net/http"

type nodeType int8

const (
	// nodePrefix denotes a 'normal' node. e.g 'api' in /api/users
	nodePrefix nodeType = iota

	// nodePathParam denotes a node representating a path param. e.g 'id' in /api/users/:id
	nodePathParam

	// nodeAny represents wildcard '*'
	nodeAny
)

type node struct {
	// prefix is the value of the node
	prefix string

	handler http.HandlerFunc

	// direct ascendant of node
	parent   *node
	children []*node
	param    string
	type_    nodeType
}

func (n *node) findChild(segment string, nodeType nodeType) *node {
	for _, child := range n.children {
		if child.prefix == segment && child.type_ == nodeType {
			return child
		}
	}
	return nil
}

func (n *node) addChild(prefix string) *node {
	newNode := &node{
		prefix:   prefix,
		parent:   n,
		type_:    determineNodeType(prefix),
		children: []*node{},
	}
	n.children = append(n.children, newNode)
	return newNode
}

// utils for working with node
func isParam(prefix string) bool {
	if len(prefix) == 0 {
		return false
	}
	return prefix[0] == '{' && prefix[len(prefix)-1] == '}'
}

func determineNodeType(prefix string) nodeType {
	if len(prefix) > 0 && prefix[0] == '*' {
		return nodeAny
	}
	if isParam(prefix) {
		return nodePathParam
	}
	return nodePrefix
}
