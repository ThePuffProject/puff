package puff

// Config defines the configuration for a PuffApp instance.
type Config struct {
	// Name is the application name
	Name string
	// Version is the application version.
	Version string
	// DocsURL is the Router prefix for Swagger documentation.
	DocsURL string
	// TLSPublicCertFile specifies the file for the TLS certificate (usually .pem or .crt).
	TLSPublicCertFile string
	// TLSPrivateKeyFile specifies the file for the TLS private key (usually .key).
	TLSPrivateKeyFile string
	// OpenAPI configuration. Gives users access to the OpenAPI spec generated. Can be manipulated by the user.
	OpenAPI *OpenAPI // Defined in openapi.go
	// SwaggerUIConfig is the UI specific configuration.
	SwaggerUIConfig *SwaggerUIConfig // Defined in openapi_ui.go
	// LoggerConfig is the application logger config.
	LoggerConfig *LoggerConfig // Defined in logger.go
	// DisableOpenAPIGeneration controls whether an OpenAPI schema will be generated.
	DisableOpenAPIGeneration bool
	// ErrorConfig defines how errors are handled and responded to.
	ErrorConfig *ErrorConfig
	// VisualizeRoutesOnStartup controls whether Puff will display the radix trie router on Startup or not.
	VisualizeRoutesOnStartup bool
	// Add other configurations here, e.g., MaxBodySize, etc.
}

// ErrorConfig defines settings for error responses.
type ErrorConfig struct {
	// UseJSONResponse, when true, makes error handlers (like BadRequest, NotFound)
	// respond with a JSON body. Otherwise, a plain text response is sent.
	UseJSONResponse bool
	// ErrorKey is the key used in JSON error responses (e.g., "error" or "message").
	ErrorKey string
}

// DefaultConfig returns a default configuration for PuffApp.
func DefaultConfig() *Config {
	return &Config{
		Name:                     "Puff App",
		Version:                  "0.0.1",
		DocsURL:                  "/docs",
		TLSPublicCertFile:        "",
		TLSPrivateKeyFile:        "",
		OpenAPI:                  nil, // Will be initialized by PuffApp if not provided
		SwaggerUIConfig:          nil, // Will be initialized by PuffApp if not provided
		LoggerConfig:             &LoggerConfig{}, // Basic default logger config
		DisableOpenAPIGeneration: false,
		ErrorConfig: &ErrorConfig{
			UseJSONResponse: false, // Default to plain text errors
			ErrorKey:        "error", // Default key for JSON errors
		},
		VisualizeRoutesOnStartup: false,
	}
}
