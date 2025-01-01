package main

import (
	"fmt"
	"net/http"
	"strings"
)

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
			newNode := &node{
				prefix:   newRouteSegment,
				parent:   current,
				children: []*node{},
				type_:    determineNodeType(newRouteSegment),
			}
			current.children = append(current.children, newNode)
			// update pointer to continue adding segments to trie from here
			current = newNode
		}
	}

	current.handler = handler
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	segments := segmentPath(req.URL.Path)
	current := r.rootNode

	if len(segments) == 0 && current.handler != nil { // Handle root route directly
		current.handler(w, req)
		return
	}

	for _, segment := range segments {
		found := false
		for _, child := range current.children {
			// FIXME: why are we checking nodepathparam twice
			// nodepathparam should be used only if tere is no match with current route.
			if child.prefix == segment || child.type_ == nodePathParam {
				if child.type_ == nodePathParam {
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
	// FIXME: what is this??!
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
	if len(prefix) == 0 || prefix[0] != '/' {
		panic("Route prefix should not be an empty string and must start with '/'")
	}

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
			fmt.Println("not found", part, "router", subRouter.name, prefix)
			subRouter.rootNode.parent = current
			subRouter.rootNode.prefix = prefix[1:] // strip '/'
			current.children = append(current.children, subRouter.rootNode)

		}
	}

	// fixme: what are we doing here??
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

	usersRouter.AddRoute("/{id}", func(w http.ResponseWriter, r *http.Request) {
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
