package puff

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime"
	"slices"
)

// Router defines a group of routes that share the same prefix and middlewares. Think of
type Router struct {
	// Name of the Router. Used to generate OpenAPI tag.
	Name string
	// Path will be used to prefix all routes/routers underneath this
	Path string
	// Routers is the children routers underneath this router. All children routers inherit routes attached to the router.
	// FIXME: likely need to remove this
	Routers []*Router
	// Routes are the routes assigned to this router. Can be assigned by called Get/Post/Patch methods on a router.
	// FIXME: likely need to remove this from here
	Routes []*Route
	// Middlewares
	Middlewares []*Middleware
	// Tag is the tag associated to the router and used to group routes together in the OpenAPI schema.
	// If not explicitly provided, will be defaulted to Router Name.
	Tag string
	// Description is the description of the router. Currently not used I believe.
	Description string
	// Responses is a map of status code to puff.Response. Possible Responses for routes can be set at the Router (root as well),
	// and Route level, however responses directly set on the route will have the highest specificity.
	Responses Responses
	// parent maps to the router's immediate parent. Will be nil for RootRouter
	parent *Router
	// puff maps to the original PuffApp
	puff     *PuffApp
	rootNode *node
}

// NewRouter creates a new router provided router name and path prefix.
func NewRouter(name string) *Router {
	// note newRouter creates a dummy node. this will be populated on IncludeRouter.
	return &Router{
		Name:      name,
		Responses: Responses{},
		rootNode:  new(node),
		Tag:       name,
	}
}

func (r *Router) registerRoute(
	method string,
	path string,
	handleFunc func(*Context),
	fields any,
) *Route {
	segments := segmentPath(path)

	current := r.rootNode

	for _, segment := range segments {
		child := current.findChild(segment, determineNodeType(segment))
		if child == nil {
			child = current.addChild(segment)
		}
		current = child
	}
	if slices.Contains(current.allMethods, method) {
		err := fmt.Errorf("cannot define route '%s' with method '%s' as it already exists", path, method)
		panic(err)
	}

	current.allMethods = append(current.allMethods, method)
	_, file, line, ok := runtime.Caller(2)
	newRoute := &Route{
		Description: readDescription(file, line, ok),
		Path:        path,
		Handler:     handleFunc,
		Protocol:    method,
		Fields:      fields,
		Router:      r,
		Responses:   Responses{},
	}
	if current.routes == nil {
		current.routes = make(map[string]*Route)
	}
	current.routes[method] = newRoute

	r.Routes = append(r.Routes, newRoute)
	return newRoute
}

func (r *Router) Get(
	path string,
	fields any,
	handleFunc func(*Context),
) *Route {
	return r.registerRoute(http.MethodGet, path, handleFunc, fields)
}

func (r *Router) Post(
	path string,
	fields any,
	handleFunc func(*Context),
) *Route {
	return r.registerRoute(http.MethodPost, path, handleFunc, fields)
}

func (r *Router) Put(
	path string,
	fields any,
	handleFunc func(*Context),
) *Route {
	return r.registerRoute(http.MethodPut, path, handleFunc, fields)
}

func (r *Router) Patch(
	path string,
	fields any,
	handleFunc func(*Context),
) *Route {
	return r.registerRoute(http.MethodPatch, path, handleFunc, fields)
}

func (r *Router) Delete(
	path string,
	fields any,
	handleFunc func(*Context),
) *Route {
	return r.registerRoute(http.MethodDelete, path, handleFunc, fields)
}

func (r *Router) WebSocket(
	path string,
	fields any,
	handleFunc func(*Context),
) *Route {
	newRoute := Route{
		WebSocket: true,
		Protocol:  "GET",
		Path:      path,
		Handler:   handleFunc,
		Fields:    fields,
	}
	r.Routes = append(r.Routes, &newRoute)
	return &newRoute
}

func (r *Router) IncludeRouter(mountPath string, subRouter *Router) {
	// if len(mountPath) == 0 || mountPath[0] != '/' {
	// 	err := fmt.Errorf("mountPath '%s' for router %s is invalid. Paths must begin with '/' and may not be empty",
	// 		mountPath, subRouter.Name,
	// 	)
	// 	panic(err)
	// }
	if subRouter == nil {
		err := fmt.Errorf("subRouter is nil. Cannot attach nil router to %s", r.Name)
		panic(err)
	}

	if len(mountPath) == 0 {
		mountPath = "/"
	}

	if subRouter.parent != nil {
		err := fmt.Errorf(
			"provided router is already attached to %s. A router may only be attached to one parent",
			subRouter.parent,
		)
		panic(err)
	}
	subRouter.Path = mountPath
	subRouter.parent = r
	subRouter.puff = r.puff

	segments := segmentPath(mountPath)
	fmt.Println("segments", segments, "mountPath", mountPath, r.rootNode.prefix)
	subRouter.rootNode.prefix = segments[0]

	current := r.rootNode

	for _, part := range segments {
		found := false
		// check for matching segments to attach it to
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

	r.Routers = append(r.Routers, subRouter)
}

// Use adds a middleware to the router's list of middlewares. Middleware functions
// can be used to intercept requests and responses, allowing for functionality such
// as logging, authentication, and error handling to be applied to all routes managed
// by this router.
//
// Example usage:
//
//	router := puff.NewRouter()
//	router.Use(myMiddleware)
//	router.Get("/endpoint", myHandler)
//
// Parameters:
// - m: A Middleware function that will be applied to all routes in this router.
// TODO: dont know if below is actually accurate. cant think
// Note: Middleware functions are executed in the order they are added. If multiple
// middlewares are registered, they will be executed sequentially for each request
// handled by the router.
func (r *Router) Use(m Middleware) {
	r.Middlewares = append(r.Middlewares, &m)
}

func (r *Router) String() string {
	return fmt.Sprintf("Name: %s Prefix: %s", r.Name, r.rootNode.prefix)
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	segments := segmentPath(req.URL.Path)
	current := r.rootNode
	params := []string{}

	c := NewContext(w, req, r.puff)

	for _, segment := range segments {
		found := false

		for _, child := range current.children {
			fmt.Println("child.prefix", child.prefix, "segment", segment)
			if child.prefix == segment {
				// prefer static match when found and break loop.
				found = true
				fmt.Println("found node", child.prefix, "moving to it")
				current = child
				break
			} else if child.type_ == nodePathParam {
				found = true
				current = child
			}
		}
		if !found {
			http.NotFound(w, req)
			return
		}
	}
	if current.routes != nil {
		route, ok := current.routes[req.Method]
		if !ok {
			ErrMethodNotAllowed(c)
			return
		}

		err := populateInputSchema(c, route.Fields, route.params, params)
		if err != nil {
			c.BadRequest(err.Error())
			return
		}
		if route.WebSocket {
			err := c.handleWebSocket()
			if err != nil { // the message has already been passed on by the function; we may just return at this point
				return
			}
		}
		route.Handler(c)
		return

	}
	http.NotFound(w, req)
}

func Unprocessable(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "StatusUnprocessableEntity", http.StatusUnprocessableEntity)
}

// AllRoutes returns all routes attached to a router as well as routes attached to the subrouters
// For just the routes attached to a router, use `Routes` attribute on Router
func (r *Router) AllRoutes() []*Route {
	var routes []*Route

	routes = append(routes, r.Routes...)

	for _, subRouter := range r.Routers {
		routes = append(routes, subRouter.AllRoutes()...)
	}
	return routes
}

func (r *Router) patchRoutes() {
	for _, route := range r.Routes {
		route.Router = r
		route.getCompletePath()
		err := route.handleInputSchema()
		if err != nil {
			panic("error with Input Schema for route " + route.Path + " on router " + r.Name + ". Error: " + err.Error())
		}
		slog.Debug(fmt.Sprintf("Serving route: %s", route.fullPath))
		// populate route with their respective responses
		route.GenerateResponses()
	}
}

func (r *Router) Visualize() {
	fmt.Println("Radix Trie Structure:")
	r.visualizeNode(r.rootNode, "", true)
}

func (r *Router) visualizeNode(n *node, prefix string, isLast bool) {
	// Determine the branch symbol
	branch := "├──"
	if isLast {
		branch = "└──"
	}

	// Print the current node's prefix and methods
	if len(n.allMethods) == 0 {
		fmt.Printf("%s%s%s\n", prefix, branch, n.prefix)
	} else {
		fmt.Printf("%s%s%s | Methods: %v\n", prefix, branch, n.prefix, n.allMethods)
	}

	// Update the prefix for children
	childPrefix := prefix
	if isLast {
		childPrefix += "    "
	} else {
		childPrefix += "│   "
	}

	// Recurse into each child node
	for i, child := range n.children {
		isLastChild := i == len(n.children)-1
		r.visualizeNode(child, childPrefix, isLastChild)
	}
}
