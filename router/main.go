package main

import (
	"fmt"
	"net/http"
	"strings"
)

type node struct {
	prefix   string
	handler  http.HandlerFunc
	parent   *node
	children []*node
	param    string
}

type Router struct {
	rootNode *node
	name     string
}

func NewRouter(prefix string, name string) *Router {
	return &Router{rootNode: &node{prefix: prefix, parent: nil}, name: name}
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
			newNode := &node{prefix: newRouteSegment, parent: current}
			current.children = append(current.children, newNode)
			current = newNode
		}
	}

	current.handler = handler
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	segments := segmentPath(req.URL.Path)
	current := r.rootNode

	for _, segment := range segments {
		found := false

		// Check the root node's prefix first
		if current.prefix == segment {
			found = true
		} else {
			for _, child := range current.children {
				if child.prefix == segment {
					current = child
					found = true
					break
				} else if child.isParam() {
					current = child
					found = true
					break
				}
			}
		}

		if !found {
			http.NotFound(w, req)
			return
		}
	}

	if current.handler != nil {
		current.handler(w, req)
	} else {
		http.NotFound(w, req)
	}
}

func (r *Router) Mount(subRouter *Router) {
	// Mount the subRouter's rootNode under the current router's rootNode
	subRouter.rootNode.parent = r.rootNode
	r.rootNode.children = append(r.rootNode.children, subRouter.rootNode)
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
	apiRouter := NewRouter("/api", "API")
	usersRouter := NewRouter("/users", "User")

	usersRouter.AddRoute("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Users Root")
	})

	usersRouter.AddRoute("/:id", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "User ID: %s\n", r.URL.Path)
	})

	usersRouter.AddRoute("/profile/:username", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "User Profile: %s\n", r.URL.Path)
	})

	apiRouter.Mount(usersRouter)
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

func (n *node) isParam() bool {
	fmt.Println("node", n.prefix, n.children)
	if len(n.prefix) == 0 {
		return false
	}
	return (n.prefix[0] == ':') || (n.prefix[0] == '{' && n.prefix[len(n.prefix)-1] == '}')
}
