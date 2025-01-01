package puff

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"strings"
)

type PuffApp struct {
	// Config is the underlying application configuration.
	Config *AppConfig
	// RootRouter is the application's default router.
	RootRouter *Router

	// Server is the http.Server that will be used to serve requests.
	Server *http.Server
}

// Add a Router to the main app.
// Under the hood attaches the router to the App's RootRouter
func (a *PuffApp) IncludeRouter(r *Router) {
	if !strings.HasPrefix(r.Prefix, "/") {
		panic("Puff Routers must have a / prefix")
	}
	r.puff = a
	a.RootRouter.IncludeRouter(r)
}

// Use registers a middleware function to be used by the root router of the PuffApp.
// The middleware will be appended to the list of middlewares in the root router.
//
// Parameters:
// - m: Middleware function to be added.
func (a *PuffApp) Use(m Middleware) {
	a.RootRouter.Middlewares = append(a.RootRouter.Middlewares, &m)
}

// addOpenAPIRoutes adds routes to serve OpenAPI documentation for the PuffApp.
// If a DocsURL is specified, the function sets up two routes:
// 1. A route to provide the OpenAPI spec as JSON.
// 2. A route to render the OpenAPI documentation in a user-friendly UI.
//
// This method will not add any routes if DocsURL is empty.
//
// Errors during spec generation are logged, and the method will exit early if any occur.
func (a *PuffApp) addOpenAPIRoutes() {
	if a.Config.DisableOpenAPIGeneration {
		return
	}
	a.GenerateOpenAPISpec()
	docsRouter := Router{
		Prefix: a.Config.DocsURL,
		Name:   "OpenAPI Documentation Router",
	}

	// Provides JSON OpenAPI Schema.
	docsRouter.Get(".json", nil, func(c *Context) {
		res := JSONResponse{
			StatusCode: 200,
			Content:    a.Config.OpenAPI,
		}

		c.SendResponse(res)
	})

	// Renders OpenAPI schema.
	docsRouter.Get("", nil, func(c *Context) {
		if a.Config.SwaggerUIConfig == nil {

			swaggerConfig := SwaggerUIConfig{
				Title:           a.Config.Name,
				URL:             a.Config.DocsURL + ".json",
				Theme:           "obsidian",
				Filter:          true,
				RequestDuration: false,
				FaviconURL:      "https://fav.farm/💨",
			}
			a.Config.SwaggerUIConfig = &swaggerConfig
		}
		res := HTMLResponse{
			Template: openAPIHTML, Data: a.Config.SwaggerUIConfig,
		}
		c.SendResponse(res)
	})

	a.IncludeRouter(&docsRouter)
}

// attachMiddlewares recursively applies middlewares to all routes within a router.
// This function traverses through the router's sub-routers and routes, applying the
// middleware functions in the given order.
//
// Parameters:
// - middleware_combo: A pointer to a slice of Middleware to be applied.
// - router: The router whose middlewares and routes should be processed.
func attachMiddlewares(middleware_combo *[]Middleware, router *Router) {
	for _, m := range router.Middlewares {
		nmc := append(*middleware_combo, *m)
		middleware_combo = &nmc
	}
	for _, route := range router.Routes {
		for _, m := range *middleware_combo {
			route.Handler = (m)(route.Handler)
		}
	}
	for _, router := range router.Routers {
		attachMiddlewares((middleware_combo), router)
	}
}

// patchAllRoutes applies middlewares to all routes and sub-routers in the root router
// of the PuffApp. It also patches the routes of each router to ensure they have been
// processed for middlewares.
func (a *PuffApp) patchAllRoutes() {
	a.RootRouter.patchRoutes()
	for _, r := range a.RootRouter.Routers {
		r.patchRoutes()
	}
	attachMiddlewares(&[]Middleware{}, a.RootRouter)
}

// ListenAndServe starts the PuffApp server on the specified address.
// Before starting, it patches all routes, adds OpenAPI documentation routes (if available),
// and sets up logging.
//
// If TLS certificates are provided (TLSPublicCertFile and TLSPrivateKeyFile), the server
// starts with TLS enabled; otherwise, it runs a standard HTTP server.
//
// Parameters:
// - listenAddr: The address the server will listen on (e.g., ":8080").
func (a *PuffApp) ListenAndServe(listenAddr string) error {

	a.patchAllRoutes()
	a.addOpenAPIRoutes()

	slog.Debug(fmt.Sprintf("Running Puff 💨 on %s", listenAddr))
	slog.Debug(fmt.Sprintf("Visit docs 💨 on %s", fmt.Sprintf("http://localhost%s%s", listenAddr, a.Config.DocsURL)))

	if a.Server == nil {
		a.Server = &http.Server{
			Addr:    listenAddr,
			Handler: a.RootRouter,
		}
	}

	var err error
	if a.Config.TLSPublicCertFile != "" && a.Config.TLSPrivateKeyFile != "" {
		err = a.Server.ListenAndServeTLS(a.Config.TLSPublicCertFile, a.Config.TLSPrivateKeyFile)
	} else {
		err = a.Server.ListenAndServe()
	}

	return err
}

// Get registers an HTTP GET route in the PuffApp's root router.
//
// Parameters:
// - path: The URL path of the route.
// - fields: Optional fields associated with the route.
// - handleFunc: The handler function that will be executed when the route is accessed.
func (a *PuffApp) Get(path string, fields any, handleFunc func(*Context)) *Route {
	return a.RootRouter.Get(path, fields, handleFunc)
}

// Post registers an HTTP POST route in the PuffApp's root router.
//
// Parameters:
// - path: The URL path of the route.
// - fields: Optional fields associated with the route.
// - handleFunc: The handler function that will be executed when the route is accessed.
func (a *PuffApp) Post(path string, fields any, handleFunc func(*Context)) *Route {
	return a.RootRouter.Post(path, fields, handleFunc)
}

// Patch registers an HTTP PATCH route in the PuffApp's root router.
//
// Parameters:
// - path: The URL path of the route.
// - fields: Optional fields associated with the route.
// - handleFunc: The handler function that will be executed when the route is accessed.
func (a *PuffApp) Patch(path string, fields any, handleFunc func(*Context)) *Route {
	return a.RootRouter.Patch(path, fields, handleFunc)
}

// Put registers an HTTP PUT route in the PuffApp's root router.
//
// Parameters:
// - path: The URL path of the route.
// - fields: Optional fields associated with the route.
// - handleFunc: The handler function that will be executed when the route is accessed.
func (a *PuffApp) Put(path string, fields any, handleFunc func(*Context)) *Route {
	return a.RootRouter.Put(path, fields, handleFunc)
}

// Delete registers an HTTP DELETE route in the PuffApp's root router.
//
// Parameters:
// - path: The URL path of the route.
// - fields: Optional fields associated with the route.
// - handleFunc: The handler function that will be executed when the route is accessed.
func (a *PuffApp) Delete(path string, fields any, handleFunc func(*Context)) *Route {
	return a.RootRouter.Delete(path, fields, handleFunc)
}

// WebSocket registers a WebSocket route in the PuffApp's root router.
// This route allows the server to handle WebSocket connections at the specified path.
//
// Parameters:
// - path: The URL path of the WebSocket route.
// - fields: Optional fields associated with the route.
// - handleFunc: The handler function to handle WebSocket connections.
func (a *PuffApp) WebSocket(path string, fields any, handleFunc func(*Context)) *Route {
	return a.RootRouter.WebSocket(path, fields, handleFunc)
}

// AllRoutes returns all routes registered in the PuffApp, including those in sub-routers.
// This function provides an aggregated view of all routes in the application.
func (a *PuffApp) AllRoutes() []*Route {
	return a.RootRouter.AllRoutes()
}

// GenerateOpenAPISpec is responsible for taking the PuffApp configuration and turning it into an OpenAPI json.
func (a *PuffApp) GenerateOpenAPISpec() {
	if reflect.ValueOf(a.Config.OpenAPI).IsZero() {
		a.Config.OpenAPI = NewOpenAPI(a)
		paths, tags := a.GeneratePathsTags()
		a.Config.OpenAPI.Tags = tags
		a.Config.OpenAPI.Paths = paths
	}
}

// GeneratePathsTags is a helper function to auto-define OpenAPI tags and paths if you would like to customize OpenAPI schema.
// Returns (paths, tags) to populate the 'Paths' and 'Tags' attribute of OpenAPI
func (a *PuffApp) GeneratePathsTags() (*Paths, *[]Tag) {
	tags := []Tag{}
	tagNames := []string{}
	var paths = make(Paths)
	for _, route := range a.RootRouter.Routes {
		addRoute(route, &tags, &tagNames, &paths)
	}
	for _, router := range a.RootRouter.Routers {
		for _, route := range router.Routes {
			addRoute(route, &tags, &tagNames, &paths)
		}
	}
	return &paths, &tags
}

// GenerateDefinitions is a helper function that takes a list of Paths and generates the OpenAPI schema for each path.
func (a *PuffApp) GenerateDefinitions(paths Paths) map[string]*Schema {
	definitions := map[string]*Schema{}
	for _, p := range paths {
		for _, routeParams := range *p.Parameters {
			definitions[routeParams.Name] = routeParams.Schema
		}
	}
	return definitions
}

// Shutdown calls shutdown on the underlying server with a non-nil empty context.
func (a *PuffApp) Shutdown(ctx context.Context) error {
	return a.Server.Shutdown(ctx)
}

// Close calls close on the underlying server.
func (a *PuffApp) Close() error {
	return a.Server.Close()
}
