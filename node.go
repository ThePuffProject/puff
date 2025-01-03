package puff

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
	mountNode := &node{
		prefix:   segments[0],
		children: []*node{},
	}

	current := mountNode

	for _, segment := range segments[1:] {
		child := current.addChild(segment)
		current = child
	}

	return &node{
		prefix:   p,
		children: []*node{},
	}
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
