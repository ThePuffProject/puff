package puff

import (
	"encoding/hex"
	"fmt"
	"maps"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/ThePuffProject/puff/openapi"
)

type Route struct {
	fullPath    string
	regexp      *regexp.Regexp
	params      []openapi.Parameter
	Description string
	WebSocket   bool
	Protocol    string
	Path        string
	Handler     func(*Context, any)
	fieldsType  reflect.Type
	// Router points to the router the route belongs to. Will always be the closest router in the tree.
	Router *Router
	// Responses are the schemas associated with a specific route. Have preference over parent router defined routes.
	// Preferably set Responses using the WithResponse/WithResponses method on Route.
	Responses Responses
}

func (r *Route) String() string {
	return fmt.Sprintf("Protocol: %s\nPath: %s\n", r.Protocol, r.Path)
}

// FullPath returns the full path of the route with all parent prefixes. If
// the full path has not been created yet, it will be created.
func (r *Route) FullPath() string {
	if r.fullPath != "" {
		return r.fullPath
	}
	r.fullPath = r.generateCompletePath()
	return r.fullPath
}

// getCompletePath generates a full path by appending prefixes.
func (route *Route) generateCompletePath() string {
	var parts []string

	router := route.Router

	for router != nil {
		parts = append([]string{router.Prefix}, parts...) // append parent prefix to the start
		router = router.parent                            // keep climbing up the tree
	}

	parts = append(parts, route.Path) // add all the parts into the slice
	return strings.Join(parts, "")
}

// createRegexMatch creates the regular expression for matches.
func (route *Route) createRegexMatch() (*regexp.Regexp, error) {
	// /api/route -> \/api\/route (to escape regexp)
	escapedPath := strings.ReplaceAll(route.fullPath, "/", "\\/")

	regexpattern, err := regexp.Compile(`\{[^}]+\}`)
	if err != nil {
		return nil, err
	}
	pattern := regexpattern.ReplaceAllString(escapedPath, "([^/]+)")

	matchregex, err := regexp.Compile("^" + pattern + "$")
	if err != nil {
		return nil, err
	}

	return matchregex, nil
}

// operationID returns a unique operationID for the OpenAPI operation. The value of the
// string returned will always be the same, as it is created using the HTTP method and full
// path of the route.
func (r *Route) operationID() string {
	return hex.EncodeToString([]byte(r.Protocol + r.fullPath))
}

func (r *Route) openAPIResponses() (map[string]openapi.OpenAPIResponse, error) {
	openAPIResponses := map[string]openapi.OpenAPIResponse{}
	for statusCode, res := range r.Responses {
		sc := strconv.Itoa(int(statusCode))
		schema, err := newSchemaDefinition(res())
		if err != nil {
			return nil, fmt.Errorf("getting schema definition for response with status code %d encountered an error: %v", statusCode, err)
		}
		openAPIResponses[sc] = openapi.OpenAPIResponse{
			Content: map[string]openapi.MediaType{
				"application/json": {Schema: schema},
			},
		}
	}
	return openAPIResponses, nil
}

// FIXME: tags shouldn't have to rely on the router's tag
func (r *Route) addRouteToPaths(paths openapi.Paths) error {
	var err error

	op := new(openapi.Operation)
	// op.Summary
	if len(r.Description) > 100 {
		op.Summary = r.Description[:98] + "..."
	} else {
		op.Summary = r.Description
	}

	// operationID, tags, description, callbacks
	op.OperationID = r.operationID()
	op.Tags = []string{r.Router.Tag}
	op.Description = r.Description
	op.Callbacks = make(map[string]openapi.Callback)

	// responses
	op.Responses, err = r.openAPIResponses()
	if err != nil {
		return fmt.Errorf("error generating paths from route: %v", err)
	}

	parameters := []openapi.Parameter{}
	var requestbody openapi.RequestBodyOrReference
	for _, p := range r.params {
		if p.In == "body" {
			requestbody = openapi.ParameterAsRequestBody(p)
			continue
		}
		if p.In == "file" {
			requestbody = openapi.RequestBodyOrReference{
				Content: map[string]openapi.MediaType{
					"multipart/form-data": {
						Schema: &openapi.Schema{
							Type:     "object",
							Required: []string{p.Name},
							Properties: map[string]*openapi.Schema{
								p.Name: {
									Type:   "string",
									Format: "binary",
								},
							},
						},
					},
				},
			}
			continue
		}
		// not body or file parameter
		np := openapi.Parameter{

			Name:        p.Name,
			Description: p.Description,
			Required:    p.Required,
			In:          p.In,
			Deprecated:  p.Deprecated,
		}
		np.Schema = p.Schema
		parameters = append(parameters, np)
	}

	op.Parameters = parameters
	op.RequestBody = &requestbody

	path := paths[r.fullPath]
	switch r.Protocol {
	// TODO: handle other protocols
	case http.MethodGet:
		path.Get = op
		path.Get.RequestBody = nil
	case http.MethodPost:
		path.Post = op
	case http.MethodPut:
		path.Put = op
	case http.MethodPatch:
		path.Patch = op
	case http.MethodDelete:
		path.Delete = op
	}
	paths[r.fullPath] = path
	return nil
}

// GenerateResponses is responsible for generating the 'responses' attribute in the OpenAPI schema.
// Since responses can be specified at multiple levels, responses at the route level will be given
// the most specificity.
func (r *Route) generateResponses() {
	if r.Router.puffapp.Config.DocsURL == "" {
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
func (r *Route) WithResponse(statusCode StatusCode, ResponseTypeFunc func() reflect.Type) *Route {
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
