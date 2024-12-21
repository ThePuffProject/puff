package middleware

import (
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/ThePuffProject/puff"
	color "github.com/ThePuffProject/puff/color"
)

// LoggingConfig defines the configuration for the Logging middleware.
type LoggingConfig struct {
	// Skip allows skipping the middleware for specific requests.
	// The function receives the request context and should return true if the middleware should be skipped.
	Skip func(*puff.Context) bool
	// LoggingFunction is a definable function for customizing the log on an http request.
	// Should theoretically call a method deriving from slog.Log
	LoggingFunction func(ctx puff.Context, startTime time.Time)
}

var DefaultLoggingConfig LoggingConfig = LoggingConfig{
	LoggingFunction: func(ctx puff.Context, startTime time.Time) {
		// lc := ctx.LoggerConfig
		// FIXME: can now be based off ctx.LoggerConfig
		processingTime := time.Since(startTime).String()
		sc := ctx.GetStatusCode()
		var statusColor string
		switch {
		case sc >= 500:
			statusColor = color.Colorize(strconv.Itoa(sc), color.FgBrightRed)
		case sc >= 400:
			statusColor = color.Colorize(strconv.Itoa(sc), color.BgBrightYellow)
		case sc >= 300:
			statusColor = color.Colorize(strconv.Itoa(sc), color.FgBrightCyan)
		default:
			statusColor = color.Colorize(strconv.Itoa(sc), color.FgBrightGreen)
		}
		// TODO: make the below configurable
		// Request ID should only be present if present
		slog.Info(
			fmt.Sprintf("%s %s| %s | %s | %s ",
				statusColor,
				fmt.Sprintf("%s %s", ctx.Request.Method, ctx.Request.URL.String()),
				processingTime,
				ctx.GetRequestID(),
				ctx.ClientIP(),
			),
		)
	},
	Skip: DefaultSkipper,
}

func createLoggingMiddleware(lc LoggingConfig) puff.Middleware {
	return func(next puff.HandlerFunc) puff.HandlerFunc {
		return func(ctx *puff.Context) {
			if lc.Skip != nil && lc.Skip(ctx) {
				next(ctx)
				return
			}
			startTime := time.Now()
			next(ctx)
			lc.LoggingFunction(*ctx, startTime)
		}
	}
}

// Logging returns a Logging middleware with the default configuration.
// BUG(Puff): Default Logging Middleware is not context aware and therefore cannot format logs based on the defined logger config.
func Logging() puff.Middleware {
	return createLoggingMiddleware(DefaultLoggingConfig)
}

// LoggingWithConfig returns a Logging middleware with the specified configuration.
func LoggingWithConfig(tc LoggingConfig) puff.Middleware {
	return createLoggingMiddleware(tc)
}
