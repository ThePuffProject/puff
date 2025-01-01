// Package middleware provides middlewares for handling common web application requirements.
// This package includes middleware for handling Cross-Origin Resource Sharing (CORS).
package middleware

import (
	"net/http"
	"strings"

	"github.com/ThePuffProject/puff"
)

// CORSConfig defines the configuration for the CORS middleware.
type CORSConfig struct {
	// Skip allows skipping the middleware for specific requests.
	// The function receives the request context and should return true if the middleware should be skipped.
	Skip func(*puff.Context) bool

	// AllowedOrigin specifies the allowed origin for CORS.
	AllowedOrigin string

	// AllowedMethods specifies the allowed HTTP methods for CORS.
	AllowedMethods []string

	// AllowedHeaders specifies the allowed HTTP headers for CORS.
	AllowedHeaders []string
}

// DefaultCORSConfig provides the default configuration for CORS middleware.
var DefaultCORSConfig CORSConfig = CORSConfig{
	AllowedOrigin: "*",
	AllowedMethods: []string{
		http.MethodGet,
		http.MethodHead,
		http.MethodConnect,
		http.MethodOptions,
		http.MethodTrace,
		http.MethodPut,
		http.MethodPatch,
		http.MethodPost,
		http.MethodDelete,
	},
	AllowedHeaders: []string{"Content-Type", "Authorization", "Origin"},
	Skip:           DefaultSkipper,
}

// createCORSMiddleware creates a CORS middleware with the given configuration.
func createCORSMiddleware(c CORSConfig) puff.Middleware {
	allowedMethods := strings.Join(c.AllowedMethods, ",")
	allowedHeaders := strings.Join(c.AllowedHeaders, ",")

	return func(next puff.HandlerFunc) puff.HandlerFunc {
		return func(ctx *puff.Context, f any) {
			if c.Skip != nil && c.Skip(ctx) {
				next(ctx, f)
				return
			}

			ctx.SetResponseHeader("Access-Control-Allow-Origin", c.AllowedOrigin)
			ctx.SetResponseHeader("Access-Control-Allow-Methods", allowedMethods)
			ctx.SetResponseHeader("Access-Control-Allow-Headers", allowedHeaders)
			next(ctx, f)
		}
	}
}

// CORS returns a CORS middleware with the default configuration.
func CORS() puff.Middleware {
	return createCORSMiddleware(DefaultCORSConfig)
}

// CORSWithConfig returns a CORS middleware with the specified configuration.
func CORSWithConfig(c CORSConfig) puff.Middleware {
	return createCORSMiddleware(c)
}
