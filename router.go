package puff

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

// Router defines a group of routes that share the same prefix and middlewares.
type Router struct {
	Name        string
	Prefix      string //(optional) prefix, all Routes underneath will have paths that start with the prefix automatically
	Routers     []*Router
	Routes      []*Route
	Middlewares []*Middleware
	Tag         string
	Description string
	// Responses is a map of status code to puff.Response. Possible Responses for routes can be set at the Router (root as well),
	// and Route level, however responses directly set on the route will have the highest specificity.
	Responses Responses

	// parent maps to the router's immediate parent. Will be nil for RootRouter
	parent *Router
	// puffapp maps to the original PuffApp
	puffapp *PuffApp
}

// NewRouter creates a new router provided router name and path prefix.
func NewRouter(name string, prefix string) *Router {
	return &Router{
		Name:      name,
		Prefix:    prefix,
		Responses: Responses{},
	}
}

// func (r *Router) registerRoute(
// 	method string,
// 	path string,
// 	handleFunc func(*Context),
// 	fields any,
// ) *Route {
// 	_, file, line, ok := runtime.Caller(2)
// 	newRoute := Route{
// 		Description: readDescription(file, line, ok),
// 		Path:        path,
// 		Handler:     handleFunc,
// 		Protocol:    method,
// 		Fields:      fields,
// 		Router:      r,
// 		Responses:   Responses{},
// 	}

// 	r.Routes = append(r.Routes, &newRoute)
// 	return &newRoute
// }

// func (r *Router) Get(
// 	path string,
// 	fields any,
// 	handleFunc func(*Context),
// ) *Route {
// 	return r.registerRoute(http.MethodGet, path, handleFunc, fields)
// }

// func (r *Router) Post(
// 	path string,
// 	fields any,
// 	handleFunc func(*Context),
// ) *Route {
// 	return r.registerRoute(http.MethodPost, path, handleFunc, fields)
// }

// func (r *Router) Put(
// 	path string,
// 	fields any,
// 	handleFunc func(*Context),
// ) *Route {
// 	return r.registerRoute(http.MethodPut, path, handleFunc, fields)
// }

// func (r *Router) Patch(
// 	path string,
// 	fields any,
// 	handleFunc func(*Context),
// ) *Route {
// 	return r.registerRoute(http.MethodPatch, path, handleFunc, fields)
// }

// func (r *Router) Delete(
// 	path string,
// 	fields any,
// 	handleFunc func(*Context),
// ) *Route {
// 	return r.registerRoute(http.MethodDelete, path, handleFunc, fields)
// }

// func (r *Router) WebSocket(
// 	path string,
// 	fields any,
// 	handleFunc func(*Context),
// ) *Route {
// 	newRoute := Route{
// 		WebSocket: true,
// 		Protocol:  "GET",
// 		Path:      path,
// 		Handler:   handleFunc,
// 		Fields:    fields,
// 	}
// 	r.Routes = append(r.Routes, &newRoute)
// 	return &newRoute
// }

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
		rt.puffapp = rt.parent.puffapp
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
	defer req.Body.Close()
	for _, router := range r.Routers {
		if strings.HasPrefix(req.URL.Path, router.Prefix) {
			router.ServeHTTP(w, req)
			return
		}
	}
	c := NewContext(w, req, r.puffapp)
	for _, route := range r.Routes {
		isMatch := route.regexp.MatchString(req.URL.Path)
		if !isMatch || req.Method != route.Protocol {
			// fmt.Println(req.URL.Path, "does not match", route.fullPath, "because the route has regexp", route.regexp)
			continue
		}
		matches := route.regexp.FindStringSubmatch(req.URL.Path)
		if len(matches) > 0 {
			matches = matches[1:]
		}
		f, err := fieldsFromIncoming(c, route, matches)
		if err != nil {
			c.BadRequest(err.Error())
			return
		}

		if route.WebSocket {
			err := c.handleWebSocket()
			if err != nil {
				c.BadRequest("websocket error")
				return
			}
		}

		handler := route.Handler
		handler(c, f)
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

func (r *Router) patchRoutes() error {
	var err error
	for _, route := range r.Routes {
		route.Router = r

		route.FullPath()

		route.regexp, err = route.createRegexMatch()
		if err != nil {
			return regexpError(route.fullPath, err)
		}

		err := handleInputSchema(&route.params, route.fieldsType)
		if err != nil {
			return schemaError(err)
		}

		// populate route with their respective responses
		route.generateResponses()

		slog.Debug(fmt.Sprintf("Serving route: %s", route.fullPath))
	}
	return nil
}
