package middleware

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/ThePuffProject/puff"
)

// PanicConfig provides a struct to configure the Panic middleware.
type PanicConfig struct {
	// Skip allows skipping the middleware for specific requests.
	// The function receives the request context and should return true if the middleware should be skipped.
	Skip func(*puff.Context) bool
	// FormatErrorResponse provides a function that recieves the context of the route that resulted in a panic and the error.
	// It should provide a response that can be sent back to the user.
	FormatErrorResponse func(c puff.Context, err any) puff.Response
}

// DefaultCSRFMiddleware is a PanicConfig with specified default values.
var DefaultPanicConfig PanicConfig = PanicConfig{
	FormatErrorResponse: func(c puff.Context, err any) puff.Response {
		errorID := puff.RandomNanoID()
		slog.Error("Panic During Execution", slog.String("ERROR ID", errorID), slog.Any("Error", err))
		errorMsg := fmt.Sprintf("There was a panic during the execution recovered by the panic handling middleware. Error ID: %s", errorID)
		return puff.JSONResponse{StatusCode: http.StatusInternalServerError, Content: map[string]any{"error": errorMsg, "Request-ID": c.GetRequestID()}}
	},
	Skip: DefaultSkipper,
}

// createCSRFMiddleware is used to create a panic middleware with a config.
func createPanicMiddleware(pc PanicConfig) puff.Middleware {
	return func(next puff.HandlerFunc) puff.HandlerFunc {
		return func(c *puff.Context) {
			if pc.Skip != nil && pc.Skip(c) {
				next(c)
				return
			}
			defer func() {
				a := recover()
				if a != nil {
					res := pc.FormatErrorResponse(*c, a)
					c.SendResponse(res)
				}
			}()
			next(c)
		}
	}
}

// Panic middleware returns a middleware with the default configuration.
func Panic() puff.Middleware {
	return createPanicMiddleware(DefaultPanicConfig)
}

// PanicWithConfig returns a middleware with your configuration.
func PanicWithConfig(pc PanicConfig) puff.Middleware {
	return createPanicMiddleware(pc)
}
