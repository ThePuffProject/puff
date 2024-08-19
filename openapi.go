package puff

import (
	_ "embed"
	"fmt"
	"net/http"
	"slices"
	"strconv"
)

//go:embed static/openAPI.html
var openAPIHTML string
var Definitions = map[string]*Schema{}

type Reference struct {
	Ref         string `json:"$ref"`
	Summary     string `json:"$summary"`
	Description string `json:"$description"`
}

// OpenAPI struct represents the root of the OpenAPI document.
type OpenAPI struct {
	SpecVersion       string                `json:"openapi"`
	Definitions       map[string]*Schema    `json:"definitions"`
	Info              Info                  `json:"info"`
	JSONSchemaDialect string                `json:"jsonSchemaDialect"`
	Servers           []Server              `json:"servers"`
	Paths             Paths                 `json:"paths"`
	Webhooks          map[string]any        `json:"webhooks"`
	Components        Components            `json:"components"`
	Security          []SecurityRequirement `json:"security"`
	Tags              []Tag                 `json:"tags"`
	ExternalDocs      ExternalDocumentation `json:"externalDocs"`
	// spec holds the OpenAPI json as bytes
	spec *[]byte
}

// // Definitions contains schemas that can be referenced throughout
// // the rest of the document. Reference:
// // https://spec.openapis.org/oas/v3.1.0#schema-object
// type Definition struct {
// 	Type       string   `json:"type"`
// 	Required   []string `json:"required"`
// 	Properties map[string]Property
// }

// Property defines a property in the OpenAPI spec that defines information
// about a property (parameters, definitions, etc).
type Property struct {
	Type    string `json:"type"`
	Format  string `json:"format"`
	Example any    `json:"example"`
}

// Info struct provides metadata about the API.
type Info struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
	// Description is an html string that describes the API service. Do *NOT* include <Doctype> or <html> tags.
	Description    string  `json:"description"`
	TermsOfService string  `json:"termsOfService"`
	Contact        Contact `json:"contact"`
	License        License `json:"license"`
	Version        string  `json:"version"`
}

// Contact struct contains contact information for the API.
type Contact struct {
	Name  string `json:"name"`
	URL   string `json:"url"`
	Email string `json:"email"`
}

// License struct contains license information for the API.
type License struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// Server struct represents a server object in OpenAPI.
type Server struct {
	URL         string                    `json:"url"`
	Description string                    `json:"description"`
	Variables   map[string]ServerVariable `json:"variables"`
}

// Components struct holds reusable objects for different aspects of the OAS.
type Components struct {
	Schemas         map[string]any `json:"schemas"`
	Responses       map[string]any `json:"responses"`
	Parameters      map[string]any `json:"parameters"`
	Examples        map[string]any `json:"examples"`
	RequestBodies   map[string]any `json:"requestBodies"`
	Headers         map[string]any `json:"headers"`
	SecuritySchemes map[string]any `json:"securitySchemes"`
	Links           map[string]any `json:"links"`
	Callbacks       map[string]any `json:"callbacks"`
	PathItems       map[string]any `json:"pathItems"`
}

// Tag struct represents a tag used by the OpenAPI document.
type Tag struct {
	Name         string                `json:"name"`
	Description  string                `json:"description"`
	ExternalDocs ExternalDocumentation `json:"externalDocs"`
}

// ExternalDocumentation struct provides external documentation for the API.
type ExternalDocumentation struct {
	Description string `json:"description"`
	URL         string `json:"url"`
}

type Paths map[string]PathItem

// PathItem struct describes operations available on a single path.
type PathItem struct {
	Ref         string      `json:"$ref"`
	Summary     string      `json:"summary"`
	Description string      `json:"description"`
	Get         *Operation  `json:"get,omitempty"`
	Put         *Operation  `json:"put,omitempty"`
	Post        *Operation  `json:"post,omitempty"`
	Delete      *Operation  `json:"delete,omitempty"`
	Options     *Operation  `json:"options,omitempty"`
	Head        *Operation  `json:"head,omitempty"`
	Patch       *Operation  `json:"patch,omitempty"`
	Trace       *Operation  `json:"trace,omitempty"`
	Servers     []Server    `json:"servers"`
	Parameters  []Parameter `json:"parameters"`
}

type SecurityRequirement map[string][]string

// Operation struct describes an operation in a PathItem.
type Operation struct {
	Tags         []string                   `json:"tags"`
	Summary      string                     `json:"summary"`
	Description  string                     `json:"description"`
	ExternalDocs ExternalDocumentation      `json:"externalDocs"`
	OperationID  string                     `json:"operationId"`
	Parameters   []Parameter                `json:"parameters"`
	RequestBody  RequestBodyOrReference     `json:"requestBody,omitempty"`
	Responses    map[string]OpenAPIResponse `json:"responses"`
	Callbacks    map[string]Callback        `json:"callbacks"`
	Deprecated   bool                       `json:"deprecated"`
	Security     []SecurityRequirement      `json:"security"`
	Servers      []Server                   `json:"servers"`
}

// Parameter struct describes a parameter in OpenAPI.
type Parameter struct {
	Name            string `json:"name"`
	In              string `json:"in"`
	Description     string `json:"description"`
	Required        bool   `json:"required"`
	Type            string `json:"type"`
	Deprecated      bool   `json:"deprecated"`
	AllowEmptyValue bool   `json:"allowEmptyValue"`
	Style           string `json:"style"`
	Explode         bool   `json:"explode"`
	AllowReserved   bool   `json:"allowReserved"`
	Schema          Schema `json:"schema"`
}

// RequestBodyOrReference is a union type representing either a Request Body Object or a Reference Object.
type RequestBodyOrReference struct {
	Reference   string               `json:"$ref,omitempty"`
	Description string               `json:"description,omitempty"`
	Content     map[string]MediaType `json:"content,omitempty"`
	Required    bool                 `json:"required,omitempty"`
}

// MediaType struct describes a media type object in OpenAPI.
type MediaType struct {
	Schema   Schema         `json:"schema"`
	Example  any            `json:"example"`
	Examples map[string]any `json:"examples"`
}

// Schema struct represents a schema object in OpenAPI.
type Schema struct {
	// Define your schema fields based on your specific requirements
	// Example fields could include type, format, properties, etc.
	// This can be expanded based on the needs of your application.
	Type                 string             `json:"type,omitempty"`
	Format               string             `json:"format,omitempty"`
	Minimum              string             `json:"minimum,omitempty"`
	Items                *Schema            `json:"items,omitempty"`
	Ref                  string             `json:"$ref,omitempty"`
	Properties           map[string]*Schema `json:"properties,omitempty"`
	AdditionalProperties *Schema            `json:"additionalProperties,omitempty"`
	Required             []string           `json:"required"`
	Example              any                `json:"example,omitempty"`
}

// OpenAPIResponse struct describes possible responses in OpenAPI.
type OpenAPIResponse struct {
	Description string               `json:"description"`
	Headers     map[string]Header    `json:"headers,omitempty"`
	Content     map[string]MediaType `json:"content,omitempty"`
	Links       map[string]Link      `json:"links,omitempty"`
}

type Callback map[string]PathItem

type Example struct {
	Summary       string `json:"summary,omitempty"`
	Description   string `json:"description,omitempty"`
	Value         any    `json:"value,omitempty"`
	ExternalValue string `json:"externalValue,omitempty"`
}

type Header struct {
	// Add header fields as needed
}

type Link struct {
	OperationRef string `json:"operationRef,omitempty"`
	OperationID  string `json:"operationId,omitempty"`
	Parameters   any    `json:"parameters,omitempty"`
	RequestBody  any    `json:"requestBody,omitempty"`
	Description  string `json:"description,omitempty"`
	Server       Server `json:"server,omitempty"`
}

type Encoding struct {
	ContentType   string            `json:"contentType,omitempty"`
	Headers       map[string]Header `json:"headers,omitempty"`
	Style         string            `json:"style,omitempty"`
	Explode       bool              `json:"explode,omitempty"`
	AllowReserved bool              `json:"allowReserved,omitempty"`
}

type ServerVariable struct {
	Enum        []string `json:"enum,omitempty"`
	Default     string   `json:"default"`
	Description string   `json:"description,omitempty"`
}

func GenerateOpenAPIUI(title, docsURL string) string {
	return fmt.Sprintf(openAPIHTML, title, docsURL)
}

func parameterToRequestBodyOrReference(p Parameter) RequestBodyOrReference {
	m := make(map[string]MediaType)
	// FIXME: support examples
	s := p.Schema
	if p.Schema.Ref != "" {
		s = Schema{Ref: p.Schema.Ref}
	}
	m["application/json"] = MediaType{
		Schema: s,
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

	description := route.Description
	summary := description
	if len(summary) > 100 {
		summary = summary[:97] + " ..."
	}
	parameters := []Parameter{}
	requestBody := RequestBodyOrReference{}
	for _, p := range route.params {
		if p.In == "body" {
			requestBody = parameterToRequestBodyOrReference(p)
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
		Summary:     summary,
		OperationID: "", //FIXME: needs operation id
		Tags:        []string{tag},
		Parameters:  parameters, //NOTE: check json struct tag on ParameterOrReference
		RequestBody: requestBody,
		Responses:   convertRouteResponsestoOpenAPIResponses(*route),
		Description: description, // TODO: needs to be dynamic on route
	}

	pathItem := (*paths)[route.fullPath]
	switch route.Protocol {
	// TODO: handle other protocols
	case http.MethodGet:
		pathItem.Get = pathMethod
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
	// FIXME: allow specifying examples, or potentially self-generate them
	// FIXME: description can potentially be pulled from a map
	openAPIResponses := map[string]OpenAPIResponse{}
	for statusCode, res := range route.Responses {
		sc := strconv.Itoa(statusCode)
		openAPIResponses[sc] = OpenAPIResponse{
			Description: "",
			Content: map[string]MediaType{
				res.GetContentType(): {Schema: newDefinition(&route, res.GetContent())},
			},
		}
	}
	return openAPIResponses
}
