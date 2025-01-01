package main

import (
	"fmt"
	"net/http"
	"strings"
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

	handler http.HandlerFunc

	// direct ascendant of node
	parent   *node
	children []*node
	param    string
	type_    nodeType
}

type Router struct {
	rootNode *node
	name     string
}

func NewRouter(name string) *Router {
	return &Router{name: name, rootNode: new(node)}
}

func (r *Router) AddRoute(path string, handler http.HandlerFunc) {
	newRouteSegments := segmentPath(path)
	current := r.rootNode

	for _, newRouteSegment := range newRouteSegments {
		found := false
		for _, childNode := range current.children {
			if childNode.prefix == newRouteSegment {
				current = childNode
				found = true
				break
			}
		}
		if !found {
			newNode := &node{prefix: newRouteSegment, parent: current, children: []*node{}}
			current.children = append(current.children, newNode)
			current = newNode
		}
	}

	current.handler = handler
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	segments := segmentPath(path)
	current := r.rootNode

	if len(segments) == 0 && current.handler != nil { // Handle root route directly
		current.handler(w, req)
		return
	}

	for _, segment := range segments {
		found := false
		for _, child := range current.children {
			if child.prefix == segment || child.isParam() {
				if child.isParam() {
					child.param = child.prefix[1:]
				}
				current = child
				found = true
				break
			}
		}
		if !found {
			http.NotFound(w, req)
			return
		}
	}

	// Check if we've landed on a mounted router's root *and it has children*
	if current.children != nil && len(current.children) == 1 && current.children[0].parent == current {
		current = current.children[0]
	}

	if current.handler != nil {
		current.handler(w, req)
	} else {
		http.NotFound(w, req)
	}
}

func (r *Router) Mount(prefix string, subRouter *Router) {
	segments := segmentPath(prefix)
	current := r.rootNode

	for _, part := range segments {
		found := false
		for _, childNode := range current.children {
			if childNode.prefix == part {
				current = childNode
				found = true
				break
			}
		}
		if !found {
			subRouter.rootNode.parent = current
			subRouter.rootNode.prefix = prefix[1:] // strip /
			current.children = append(current.children, subRouter.rootNode)
		}
	}

	// Crucial fix: Preserve the sub-router's original prefix
	subRouter.rootNode.prefix = strings.TrimPrefix(subRouter.rootNode.prefix, "/")
	if current.prefix != "" {
		subRouter.rootNode.prefix = current.prefix + "/" + subRouter.rootNode.prefix
	}
	subRouter.rootNode.parent = current
	current.children = append(current.children, subRouter.rootNode)
}

func (r *Router) Visualize() {
	fmt.Println("Radix Trie Structure:")
	r.visualizeNode(r.rootNode, "")
}

func (r *Router) visualizeNode(n *node, indent string) {
	path := n.prefix
	if n.param != "" {
		path = ":" + n.param
	}
	fmt.Printf("%s%s", indent, path)
	if n.handler != nil {
		fmt.Print(" [Handler]")
	}
	fmt.Println()

	for _, child := range n.children {
		r.visualizeNode(child, indent+"  ")
	}
}

func main() {
	apiRouter := NewRouter("API")
	usersRouter := NewRouter("Users")
	cheeseRouter := NewRouter("Cheese")

	usersRouter.AddRoute("/root", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Users Root")
	})

	cheeseRouter.AddRoute("/blue", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "blue cheese")
	})

	usersRouter.AddRoute("/:id", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "User ID: %s\n", r.URL.Path)
	})

	usersRouter.AddRoute("/profile/:username", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "User Profile: %s\n", r.URL.Path)
	})

	usersRouter.Mount("/cheese", cheeseRouter)
	apiRouter.Mount("/users", usersRouter)

	apiRouter.Visualize()

	http.ListenAndServe(":8080", apiRouter)
}

func segmentPath(path string) []string {
	return strings.Split(strings.Trim(path, "/"), "/")
}

// type contextKey string

// const paramsKey contextKey = "params"

// func contextWithParams(ctx context.Context, params map[string]string) context.Context {
// 	return context.WithValue(ctx, paramsKey, params)
// }

// func ParamsFromContext(ctx context.Context) map[string]string {
// 	params, ok := ctx.Value(paramsKey).(map[string]string)
// 	if !ok {
// 		return nil
// 	}
// 	return params
// }

// func insertNode(path string) {
// 	pieces := segmentPath(path)

// 	for _, p := range pieces {
// 		if p.
// 	}
// }

func (n *node) isParam() bool {
	if len(n.prefix) == 0 {
		return false
	}
	return n.prefix[0] == '{' && n.prefix[len(n.prefix)-1] == '}'
}

func (n *node) findChild(prefix string) *node {
	for _, child := range n.children {
		if child.prefix == prefix {
			return child
		}
	}
	return nil
}

// func insertNode(parent *node, prefix string) *node {
// 	if child := parent.findChild(prefix); child != nil {

// 		newNode := &node{
// 			prefix:   prefix,
// 			parent:   parent,
// 			children: make([]*node, 0),
// 		}
// 		if newNode.isParam() {
// 			newNode.type_ = nodePathParam
// 		}
// 		parent.children = append(parent.children, newNode)
// 		return newNode
// 	}
// }
