package middleware

import (
	"net/http"

	"github.com/nikumar1206/puff"
)

// CSRFMiddlewareConfig is a struct to configure the CSRF middleware.
type CSRFMiddlewareConfig struct {
	// CookieLength specifies the length of the token.
	CookieLength int
	// MaxAge specifies the maximum length for the CSRF cookie.
	MaxAge int
	// ExpectedHeader declares what request header CSRF should be looking for when verifying.
	ExpectedHeader string
	// ProtectedMethods declares what http methods CSRF should secure.
	ProtectedMethods []string
}

// DefaultCSRFMiddleware is a CSRFMiddlewareConfig with specified default values.
var DefaultCSRFMiddleware *CSRFMiddlewareConfig = &CSRFMiddlewareConfig{
	CookieLength:     32,
	MaxAge:           31449600,
	ExpectedHeader:   "X-CSRFMiddlewareToken",
	ProtectedMethods: []string{},
}

// createCSRFMiddleware is used to create a CSRF middleware with a config.
func createCSRFMiddleware(config *CSRFMiddlewareConfig) puff.Middleware {
	cookie_name := "CSRFMiddlewareToken"
	return func(next puff.HandlerFunc) puff.HandlerFunc {
		return func(c *puff.Context) {
			for _, m := range config.ProtectedMethods {
				if c.Request.Method != m {
					continue
				}
				if c.GetCookie(cookie_name) != c.GetHeader(config.ExpectedHeader) {
					c.Forbidden("CSRFMiddlewareToken missing or incorrect.")
					return
				}
				c.SetCookie(&http.Cookie{
					Name:   cookie_name,
					Value:  puff.RandomToken(config.CookieLength),
					MaxAge: config.MaxAge, //expires after hour or session whichever comes first
				})
				next(c)
				break
			}
		}
	}
}

// CSRF middleware automatically injects a cookie with a unique token
// and requires the request to provide the csrf token in the response header.
// If the CSRF Token is not present in the response header, the request is rejected
// with a 403 error.
// The function returns a middleware with the default configuration.
func CSRF() puff.Middleware {
	return createCSRFMiddleware(DefaultCSRFMiddleware)
}

// CSRFWithConfig returns a CSRF middleware with your configuration.
func CSRFWithConfig(config *CSRFMiddlewareConfig) puff.Middleware {
	return createCSRFMiddleware(config)
}