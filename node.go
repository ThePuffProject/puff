package puff

import "fmt"

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
	prefix     string
	routes     map[string]*Route
	allMethods []string
	// direct ascendant of node
	parent *node
	// TODO: split children into dynamic vs static children. so we can look certain things up via map if static and fall-back to param if exists.
	children []*node
	param    string // what is param even doing??
	type_    nodeType
}

func insertNode(p string) *node {
	segments := segmentPath(p)
	if len(segments) == 0 {
		return nil // Handle edge cases where the path is empty
	}
	fmt.Println("entrypoint insertNode", segments)
	// Create the root node with the first segment
	mountNode := &node{
		prefix:   segments[0],
		children: []*node{},
	}

	current := mountNode

	// Add children for subsequent segments
	for _, segment := range segments[1:] {
		child := current.addChild(segment)
		fmt.Println("adding child while insertNode", child.prefix, "to parent", current.prefix)
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

func (n *node) addChild(prefix string) *node {
	// Validate the prefix
	if prefix == "" {
		panic("prefix cannot be empty when adding a child node")
	}

	// Check for duplicate prefixes among children
	for _, child := range n.children {
		if child.prefix == prefix {
			panic(fmt.Sprintf("child with prefix '%s' already exists under parent '%s'", prefix, n.prefix))
		}
	}

	// Create the new child node
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
