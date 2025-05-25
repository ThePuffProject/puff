package puff

import (
	"fmt"
	// "maps" // Removed unused import
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
		// Initialize rootNode with a base prefix, typically "/".
		// The router's actual Path field will define its mount point.
		rootNode:  newNode("/", nil),
		Tag:       name,
	}
}

func (r *Router) registerRoute(
	method string,
	path string,
	handleFunc func(*Context),
	fields any,
) *Route {
	// Path Preparation
	preparedPath := path
	if preparedPath == "" {
		// If path is empty, it refers to the router's root.
		// node.insert expects a path; use "/" for the rootNode's own definition.
		// This assumes rootNode.prefix is typically "/" or similar.
		// If rootNode.prefix is empty, node.insert might need to handle "" differently.
		// For now, using "/" for empty path registrations.
		preparedPath = "/" 
	} else if preparedPath[0] != '/' {
		preparedPath = "/" + preparedPath
	}

	// Route Object Creation (logic moved from addRouteToNode)
	// runtime.Caller(2) means the caller of Get/Post/etc.
	_, file, line, ok := runtime.Caller(2) 
	newRoute := &Route{
		Description: readDescription(file, line, ok),
		Path:        path, // Store original path for user reference
		Handler:     handleFunc,
		Protocol:    method,
		Fields:      fields,
		Router:      r,
		Responses:   Responses{},
	}

	// Ensure rootNode is initialized (safety check, should be done in NewRouter)
	if r.rootNode == nil {
		r.rootNode = newNode("/", nil) 
	}
	
	// The check for existing method on a specific node (node.isMethodTaken)
	// is now implicitly handled by node.insert. If node.insert decides to overwrite
	// or error on duplicate method for the same path, that logic resides in node.go.
	// Currently, node.insert overwrites.

	r.rootNode.insert(preparedPath, newRoute, []string{method})
	
	// Add route to r.Routes for other purposes (like AllRoutes, patching routes)
	r.Routes = append(r.Routes, newRoute)
	return newRoute
}

// addRouteToNode is now superseded by logic within registerRoute and node.insert.
// func addRouteToNode(...) { ... } // Removed

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

	// Iterate through all routes in the subRouter (including those from its own sub-routers)
	// AllRoutes() should provide routes with their Path relative to the subRouter.
	allSubRoutes := subRouter.AllRoutes() 

	for _, routeToMount := range allSubRoutes {
		// Construct the full path for this route in the parent router.
		// routeToMount.Path is relative to the subRouter.
		// mountPath is the path where subRouter is being mounted.
		fullPath := joinPaths(mountPath, routeToMount.Path)
		
		// The routeToMount object itself is being added. Its Router field should point
		// to the subRouter it originated from (or one of its children if nested).
		// This context is important for middleware lookup or other router-specific logic.
		// node.insert will handle creating the necessary nodes in r.rootNode's trie.
		if r.rootNode == nil { // Should be initialized by NewRouter, but as a safeguard.
			r.rootNode = newNode("/", nil)
		}
		r.rootNode.insert(fullPath, routeToMount, []string{routeToMount.Protocol})
	}

	// Keep the subRouter in the parent's list for other purposes (e.g., AllRoutes on parent, middleware hierarchy)
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
	requestPath := req.URL.Path         // Step 1: Get request path
	currentNode := r.rootNode           // Start traversal from root
	paramsMap := make(map[string]string) // Initialize params map
	c := NewContext(w, req, r.puff)

	// Path Normalization
	if !strings.HasPrefix(requestPath, "/") && currentNode != nil && strings.HasPrefix(currentNode.prefix, "/") {
		requestPath = "/" + requestPath
	}
	if len(requestPath) > 1 && requestPath[len(requestPath)-1] == '/' { // Remove trailing slash unless it's "/"
		requestPath = requestPath[:len(requestPath)-1]
	}

	// 2. Traversal Loop
	for currentNode != nil {
		// LCP and Path Consumption Block
		if currentNode.type_ == nodePathParam {
			// Param node's prefix (e.g. "{id}") is a placeholder.
			// The actual path segment (param value) was consumed when this node was CHOSEN in the parent.
			// So, `requestPath` is already what's *after* the param value.
			// No LCP or path consumption against `currentNode.prefix` itself here.
		} else if currentNode.prefix == "/" && requestPath == "" { // Root path "/" was normalized to ""
			// Exact match for the root node. Path is fully consumed.
		} else if lcp := longestCommonPrefix(requestPath, currentNode.prefix); lcp == len(currentNode.prefix) {
			requestPath = requestPath[lcp:] // Consume the matched static prefix
		} else {
			currentNode = nil // Prefix mismatch
			break
		}

		// Path Normalization after LCP consumption (for next iteration's LCP or child matching)
		// If requestPath is not empty and starts with "/", trim it.
		// This ensures that child prefixes (like "foo") can be matched directly
		// against a requestPath segment (also like "foo", not "/foo").
		if len(requestPath) > 0 && requestPath[0] == '/' {
			requestPath = requestPath[1:]
		}


		// If requestPath is now empty, it means the path terminates exactly at currentNode.
		if requestPath == "" {
			break
		}

		// 3. Child Selection (if requestPath is not empty)
		// `requestPath` here is the *next segment* to match (e.g., "foo" or "bar" from "foo/bar")
		// It has been normalized (leading slash removed if it wasn't just "/").
		nextNodeFound := false
		
		// i. Static Child
		if len(requestPath) > 0 { // Path segment for matching
			firstCharKey := requestPath[0:1] // Key for staticChildren map (e.g., "f" for "foo")
			if staticChild, exists := currentNode.staticChildren[firstCharKey]; exists {
				// staticChild.prefix is typically the full segment name (e.g., "foo" or "foo/bar" if it's a multi-part static child)
				// We need to check if the current requestPath starts with this child's prefix.
				if strings.HasPrefix(requestPath, staticChild.prefix) {
					currentNode = staticChild
					nextNodeFound = true
					// Path consumption for staticChild.prefix will be handled by LCP at the start of the next iteration.
					continue 
				}
			}
		}

		// ii. Parameter Child (if no static match found)
		if !nextNodeFound && currentNode.paramChild != nil {
			// `requestPath` is the current segment to be considered as a parameter value (e.g., "123" from "123/details")
			pathForParamExtraction := requestPath 
			paramValue := ""
			
			slashIndex := strings.Index(pathForParamExtraction, "/")
			if slashIndex == -1 { // Parameter is the last segment
				paramValue = pathForParamExtraction
				requestPath = "" // Consumed the whole segment
			} else { // Parameter is segment up to the next slash
				paramValue = pathForParamExtraction[:slashIndex]
				// Remaining path starts *after* the slash. This will be normalized at the top of the next loop.
				requestPath = pathForParamExtraction[slashIndex:] 
			}

			if paramValue != "" { // Parameter value must not be empty
				paramName := extractParamName(currentNode.paramChild.prefix)
				paramsMap[paramName] = paramValue
				
				currentNode = currentNode.paramChild
				nextNodeFound = true
				// `requestPath` is now what remains *after* the paramValue and its slash (if any).
				// It will be normalized (leading slash removed) at the start of the next loop if not empty.
				continue 
			} else {
				// If paramValue is empty (e.g., if pathForParamExtraction was "" or just "/"),
				// this param child cannot match. Restore requestPath to its state before this attempt.
				requestPath = pathForParamExtraction
			}
		}

		// iii. Any Child (if no static or param match found) - UNCHANGED for this subtask
		if !nextNodeFound && currentNode.anyChild != nil {
			paramName := extractParamName(currentNode.anyChild.prefix)
			if paramName == currentNode.anyChild.prefix && paramName == "*" { // Default name for anonymous "*"
				paramName = "wildcard"
			}
			// The "any" child consumes the rest of the nextTokenSegment as its value.
			paramsMap[paramName] = nextTokenSegment 
			
			// The LCP with anyChild.prefix (e.g. "*") should consume the rest of nextTokenSegment.
			// To ensure correct termination, we set requestPath to what will be consumed by anyChild's prefix.
			// If anyChild.prefix is "*", it will match all of nextTokenSegment.
			// The effect is that after LCP in the next iteration, requestPath will be empty.
			// No need to explicitly set requestPath = "" here if LCP handles "*" correctly.
			
			currentNode = currentNode.anyChild
			nextNodeFound = true
			// The LCP at the start of the next iteration will match anyChild.prefix against nextTokenSegment.
			// If anyChild.prefix is "*", it will match all of nextTokenSegment, and requestPath will become "" after consumption.
			continue
		}

		// iv. No Match: If no child was found for the nextTokenSegment
		if !nextNodeFound {
			currentNode = nil // Signal Not Found
			break
		}
	}

	// 4. Route Execution
	if currentNode == nil { // Path did not lead to a valid node
		http.NotFound(w, req)
		return
	}

	// After the loop, if requestPath is not empty, it means the path was not fully consumed by the trie nodes.
	if requestPath != "" {
		http.NotFound(w, req)
		return
	}

	// currentNode is the matched node, and requestPath is empty.
	route, ok := currentNode.routes[req.Method]
	if !ok {
		if len(currentNode.allMethods) > 0 { // Other methods exist for this path
			ErrMethodNotAllowed(c)
		} else { // No methods at all for this path
			http.NotFound(w, req)
		}
		return
	}

	// Populate context with extracted path parameters
	c.PathParams = paramsMap

	// Parameter conversion for populateInputSchema (HACK - needs robust fix)
	var collectedParamsForSchema []string
	if route.params != nil { // route.params is []*InputParameter from route definition
		for _, pInfo := range route.params { // pInfo is *InputParameter (contains Name, Type, etc.)
			if val, found := paramsMap[pInfo.Name]; found {
				collectedParamsForSchema = append(collectedParamsForSchema, val)
			} else {
				// Param expected by route definition but not found in map.
				// This could be an issue or an optional param. Add empty for now.
				collectedParamsForSchema = append(collectedParamsForSchema, "") 
			}
		}
	}

	err := populateInputSchema(c, route.Fields, route.params, collectedParamsForSchema)
	if err != nil {
		c.BadRequest(err.Error())
		return
	}

	if route.WebSocket {
		err := c.handleWebSocket()
		if err != nil {
			return // handleWebSocket writes response
		}
	}
	route.Handler(c)
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
	// Updated to use legacyChildren, and then iterate through new child types
	// For visualization, we might want to show all types of children.
	// This visualization part needs a more thought-out update for the new structure.
	// For now, let's iterate legacyChildren if it's still relevant, or comment out/simplify.
	// Given the refactor, legacyChildren is likely empty or unused for routing.
	// The new model uses staticChildren, paramChild, anyChild.

	// Simple visualization for new structure (can be expanded)
	childIndex := 0
	totalChildren := len(n.staticChildren)
	if n.paramChild != nil {
		totalChildren++
	}
	if n.anyChild != nil {
		totalChildren++
	}

	for _, child := range n.staticChildren {
		childIndex++
		isLastChild := childIndex == totalChildren
		r.visualizeNode(child, childPrefix, isLastChild)
	}
	if n.paramChild != nil {
		childIndex++
		isLastChild := childIndex == totalChildren
		r.visualizeNode(n.paramChild, childPrefix+"(param):", isLastChild)
	}
	if n.anyChild != nil {
		childIndex++
		isLastChild := childIndex == totalChildren
		r.visualizeNode(n.anyChild, childPrefix+"(any):", isLastChild)
	}
}
