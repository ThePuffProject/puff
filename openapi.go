package puff

import (
	_ "embed"
	"net/http"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

//go:embed static/openAPI.html
var openAPIHTML string
var Schemas = make(SchemaDefinition)

func parameterToRequestBodyOrReference(p Parameter) RequestBodyOrReference {
	m := make(map[string]MediaType)
	s := p.Schema

	if p.Schema.Ref != "" {
		s = &Schema{Ref: p.Schema.Ref}
	}

	m["application/json"] = MediaType{
		Schema: s, // schema with just ref or entire schema
	}
	requestBody := RequestBodyOrReference{
		Reference:   "",
		Description: p.Description,
		Content:     m,
		Required:    p.Required,
	}
	return requestBody
}

func addRoute(route *Route, tags *[]Tag, tagNames *[]string, paths *Paths) *Paths {
	tag := route.Router.Tag //FIXME: tag on route should not just be tag on router
	if tag == "" {
		tag = route.Router.Name
	}
	if !slices.Contains(*tagNames, tag) {
		*tagNames = append(*tagNames, tag)
		*tags = append(*tags, Tag{Name: tag, Description: ""})
	}
	parameters := make([]Parameter, len(route.params))
	var requestBody RequestBodyOrReference
	for _, p := range route.params {
		if p.In == "body" {
			requestBody = parameterToRequestBodyOrReference(p)
			continue
		}
		if p.In == "file" {
			requestBody = RequestBodyOrReference{
				Content: map[string]MediaType{
					"multipart/form-data": {
						Schema: &Schema{
							Type:     "object",
							Required: []string{p.Name},
							Properties: map[string]*Schema{
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
		np := Parameter{
			Name:        p.Name,
			Description: p.Description,
			Required:    p.Required,
			In:          p.In,
			Deprecated:  p.Deprecated,
		}
		np.Schema = p.Schema
		parameters = append(parameters, np)
	}

	pathMethod := &Operation{
		Summary:     generateSummary(*route),
		OperationID: generateOperationId(*route),
		Tags:        []string{tag},
		Parameters:  parameters, //NOTE: check json struct tag on ParameterOrReference
		RequestBody: &requestBody,
		Responses:   convertRouteResponsestoOpenAPIResponses(*route),
		Description: route.Description,
		Callbacks:   map[string]Callback{},
	}

	pathItem := (*paths)[route.fullPath]
	switch route.Protocol {
	// TODO: handle other protocols
	case http.MethodGet:
		pathItem.Get = pathMethod
		// explicity remove request body for GET requests
		pathItem.Get.RequestBody = nil
	case http.MethodPost:
		pathItem.Post = pathMethod
	case http.MethodPut:
		pathItem.Put = pathMethod
	case http.MethodPatch:
		pathItem.Patch = pathMethod
	case http.MethodDelete:
		pathItem.Delete = pathMethod
	}
	(*paths)[route.fullPath] = pathItem

	return paths
}

func convertRouteResponsestoOpenAPIResponses(route Route) map[string]OpenAPIResponse {
	// FIXME: description can potentially be pulled from a map
	openAPIResponses := map[string]OpenAPIResponse{}
	for statusCode, res := range route.Responses {
		sc := strconv.Itoa(statusCode)
		realRes := reflect.New(res()).Interface()
		schema := newDefinition(&route, realRes)
		openAPIResponses[sc] = OpenAPIResponse{
			Content: map[string]MediaType{
				"application/json": {Schema: schema},
			},
		}
	}
	return openAPIResponses
}

func generateOperationId(r Route) string {
	path := r.fullPath
	re := regexp.MustCompile(`/([a-zA-Z])`)

	normalizedPath := re.ReplaceAllStringFunc(path, func(match string) string {
		// Extract the character after "/"
		char := match[1:]
		// Capitalize the character and return it
		return strings.ToUpper(char)
	})
	return strings.ToLower(r.Protocol) + normalizedPath
}

func generateSummary(r Route) string {
	summary := r.Description
	if len(summary) > 100 {
		summary = summary[:97] + " ..."
	}
	return summary
}
