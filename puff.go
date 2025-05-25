// Package puff provides primitives for implementing a Puff Server
package puff

import "log/slog"

type (
	HandlerFunc func(*Context)
	Middleware  func(next HandlerFunc) HandlerFunc
)

// ErrorConfig struct is now defined in config.go. This local definition is removed.
// type ErrorConfig struct {
// 	// ErrorKey is the key Puff will use to return the error. UseJSONResponse must be set to true.
// 	ErrorKey string
// 	// UseJSONResponse determines if Puff will use JSON to return error. If false, errors will be returned as 'plain-text'.
// 	UseJSONResponse bool
// }

// Commenting out the old AppConfig as it's being superseded by Config from config.go
/*
type AppConfig struct {
	Name string
	Version string
	DocsURL string
	TLSPublicCertFile string
	TLSPrivateKeyFile string
	OpenAPI *OpenAPI
	SwaggerUIConfig *SwaggerUIConfig
	LoggerConfig *LoggerConfig
	DisableOpenAPIGeneration bool
	ErrorConfig *ErrorConfig // This was already changed to *ErrorConfig
	VisualizeRoutesOnStartup bool
}
*/

// App function now takes *Config.
// PuffApp.Config field type must be changed from *AppConfig to *Config where PuffApp is defined.
func App(appName string, cfg *Config) *PuffApp { // Changed signature
	r := NewRouter(appName) // Router name taken from a new parameter

	if cfg == nil {
		cfg = DefaultConfig()
	}
	if cfg.ErrorConfig == nil {
		// Ensure ErrorConfig is initialized, as DefaultConfig() provides it.
		// This handles cases where a Config literal might be passed with nil ErrorConfig.
		defaultErrCfg := DefaultConfig().ErrorConfig
		if defaultErrCfg != nil {
			cfg.ErrorConfig = defaultErrCfg
		} else { // Fallback, though DefaultConfig should always provide it
			cfg.ErrorConfig = &ErrorConfig{}
		}
	}
	
	// Assuming PuffApp struct has its Config field type updated to *Config
	a := &PuffApp{
		Config:     cfg,
		rootRouter: r,
	}

	// Logger setup using LoggerConfig from the merged Config struct
	if cfg.LoggerConfig == nil {
		cfg.LoggerConfig = &LoggerConfig{} // Provide a default if nil
	}
	l := NewLogger(cfg.LoggerConfig)
	slog.SetDefault(l)

	a.rootRouter.puff = a
	a.rootRouter.Responses = Responses{}
	return a
}

func DefaultApp(name string) *PuffApp {
	// Now uses the new App signature and DefaultConfig() which returns *Config
	cfg := DefaultConfig()
	cfg.Name = name // Set the application name from the parameter

	// Other specific defaults from the old AppConfig if needed:
	// cfg.Version = "0.0.0" // Already set by DefaultConfig()
	// cfg.DocsURL = "/docs"   // Already set by DefaultConfig()
	
	app := App(name, cfg)
	return app
}
