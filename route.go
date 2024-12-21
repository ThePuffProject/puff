package puff

import (
	"fmt"
	"maps"
	"reflect"
	"regexp"
	"strings"
)

type Route struct {
	fullPath    string
	regexp      *regexp.Regexp
	params      []Parameter
	Description string
	WebSocket   bool
	Protocol    string
	Path        string
	Handler     func(*Context)
	Fields      any
	// Router points to the router the route belongs to. Will always be the closest router in the tree.
	Router *Router
	// Responses are the schemas associated with a specific route. Have preference over parent router defined routes.
	// Preferably set Responses using the WithResponse/WithResponses method on Route.
	Responses Responses
}

func (r *Route) String() string {
	return fmt.Sprintf("Protocol: %s\nPath: %s\n", r.Protocol, r.Path)
}

func (r *Route) GetFullPath() string {
	return r.fullPath
}

func (route *Route) getCompletePath() {
	var parts []string
	currentRouter := route.Router
	for currentRouter != nil {
		parts = append([]string{currentRouter.Prefix}, parts...)
		currentRouter = currentRouter.parent
	}

	parts = append(parts, route.Path)
	route.fullPath = strings.Join(parts, "")
}

func (route *Route) createRegexMatch() {
	escapedPath := strings.ReplaceAll(route.fullPath, "/", "\\/")
	regexPattern := regexp.MustCompile(`\{[^}]+\}`).ReplaceAllString(escapedPath, "([^/]+)")
	route.regexp = regexp.MustCompile("^" + regexPattern + "$")
}

func (route *Route) handleInputSchema() error { // should this return an error or should it panic?
	if route.Fields == nil {
		route.params = []Parameter{}
		return nil
	}
	sv := reflect.ValueOf(route.Fields) //
	svk := sv.Kind()
	if svk != reflect.Ptr {
		return fmt.Errorf("fields must be POINTER to struct")
	}
	sve := sv.Elem()
	svet := sve.Type()
	if sve.Kind() != reflect.Struct {
		return fmt.Errorf("fields must be pointer to STRUCT")
	}

	newParams := []Parameter{}
	for i := range svet.NumField() {
		newParam := Parameter{}
		svetf := svet.Field(i)

		name := svetf.Tag.Get("name")
		if name == "" {
			name = svetf.Name
		}

		// param.Schema
		newParam.Schema = newDefinition(route, sve.Field(i).Interface())

		//param.In
		specified_kind := svetf.Tag.Get("kind") //ref: Parameters object/In
		if name == "Body" && specified_kind == "" {
			specified_kind = "body"
		}
		if !isValidKind(specified_kind) {
			return fmt.Errorf("specified kind on field %s in struct tag must be header, path, query, cookie, body, or formdata", svetf.Name)
		}

		//param.Description
		description := svetf.Tag.Get("description")

		//param.Required
		specified_required := svetf.Tag.Get("required")
		specified_deprecated := svetf.Tag.Get("deprecated")

		required_def := true
		if specified_kind == "cookie" { // cookies by default should never be required
			required_def = false
		}

		required, err := resolveBool(specified_required, required_def)
		if err != nil {
			return err
		}
		deprecated, err := resolveBool(specified_deprecated, false)
		if err != nil {
			return err
		}

		//param.Schema.format
		format := svetf.Tag.Get("format")
		if format != "" {
			newParam.Schema.Format = format
		}

		newParam.Name = name
		newParam.In = specified_kind
		newParam.Description = description
		newParam.Required = required
		newParam.Deprecated = deprecated

		newParams = append(newParams, newParam)
	}
	route.params = newParams
	return nil
}

// GenerateResponses is responsible for generating the 'responses' attribute in the OpenAPI schema.
// Since responses can be specified at multiple levels, responses at the route level will be given the most specificity.
func (r *Route) GenerateResponses() {

	if r.Router.puff.Config.DocsURL == "" {
		// if swagger documentation is off, we will not set responses
		return
	}

	currentRouter := r.Router

	for currentRouter != nil {
		// avoid over-writing the original responses for the routers
		clonedResponses := maps.Clone(currentRouter.Responses)
		if clonedResponses == nil {
			clonedResponses = make(Responses)
		}
		maps.Copy(clonedResponses, r.Responses)
		currentRouter = currentRouter.parent
	}
}

// WithResponse registers a single response type for a specific HTTP status code
// for the route. This method is used exclusively for generating Swagger documentation,
// allowing users to specify the response type that will be represented in the Swagger
// API documentation when this status code is encountered.
//
// Example usage:
//
//	app.Get("/pizza", func(c puff.Context) {
//	    c.SendResponse(puff.JSONResponse{http.StatusOK, PizzaResponse{Name: "Margherita", Price: 10, Size: "Medium"}})
//	}).WithResponse(http.StatusOK, puff.ResponseType[PizzaResponse])
//
// Parameters:
//   - statusCode: The HTTP status code that this response corresponds to.
//   - ResponseType: The Go type that represents the structure of the response body.
//     This should be the type (not an instance) of the struct that defines the
//     response schema.
//
// Returns:
// - The updated Route object to allow method chaining.
func (r *Route) WithResponse(statusCode int, ResponseTypeFunc func() reflect.Type) *Route {
	r.Responses[statusCode] = ResponseTypeFunc
	return r
}

// WithResponses registers multiple response types for different HTTP status codes
// for the route. This method is used exclusively for generating Swagger documentation,
// allowing users to define various response types based on the possible outcomes
// of the route's execution, as represented in the Swagger API documentation.
//
// Example usage:
//
//	app.Get("/pizza", func(c puff.Context) {
//	    ~ logic here
//	    if found {
//	        c.SendResponse(puff.JSONResponse{http.StatusOK, PizzaResponse{Name: "Margherita", Price: 10, Size: "Medium"}})
//	    } else {
//	        c.SendResponse(puff.JSONResponse{http.StatusNotFound, ErrorResponse{Message: "Not Found"}})
//	    }
//	}).WithResponses(
//	    puff.DefineResponse(http.StatusOK, puff.ResponseType[PizzaResponse]),
//	    puff.DefineResponse(http.StatusNotFound, puff.ResponseType[ErrorResponse]),
//	)
//
// Parameters:
//   - responses: A variadic list of ResponseDefinition objects that define the
//     mapping between HTTP status codes and their corresponding response types.
//     Each ResponseDefinition includes a status code and a type representing the
//     response body structure.
//
// Returns:
// - The updated Route object to allow method chaining.
func (r *Route) WithResponses(responses ...ResponseDefinition) *Route {
	for _, response := range responses {
		r.Responses[response.StatusCode] = response.ResponseType
	}
	return r
}
