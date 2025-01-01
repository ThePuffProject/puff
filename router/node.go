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
