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
	segments := segmentPath(path)
	current := r.rootNode

	for _, segment := range segments {
		found := false
		for _, child := range current.children {
			if child.prefix == segment && child.type_ == determineNodeType(segment) {
				current = child
				found = true
				break
			}
		}
		if !found {
			newNode := &node{
				prefix:   segment,
				parent:   current,
				children: []*node{},
				type_:    determineNodeType(segment),
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
	params := make(map[string]string)

	for _, segment := range segments {
		found := false

		for _, child := range current.children {
			if child.prefix == segment {
				// prefer static match when found and break loop.
				found = true
				current = child
				break
			} else if child.type_ == nodePathParam {
				params[child.prefix[1:len(child.prefix)-1]] = segment
				found = true
				current = child
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

func (r *Router) Mount(prefix string, subRouter *Router) {
	if len(prefix) == 0 || prefix[0] != '/' {
		panic("Route prefix should not be an empty string and must start with '/'")
	}

	segments := segmentPath(prefix)
	current := r.rootNode

	for _, part := range segments {
		found := false
		for _, child := range current.children {
			if child.prefix == part {
				current = child
				found = true
				break
			}
		}
		if !found {
			newNode := &node{
				prefix:   part,
				parent:   current,
				children: []*node{},
			}
			current.children = append(current.children, newNode)
			current = newNode
		}
	}

	// copy over children from subRouter as well.
	for _, child := range subRouter.rootNode.children {
		child.parent = current
		current.children = append(current.children, child)
	}
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

	usersRouter.AddRoute("/profile/{username}", func(w http.ResponseWriter, r *http.Request) {
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
