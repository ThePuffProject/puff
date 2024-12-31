package puff

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"text/template"
)

func ResponseType[T any]() reflect.Type {
	return reflect.TypeOf(new(T)).Elem()
}

type StatusCode int

// Responses type maps together the HTTPStatusCode with a function returning the reflect.Type
type Responses = map[StatusCode]func() reflect.Type

// ResponseDefinition represents a definition of a response for a specific HTTP status code.
// It is used to map an HTTP status code to the corresponding response type for a route.
// Puff uses this to automatically generate Swagger documentation.
//
// Fields:
//   - StatusCode: The HTTP status code associated with this response (e.g., http.StatusOK for success,http.StatusNotFound for not found).
//   - ResponseType: The Go type that defines the response body (e.g., a struct). This type is used to generate
//     the corresponding Swagger schema. The type should not be an instance; just the a function returning reflect.Type (e.g., `puff.ResponseType[Pizza]`).
type ResponseDefinition struct {
	StatusCode   int
	ResponseType func() reflect.Type
}

// DefineResponse creates a ResponseDefinition mapping an HTTP status code
// to the corresponding response type for a route.
//
// Example:
//
//	app.Get("/pizza", handler).WithResponses(
//	    puff.DefineResponse(http.StatusOK, PizzaResponse),
//	    puff.DefineResponse(http.StatusNotFound, ErrorResponse),
//	)
//
// Parameters:
//   - statusCode: The HTTP status code that this response corresponds to.
//   - ResponseType: The Go type that represents the response body. This should be the type (not an instance)
//     of the struct that defines the response schema.
//
// Returns:
// - A ResponseDefinition that maps the provided status code to the response type.
func DefineResponse(statusCode int, ResponseType func() reflect.Type) ResponseDefinition {
	return ResponseDefinition{
		StatusCode:   statusCode,
		ResponseType: ResponseType,
	}
}

// Response is an interface that all response types should implement.
type Response interface {
	GetStatusCode() int
	GetContentType() string
	WriteContent(*Context) error
}

// JSONResponse represents a response with JSON content.
type JSONResponse struct {
	StatusCode int
	Content    any
}

// GetStatusCode returns the status code of the JSON response.
func (j JSONResponse) GetStatusCode() int {
	return resolveStatusCode(j.StatusCode, 200)
}

func (j JSONResponse) GetContentType() string {
	return "application/json"
}

// GetContent returns the content of the JSON response.
func (j JSONResponse) WriteContent(c *Context) error {
	err := json.NewEncoder(c.ResponseWriter).Encode(j.Content)
	if err != nil {
		return fmt.Errorf("writing JSONResponse content failed with: %s", err.Error())
	}
	return nil
}

// HTMLResponse represents a response with HTML content.
// It supports both file-based templates and inline string templates.
type HTMLResponse struct {
	StatusCode int
	// Content to render if TemplateFile is not used.
	Content string
	// TemplateFile is the path to the template file to use.
	TemplateFile string
	// Template is the inline template string. (optional)
	Template string
	// Data to pass to the template.
	Data any
}

// GetStatusCode returns the status code of the HTML response.
func (h HTMLResponse) GetStatusCode() int {
	return resolveStatusCode(h.StatusCode, 200)
}

func (h HTMLResponse) GetContentType() string {
	return "text/html"
}

// WriteContent writes the HTML content to the response.
// It checks whether to use an inline template or a template file.
func (h HTMLResponse) WriteContent(c *Context) error {
	var tmpl *template.Template
	var err error

	if h.TemplateFile != "" { // If TemplateFile is provided, use it.
		tmpl, err = template.ParseFiles(h.TemplateFile)
		if err != nil {
			return fmt.Errorf("parsing template file failed: %s", err.Error())
		}
	} else if h.Template != "" { // If Template string is provided, use it.
		tmpl, err = template.New("inlineTemplate").Parse(h.Template)
		if err != nil {
			return fmt.Errorf("parsing inline template failed: %s", err.Error())
		}
	} else { // If no TemplateFile and Template, render the content as plain HTML.
		fmt.Fprint(c.ResponseWriter, h.Content)
		return nil
	}

	// Execute the template with the provided data.
	err = tmpl.Execute(c.ResponseWriter, h.Data)
	if err != nil {
		return fmt.Errorf("executing template failed: %s", err.Error())
	}
	return nil
}

// FileResponse represents a response that sends a file.
type FileResponse struct {
	StatusCode  int
	FilePath    string
	FileContent []byte
	ContentType string
}

// GetStatusCode returns the status code of the file response.
func (f FileResponse) GetStatusCode() int {
	return resolveStatusCode(f.StatusCode, 0)
}

func (f FileResponse) GetContentType() string {
	return resolveContentType(f.ContentType, contentTypeFromFileName(f.FilePath))
}

// WriteContent serves the file from the provided path.
func (f FileResponse) WriteContent(c *Context) error {
	if _, err := os.Stat(f.FilePath); errors.Is(err, os.ErrNotExist) {
		c.InternalServerError("File %s was not found", f.FilePath)
		return nil
	}

	if c.GetRequestHeader("Range") != "" {
		c.statusCode = 206
	} else {
		c.statusCode = 200
	}
	http.ServeFile(c.ResponseWriter, c.Request, f.FilePath)
	return nil
}

// Handler returns a handler function for serving the file response.
func (f *FileResponse) Handler() func(*Context) {
	return func(c *Context) {
		c.SendResponse(f)
	}
}

// StreamingResponse represents a response that streams content.
type StreamingResponse struct {
	StatusCode int
	// StreamHandler is a function that takes in a pointer to a channel.
	// The channel should be written to with a ServerSideEvent to write
	// to the response. It should be closed once done writing.
	StreamHandler func(*chan ServerSideEvent)
}

type ServerSideEvent struct {
	Event string
	Data  string
	ID    string
	Retry int
}

// GetStatusCode returns the status code of the streaming response.
func (s StreamingResponse) GetStatusCode() int {
	return resolveStatusCode(s.StatusCode, 200)
}

func (s StreamingResponse) GetContentType() string {
	return "text/event-stream"
}

// GetContent returns the content of the streaming response.
func (s StreamingResponse) WriteContent(c *Context) error {
	c.ResponseWriter.Header().Set("Cache-Control", "no-cache")
	c.ResponseWriter.Header().Set("Connection", "keep-alive")

	stream := make(chan ServerSideEvent)
	go func() {
		defer close(stream)
		s.StreamHandler(&stream)
	}()
	for value := range stream {
		fmt.Fprint(c.ResponseWriter, constructSSE(value))
		c.ResponseWriter.(http.Flusher).Flush()
	}
	return nil
}

func constructSSE(eventStruct ServerSideEvent) string {
	finalEvent := ""

	if eventStruct.ID != "" {
		finalEvent += fmt.Sprintf("id: %s\n", eventStruct.ID)
	}

	if eventStruct.Event != "" {
		finalEvent += fmt.Sprintf("event: %s\n", eventStruct.Event)
	}

	if eventStruct.Retry != 0 {
		finalEvent += fmt.Sprintf("retry: %d\n", eventStruct.Retry)
	}

	finalEvent += fmt.Sprintf("data: %s\n\n", eventStruct.Data)

	return finalEvent
}

func (s StreamingResponse) Handler() func(*Context) {
	return func(c *Context) {
		c.SendResponse(s)
	}
}

// RedirectResponse represents a response that sends a redirect to the client.
type RedirectResponse struct {
	// StatusCode provides the 3xx status code of the redirect response. Default: 308.
	StatusCode int
	// To provides the URL to redirect the client to.
	To string
}

// GetStatusCode returns the status code for the redirect response.
// If the status code is not provided, or not valid for a redirect, it will default to 308.
func (r RedirectResponse) GetStatusCode() int {
	if r.StatusCode == 0 || !(r.StatusCode >= 300 && r.StatusCode <= 308) {
		return 308
	}
	return r.StatusCode
}

// GetContentType returns the content type for the redirect response.
// It will always return an empty string since there is no body to
// describe in content typc.ResponseWriter.
func (r RedirectResponse) GetContentType() string {
	return "text/html; charset=utf-8"
}

// WriteContent writes the header Location to redirect the client to.
func (r RedirectResponse) WriteContent(c *Context) error {
	c.SetResponseHeader("Location", r.To)
	fmt.Fprintf(c.ResponseWriter, `<!DOCTYPE HTML>
    <html lang='en-US'>
    <head>
        <meta charset='UTF-8'>
        <meta http-equiv='refresh' content='0; url=%s'>
        <script type='text/javascript'>
            window.location.href = '%s'
        </script>
        <title>Page Redirection</title>
    </head>
    <body>
        If you are not redirected automatically, follow this <a href='%s'>link to example</a>.
    </body>
    </html>`, r.To, r.To, r.To)
	return nil
}

// GenericResponse represents a response with plain text content.
type GenericResponse struct {
	StatusCode  int
	Content     string
	ContentType string
}

// GetStatusCode returns the status code of the generic response.
func (g GenericResponse) GetStatusCode() int {
	return resolveStatusCode(g.StatusCode, 200)
}

func (g GenericResponse) GetContentType() string {
	return resolveContentType(g.ContentType, "text/plain")
}

// GetContent returns the content of the generic response.
func (g GenericResponse) WriteContent(c *Context) error {
	fmt.Fprint(c.ResponseWriter, g.Content)
	return nil
}
