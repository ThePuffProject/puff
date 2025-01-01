package puff

import (
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/tiredkangaroo/websocket"
)

// Context provides functionality for the route.
type Context struct {
	// Request is the underlying *http.Request object.
	Request *http.Request
	// ResponseWriter is the underlying http.ResponseWriter object.
	ResponseWriter http.ResponseWriter
	// Registry is a map allowing for communication between anything that
	// can access context (including middlewares and the route handler function).
	registry map[string]any
	// WebSocket represents WebSocket connection and its related context, connection, and events.
	// WebSocket will be nil if the route does not use websockets.
	WebSocket *websocket.Conn

	// LoggerConfig is the configuration of the logger in Puff.
	LoggerConfig LoggerConfig

	// body is a stored slice of the body.
	body []byte

	statusCode int
}

func NewContext(w http.ResponseWriter, r *http.Request, a *PuffApp) *Context {
	return &Context{
		Request:        r,
		ResponseWriter: w,
		registry:       make(map[string]any), // prevents assignment to nil map
		LoggerConfig:   *a.Config.LoggerConfig,
	}
}

func (ctx *Context) isWebSocket() bool {
	return ctx.GetRequestHeader("Upgrade") == "websocket" &&
		ctx.GetRequestHeader("Connection") == "Upgrade" &&
		ctx.GetRequestHeader("Sec-WebSocket-Version") == "13"
}

// GetBody retrieves the body from the request. This function may be called
// multiple times.
func (ctx *Context) GetBody() ([]byte, error) {
	if ctx.body == nil {
		b, err := io.ReadAll(ctx.Request.Body)
		if err != nil {
			return nil, err
		}
		ctx.body = b
	}
	return ctx.body, nil
}

// Get gets a value from Context with the key passed in.
// It returns nil if the value is not found.
func (ctx *Context) Get(key string) any {
	return ctx.registry[key]
}

// Set sets a value to Context with a key.
func (ctx *Context) Set(key string, value any) {
	ctx.registry[key] = value
}

// GetRequestHeader gets the value of a request header with key k.
// It returns an empty string if not found.
func (ctx *Context) GetRequestHeader(k string) string {
	return ctx.Request.Header.Get(k)
}

// GetResponseHeader gets the value of a response header with key k.
// It returns an empty string if not found.
func (ctx *Context) GetResponseHeader(k string) string {
	return ctx.ResponseWriter.Header().Get(k)
}

// SetResponseHeader sets the value of the response header k to v.
func (ctx *Context) SetResponseHeader(k, v string) {
	ctx.ResponseWriter.Header().Set(k, v)
}

// GetQueryParam retrives the value of a query param from k.
// If not found, it will return an empty string.
func (ctx *Context) GetQueryParam(k string) string {
	return ctx.Request.URL.Query().Get(k)
}

// GetFormValue retrives the value of a form key named k.
// If not found, it will return an empty string.
func (ctx *Context) GetFormValue(k string) string {
	return ctx.Request.FormValue(k)
}

// GetCookie retrives a cookie from the context with key "k".
// If not found, it will return an empty string.
func (ctx *Context) GetCookie(k string) string {
	cookie, err := ctx.Request.Cookie(k)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// SetCookie writes a new cookie to the request with key "k" and
// value "v". Invalid cookies will be silently dropped. Invalid
// characters will also be silently dropped. Ex. SetCookie with value
// ""HELLO WORLD"". The quotation marks are invalid characters,
// therefore the final cookie will be "HELLO WORLD" instead.
func (ctx *Context) SetCookie(cookie *http.Cookie) {
	http.SetCookie(ctx.ResponseWriter, cookie)
}

// SetContentType sets the content type of the response.
func (ctx *Context) SetContentType(v string) {
	ctx.SetResponseHeader("Content-Type", v)
}

// SetStatusCode sets the status code of the response.
func (ctx *Context) SetStatusCode(sc int) {
	ctx.ResponseWriter.WriteHeader(sc)
	ctx.statusCode = sc
}

// GetStatusCode returns the status code. If response not written, returns default 0.
func (ctx *Context) GetStatusCode() int {
	return ctx.statusCode
}

// GetFormFile returns the multipart file and the multipart file header associated with the key.
// It will only provide the first file associated with that form key. It may return an error that
// is not nil.
func (ctx *Context) GetFormFile(key string) (multipart.File, *multipart.FileHeader, error) {
	return ctx.Request.FormFile(key)
}

// below are methods that are more utility focused.

// GetRequestID gets the X-Request-ID if set (empty string if not set).
// puff/middleware provides a tracing middleware the sets X-Request-ID.
func (ctx *Context) GetRequestID() string {
	return ctx.GetResponseHeader("X-Request-ID")
}

// SendResponse sends res back to the client.
// Any errors at this point will be logged and sending a response will fail.
func (c *Context) SendResponse(res Response) {
	if c.WebSocket != nil {
		slog.Error("calls to SendResponse on routes using websockets is not permitted.")
		return
	}

	c.SetContentType(res.GetContentType())

	if res.GetStatusCode() != 0 { // don't write statusCode for certain content types
		c.SetStatusCode(res.GetStatusCode())
	}

	err := res.WriteContent(c)
	if err != nil {
		msg := fmt.Sprintf(
			"An unexpected error occured while writing content with context: %s.",
			err.Error(),
		)
		slog.Error(msg)
		fmt.Fprint(c.ResponseWriter, "An unknown error occured.")
	}
}

func (ctx *Context) ClientIP() (IPAddress string) {
	return ctx.Request.RemoteAddr
}

// GetBearerToken gets the Bearer token if it exists.
// This will work if the request contains an Authorization header
// that has this syntax: Bearer this_token_here.
func (ctx *Context) GetBearerToken() string {
	bt := ctx.GetRequestHeader("Authorization")

	token_arr := strings.Split(bt, "Bearer ")

	if len(token_arr) > 1 {
		return token_arr[1]
	}

	return ""
}

// below are methods that are more error message focused.

func (ctx *Context) response(status_code int, message string, a ...any) {
	ctx.SendResponse(JSONResponse{
		StatusCode: status_code,
		Content: map[string]any{
			"error": fmt.Sprintf(message, a...),
		},
	})
}

// BadRequest returns a json response with status code 400
// a key error and a value of the formatted string from
// message and the arguments following.
func (ctx *Context) BadRequest(message string, a ...any) {
	ctx.response(400, message, a...)
}

// Forbidden returns a json response with status code 403
// a key error and a value of the formatted string from
// message and the arguments following.
func (ctx *Context) Forbidden(message string, a ...any) {
	ctx.response(403, message, a...)
}

// NotFound returns a json response with status code 404
// with a key error and a value of the formatted string from
// message and the arguments following.
func (ctx *Context) NotFound(message string, a ...any) {
	ctx.response(404, message, a...)
}

// InternalServerError returns a json response with status code 500
// with a key error and a value of the formatted string from
// message and the arguments following.
func (ctx *Context) InternalServerError(message string, a ...any) {
	ctx.response(500, message, a...)
}
