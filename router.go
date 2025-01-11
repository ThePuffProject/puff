package puff

import (
	"fmt"
	"maps"
	"net/http"
	"runtime"
	"strings"
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
	// parent maps to the router's immediate parent. Will be nil for rootRouter
	parent *Router
	// puff maps to the original PuffApp
	puff     *PuffApp
	rootNode *node
}

// NewRouter creates a new router provided router name and path prefix.
func NewRouter(name string) *Router {
	// note newRouter creates a dummy node. this will be populated on Mount.
	return &Router{
		Path:      "", // will be updated later
		Name:      name,
		Responses: Responses{},
		rootNode:  insertNode(name),
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

	if (path == "" || path[0] != '/') && len(segments) < 2 {
		if path == "" && !current.isMethodTaken(method, path) {
			return addRouteToNode(current, r, method, path, handleFunc, fields)
		}
		child := current.findChild(path, determineNodeType(path))
		if child == nil {
			current = current.addChild(path)
		}
		return addRouteToNode(
			current, r, method, path, handleFunc, fields,
		)

	}

	if path[0] != '/' && len(segments) > 0 {
		// avoiding routes such as 'users/foo'
		err := fmt.Sprintf("Improperly formatted route with method %s and path %s", method, path)
		panic(err)
	}

	for _, segment := range segments {
		segment_w_slash := "/" + segment
		child := current.findChild(segment_w_slash, determineNodeType(segment))
		if child == nil {
			child = current.addChild(segment_w_slash)
		}

		current = child
	}

	return addRouteToNode(
		current, r, method, path, handleFunc, fields,
	)
}

// Helper to add a route to the current node
func addRouteToNode(
	node *node,
	r *Router,
	method string,
	path string,
	handleFunc func(*Context),
	fields any,
) *Route {
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
	if node.isMethodTaken(method, path) {
		panic(fmt.Sprintf("cannot add route with prefix %s on Router %s due to method conflict. Method %s is already being used. Existing route is %s %s", path, r.Name, method, node.routes[method].Path, node.routes[method]))
	}
	if node.routes == nil {
		node.routes = make(map[string]*Route)
	}
	node.routes[method] = newRoute
	node.allMethods = append(node.allMethods, method)
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

func (r *Router) Mount(mountPath string, subRouter *Router) *Router {
	if r == subRouter {
		panic("u are being a silly goose")
	}

	if subRouter == nil {
		err := fmt.Errorf("subRouter is nil. Cannot attach nil router to %s", r.Name)
		panic(err)
	}

	if len(mountPath) == 0 || mountPath[0] != '/' {
		err := fmt.Errorf("mountPath '%s' for router %s is invalid. Paths must begin with '/' and may not be empty",
			mountPath, subRouter.Name,
		)
		panic(err)
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
	subRouter.rootNode.prefix = segments[0]

	current := r.rootNode

	for _, part := range segments {
		segment_w_slash := "/" + part
		found := false
		// check for matching segments to attach it to
		for _, child := range current.children {
			if child.prefix == segment_w_slash {
				current = child
				found = true
				break
			}
		}
		if !found {
			newNode := newNode(segment_w_slash, current)
			current.children = append(current.children, newNode)
			current = newNode
		}
	}

	// copy over children from subRouter as well.
	for _, child := range subRouter.rootNode.children {
		child.parent = current
		current.children = append(current.children, child)
	}

	maps.Copy(current.routes, subRouter.rootNode.routes)
	current.allMethods = append(current.allMethods, subRouter.rootNode.allMethods...)

	r.Routers = append(r.Routers, subRouter)
	return subRouter
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
	runningPrefixMatches := ""

	c := NewContext(w, req, r.puff)

	for _, segment := range segments {
		segmentWithSlash := "/" + segment
		found := false

		for _, child := range current.children {
			if child.prefix == segmentWithSlash { // Exact match
				runningPrefixMatches += child.prefix
				found = true
				current = child
				break
			} else if child.type_ == nodePathParam { // Path parameter match
				runningPrefixMatches += segmentWithSlash
				found = true
				params = append(params, segment)
				current = child
				break
			} else if strings.HasPrefix(segmentWithSlash, child.prefix) || strings.HasPrefix(segment, child.prefix) { // Prefix match
				runningPrefixMatches += child.prefix
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
			if err != nil {
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
		// populate route with their respective responses
		route.GenerateResponses()
	}
}

func (r *Router) Visualize() {
	r.visualizeNode(r.rootNode, "", true)
}

func (r *Router) visualizeNode(n *node, prefix string, isLast bool) {
	// fmt.Println("visualizing node", n.prefix)
	// Determine the branch symbol
	branch := "├──"
	if isLast {
		branch = "└──"
	}
	if n.parent == nil {
		branch = ""
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
		// fmt.Println("what is child of n", n.prefix, child.prefix)
		isLastChild := i == len(n.children)-1
		r.visualizeNode(child, childPrefix, isLastChild)
	}
}
