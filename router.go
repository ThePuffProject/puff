package puff

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime"
	"strings"
)

// Router defines a group of routes that share the same prefix and middlewares. Think of
type Router struct {
	// Name of the Router. Used to generate OpenAPI tag.
	Name string
	// Prefix wil be used to prefix all routes/routers underneath this
	Prefix string
	// Routers is the children routers underneath this router. All children routers inherit routes attached to the router.
	Routers []*Router
	// Routes are the routes assigned to this router. Can be assigned by called Get/Post/Patch methods on a router.
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
	puff *PuffApp
}

// NewRouter creates a new router provided router name and path prefix.
func NewRouter(name string, prefix string) *Router {
	return &Router{
		Name:      name,
		Prefix:    prefix,
		Responses: Responses{},
	}
}

func (r *Router) registerRoute(
	method string,
	path string,
	handleFunc func(*Context),
	fields any,
) *Route {
	_, file, line, ok := runtime.Caller(2)
	newRoute := Route{
		Description: readDescription(file, line, ok),
		Path:        path,
		Handler:     handleFunc,
		Protocol:    method,
		Fields:      fields,
		Router:      r,
		Responses:   Responses{},
	}

	r.Routes = append(r.Routes, &newRoute)
	return &newRoute
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

func (r *Router) IncludeRouter(rt *Router) {
	if rt.parent != nil {
		err := fmt.Errorf(
			"provided router is already attached to %s. A router may only be attached to one parent",
			rt.parent,
		)
		panic(err)
	}

	rt.parent = r
	if rt.parent != nil {
		rt.puff = rt.parent.puff
	}
	r.Routers = append(r.Routers, rt)
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
	return fmt.Sprintf("Name: %s Prefix: %s", r.Name, r.Prefix)
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	for _, router := range r.Routers {
		if strings.HasPrefix(req.URL.Path, router.Prefix) {
			router.ServeHTTP(w, req)
			return
		}
	}
	c := NewContext(w, req, r.puff)
	for _, route := range r.Routes {
		if route.regexp == nil {
			// TODO: need to fix this. this will be nil for the doc routes.
			route.getCompletePath()
			route.createRegexMatch()
		}
		isMatch := route.regexp.MatchString(req.URL.Path)
		if isMatch && req.Method == route.Protocol {
			matches := route.regexp.FindStringSubmatch(req.URL.Path)
			err := populateInputSchema(c, route.Fields, route.params, matches)
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
			handler := route.Handler
			handler(c)
			return
		}
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
		route.createRegexMatch()
		err := route.handleInputSchema()
		if err != nil {
			panic("error with Input Schema for route " + route.Path + " on router " + r.Name + ". Error: " + err.Error())
		}
		slog.Debug(fmt.Sprintf("Serving route: %s", route.fullPath))
		// populate route with their respective responses
		route.GenerateResponses()
	}
}
