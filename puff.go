// Package puff provides primitives for implementing a Puff Server
package puff

import (
	"log/slog"
	"reflect"
	"runtime"

	"github.com/ThePuffProject/puff/openapi"
)

const documentationURL = "https://ThePuffProject.github.io/#"

type HandlerFunc func(*Context, any)
type Middleware func(next HandlerFunc) HandlerFunc

// AppConfig defines PuffApp parameters.
type AppConfig struct {
	// Name is the application name
	Name string

	// Version is the application version.
	Version string

	// TLSPublicCertFile specifies the file for the TLS certificate (usually .pem or .crt).
	TLSPublicCertFile string

	// TLSPrivateKeyFile specifies the file for the TLS private key (usually .key).
	TLSPrivateKeyFile string

	// DocsURL specifies the prefix for the Swagger Docs router. Regardless of this value, if
	// DisableOpenAPIGeneration is set to true, the Swagger Docs router will not be created.
	DocsURL string

	// OpenAPI configuration. Gives users access to the OpenAPI spec generated. Can be manipulated by the user.
	OpenAPI *openapi.OpenAPI

	// SwaggerUIConfig is the UI specific configuration.
	SwaggerUIConfig *openapi.SwaggerUIConfig

	// DisableOpenAPIGeneration controls whether an OpenAPI schema will be generated.
	DisableOpenAPIGeneration bool

	// LoggerConfig is the application logger config.
	LoggerConfig LoggerConfig
}

func App(c *AppConfig) *PuffApp {
	r := &Router{Name: "Default", Tag: "Default", Description: "Default Router"}

	a := &PuffApp{
		Config:     c,
		RootRouter: r,
	}
	if a.Config.LoggerConfig == nil {
		a.Config.LoggerConfig = &LoggerConfig{}
	}
	l := NewLogger(a.Config.LoggerConfig)
	slog.SetDefault(l)

	a.Router.puff = a
	a.Router.Responses = Responses{}
	return a
}

func DefaultApp(name string) *PuffApp {
	app := App(&AppConfig{
		Version: "0.0.0",
		Name:    name,
		DocsURL: "/docs",
	})

	return app
}

// registerRoute registers a route on the router and compiles the fields.
func registerRoute(router *Router, method string, path string, fieldsType reflect.Type, handlerFunc func(*Context, any)) *Route {
	_, file, line, ok := runtime.Caller(2)
	description := ""
	if !ok {
		slog.Error("puff.registerRoute: runtime.Caller failed")
	} else {
		description = readDescription(file, line)
	}

	if fieldsType == nilschema { // no fields
		fieldsType = nil
	}

	route := &Route{
		Path:        path,
		Description: description,
		Handler:     handlerFunc,
		Protocol:    method,
		Router:      router,
		Responses:   Responses{},
		fieldsType:  fieldsType,
	}
	router.Routes = append(router.Routes, route)
	return route
}

// Get registers a GET route on the router.
func Get[T any](router *Router, path string, handler func(*Context, *T)) *Route {
	return registerRoute(router, "GET", path, reflect.TypeFor[T](), func(ctx *Context, f any) {
		handler(ctx, f.(*T))
	})
}

// Post registers a POST route on the router.
func Post[T any](router *Router, path string, handler func(*Context, *T)) *Route {
	return registerRoute(router, "POST", path, reflect.TypeFor[T](), func(ctx *Context, f any) {
		handler(ctx, f.(*T))
	})
}

// Put registers a PUT route on the router.
func Put[T any](router *Router, path string, handler func(*Context, *T)) *Route {
	return registerRoute(router, "PUT", path, reflect.TypeFor[T](), func(ctx *Context, f any) {
		handler(ctx, f.(*T))
	})
}

// Patch registers a PATCH route on the router.
func Patch[T any](router *Router, path string, handler func(*Context, *T)) *Route {
	return registerRoute(router, "PATCH", path, reflect.TypeFor[T](), func(ctx *Context, f any) {
		handler(ctx, f.(*T))
	})
}

// Delete registers a DELETE route on the router.
func Delete[T any](router *Router, path string, handler func(*Context, *T)) *Route {
	return registerRoute(router, "DELETE", path, reflect.TypeFor[T](), func(ctx *Context, f any) {
		handler(ctx, f.(*T))
	})
}

// WebSocket registers a WebSocket route on the router. WithResponse() must not be called on the Route returned.
func WebSocket[T any](router *Router, path string, handler func(*Context, *T)) *Route {
	route := registerRoute(router, "GET", path, reflect.TypeFor[T](), func(ctx *Context, f any) {
		handler(ctx, f.(*T))
	})
	route.WebSocket = true
	return route
}
