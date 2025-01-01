package puff

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/ThePuffProject/puff/openapi"
)

type PuffApp struct {
	// Config is the underlying application configuration.
	Config *AppConfig
	// Router is the root of the application's routing tree.
	Router *Router

	// Server is the http.Server that will be used to serve requests.
	Server *http.Server
}

// Add a Router to the main app.
// Under the hood attaches the router to the App's RootRouter
func (a *PuffApp) IncludeRouter(r *Router) {
	r.puffapp = a
	a.Router.IncludeRouter(r)
}

// Use registers a middleware function to be used by the root router of the PuffApp.
// The middleware will be appended to the list of middlewares in the root router.
//
// Parameters:
// - m: Middleware function to be added.
func (a *PuffApp) Use(m Middleware) {
	a.Router.Middlewares = append(a.Router.Middlewares, &m)
}

// handleOpenAPI handles all OpenAPI generation and routing.
func (a *PuffApp) handleOpenAPI() error {
	if a.Config.DisableOpenAPIGeneration {
		return nil
	}
	a.generateOpenAPISpec()

	docsRouter := NewRouter(fmt.Sprintf("Swagger Documentation for %s", a.Config.Name), a.Config.DocsURL)

	// Swagger Page
	r := Get(docsRouter, "", func(ctx *Context, _ *NoFields) {
		ctx.SendResponse(HTMLResponse{
			StatusCode: 200,
			Template:   openapi.SwaggerHTML,
			Data:       a.Config.SwaggerUIConfig,
		})
	})
	r.FullPath()
	r.regexp, _ = r.createRegexMatch()

	// OpenAPI JSON
	r = Get(docsRouter, ".json", func(ctx *Context, _ *NoFields) {
		ctx.SendResponse(JSONResponse{
			StatusCode: 200,
			Content:    a.Config.OpenAPI,
		})
	})
	r.FullPath()
	r.regexp, _ = r.createRegexMatch()

	a.IncludeRouter(docsRouter)
	return nil
}

// // addOpenAPIRoutes adds routes to serve OpenAPI documentation for the PuffApp.
// // If a DocsURL is specified, the function sets up two routes:
// // 1. A route to provide the OpenAPI spec as JSON.
// // 2. A route to render the OpenAPI documentation in a user-friendly UI.
// //
// // This method will not add any routes if DocsURL is empty.
// //
// // Errors during spec generation are logged, and the method will exit early if any occur.
// func (a *PuffApp) addOpenAPIRoutes() {
// 	if a.Config.DisableOpenAPIGeneration {
// 		return
// 	}
// 	a.GenerateOpenAPISpec()
// 	docsRouter := &Router{
// 		Prefix: a.Config.DocsURL,
// 		Name:   "OpenAPI Documentation Router",
// 	}

// 	// Provides JSON OpenAPI Schema.
// 	Get(docsRouter, ".json", func(c *Context, _ *NoFields) {
// 		res := JSONResponse{
// 			StatusCode: 200,
// 			Content:    a.Config.OpenAPI,
// 		}
// 		c.SendResponse(res)
// 	})

// 	// Renders OpenAPI schema.
// 	Get(docsRouter, "", func(c *Context, _ *NoFields) {
// 		if a.Config.SwaggerUIConfig == nil {
// 			swaggerConfig := SwaggerUIConfig{
// 				Title:           a.Config.Name,
// 				URL:             a.Config.DocsURL + ".json",
// 				Theme:           "obsidian",
// 				Filter:          true,
// 				RequestDuration: false,
// 				FaviconURL:      "https://fav.farm/💨",
// 			}
// 			a.Config.SwaggerUIConfig = &swaggerConfig
// 		}
// 		res := HTMLResponse{
// 			Template: openAPIHTML, Data: a.Config.SwaggerUIConfig,
// 		}
// 		c.SendResponse(res)
// 	})

// 	a.IncludeRouter(docsRouter)
// }

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
	a.Router.patchRoutes()
	for _, r := range a.Router.Routers {
		r.patchRoutes()
	}
	attachMiddlewares(&[]Middleware{}, a.Router)
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
	a.handleOpenAPI()

	slog.Debug(fmt.Sprintf("Running Puff 💨 on %s", listenAddr))
	slog.Debug(fmt.Sprintf("Visit docs 💨 on %s", fmt.Sprintf("http://localhost%s%s", listenAddr, a.Config.DocsURL)))

	if a.Server == nil {
		a.Server = &http.Server{
			Addr:    listenAddr,
			Handler: a.Router,
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

// AllRoutes returns all routes registered in the PuffApp, including those in sub-routers.
// This function provides an aggregated view of all routes in the application.
func (a *PuffApp) AllRoutes() []*Route {
	return a.Router.AllRoutes()
}

// GenerateOpenAPISpec is responsible for taking the PuffApp configuration and turning it into an OpenAPI json.
func (a *PuffApp) generateOpenAPISpec() {
	if a.Config.OpenAPI == nil { // keep the OpenAPI spec if specified by the user prior
		a.Config.OpenAPI = openapi.NewOpenAPI(a.Config.Name, a.Config.Version)
		paths, tags := a.generatePathsTags()
		a.Config.OpenAPI.Tags = tags
		a.Config.OpenAPI.Paths = paths
	}
}

// GeneratePathsTags is a helper function to auto-define OpenAPI tags and paths if you would like to customize OpenAPI schema.
// Returns (paths, tags) to populate the 'Paths' and 'Tags' attribute of OpenAPI
func (a *PuffApp) generatePathsTags() (*openapi.Paths, *[]openapi.Tag) {
	tags := []openapi.Tag{}
	var paths = make(openapi.Paths)
	for _, route := range a.Router.Routes {
		route.addRouteToPaths(paths)
	}
	for _, router := range a.Router.Routers {
		for _, route := range router.Routes {
			route.addRouteToPaths(paths)
		}
	}
	return &paths, &tags
}

// Shutdown calls shutdown on the underlying server with a non-nil empty context.
func (a *PuffApp) Shutdown(ctx context.Context) error {
	return a.Server.Shutdown(ctx)
}

// Close calls close on the underlying server.
func (a *PuffApp) Close() error {
	return a.Server.Close()
}
