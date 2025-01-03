// Package puff provides primitives for implementing a Puff Server
package puff

import "log/slog"

type (
	HandlerFunc func(*Context)
	Middleware  func(next HandlerFunc) HandlerFunc
)

// ErrorConfig determines how Puff auto-returns errors in case of request-schema validation errors, among other things.
type ErrorConfig struct {
	// ErrorKey is the key Puff will use to return the error. UseJSONResponse must be set to true.
	ErrorKey string
	// UseJSONResponse determines if Puff will use JSON to return error. If false, errors will be returned as 'plain-text'.
	UseJSONResponse bool
}

// AppConfig defines PuffApp parameters.
type AppConfig struct {
	// Name is the application name
	Name string
	// Version is the application version.
	Version string
	// Path is the RootRouter prefix under which all routes will be hosted. e.g "/api"
	Path string
	// DocsURL is the Router prefix for Swagger documentation.
	DocsURL string
	// TLSPublicCertFile specifies the file for the TLS certificate (usually .pem or .crt).
	TLSPublicCertFile string
	// TLSPrivateKeyFile specifies the file for the TLS private key (usually .key).
	TLSPrivateKeyFile string
	// OpenAPI configuration. Gives users access to the OpenAPI spec generated. Can be manipulated by the user.
	OpenAPI *OpenAPI
	// SwaggerUIConfig is the UI specific configuration.
	SwaggerUIConfig *SwaggerUIConfig
	// LoggerConfig is the application logger config.
	LoggerConfig *LoggerConfig
	// DisableOpenAPIGeneration controls whether an OpenAPI schema will be generated.
	DisableOpenAPIGeneration bool
	// ErrorConfig determines how Puff auto-returns errors.
	ErrorConfig ErrorConfig
}

func App(c *AppConfig) *PuffApp {
	r := &Router{
		Name:        "Default",
		Tag:         "Default",
		Description: "Default Router",
		Path:        c.Path,
		rootNode:    insertNode(c.Path),
	}

	a := &PuffApp{
		Config:     c,
		RootRouter: r,
	}
	if a.Config.LoggerConfig == nil {
		a.Config.LoggerConfig = &LoggerConfig{}
	}
	l := NewLogger(a.Config.LoggerConfig)
	slog.SetDefault(l)

	a.RootRouter.puff = a
	a.RootRouter.Responses = Responses{}
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
