package puff

import (
	"fmt"
)

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
	routes map[string]*Route
	// allMethods only needed, to avoid looping over methodRoutes to return Allow header
	allMethods []string
	// direct ascendant of node
	parent *node
	// TODO: split children into dynamic vs static children. so we can look certain things up via map if static and fall-back to param if exists.
	children []*node
	param    string // what is param even doing??
	type_    nodeType
}

func newNode(prefix string, parent *node) *node {
	return &node{
		prefix:     prefix,
		routes:     map[string]*Route{},
		allMethods: []string{},
		parent:     parent,
		children:   []*node{},
		type_:      determineNodeType(prefix),
	}
}

func insertNode(p string) *node {
	segments := segmentPath(p)
	if len(segments) == 0 {
		return nil // Handle edge cases where the path is empty
	}
	// Create the root node with the first segment
	mountNode := newNode(segments[0], nil)

	current := mountNode

	// Add children for subsequent segments
	for _, segment := range segments[1:] {
		child := current.addChild(segment)
		current = child
	}

	return mountNode // Return the root of the hierarchy
}

func (n *node) findChild(segment string, nodeType nodeType) *node {
	for _, child := range n.children {
		if child.prefix == segment && child.type_ == nodeType {
			return child
		}
	}
	return nil
}

func (n *node) isMethodTaken(method, path string) bool {
	r, exists := n.routes[method]
	if exists {
		return r.Path == path
	}
	return false
}

func (n *node) addChild(prefix string) *node {
	// Validate the prefix
	// if prefix == "" {
	// 	err := fmt.Errorf("prefix was empty when adding child to node %s", n.prefix)
	// 	panic(err)
	// }

	// Check for duplicate prefixes among children
	for _, child := range n.children {
		if child.prefix == prefix {
			panic(fmt.Sprintf("child with prefix '%s' already exists under parent '%s'", prefix, n.prefix))
		}
	}

	// Create the new child node
	newNode := newNode(prefix, nil)

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
