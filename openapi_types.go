package puff

// OpenAPI struct represents the root of the OpenAPI document.
type OpenAPI struct {
	SpecVersion       string                 `json:"openapi"`
	Info              *Info                  `json:"info"`
	JSONSchemaDialect string                 `json:"jsonSchemaDialect"`
	Servers           *[]Server              `json:"servers"`
	Paths             *Paths                 `json:"paths"`
	Webhooks          map[string]any         `json:"webhooks"`
	Components        *Components            `json:"components"`
	Security          *[]SecurityRequirement `json:"security"`
	Tags              *[]Tag                 `json:"tags"`
	ExternalDocs      *ExternalDocumentation `json:"externalDocs"`
	// schemas holds the openAPI schemas generated
	schemas *SchemaDefinition
}

func NewOpenAPI(a *PuffApp) *OpenAPI {
	o := &OpenAPI{
		SpecVersion: "3.1.0",
		Info: &Info{
			Title:   a.Config.Name,
			Version: a.Config.Version,
			Contact: &Contact{},
			License: &License{},
		},
		Servers:      &[]Server{},
		Paths:        new(Paths),
		Components:   NewComponents(a),
		Webhooks:     make(map[string]any),
		Security:     &[]SecurityRequirement{},
		Tags:         &[]Tag{},
		ExternalDocs: &ExternalDocumentation{},
		schemas:      &SchemaDefinition{},
	}
	return o
}

type Reference struct {
	Ref         string `json:"$ref"`
	Summary     string `json:"$summary"`
	Description string `json:"$description"`
}

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
	Description    string   `json:"description"`
	TermsOfService string   `json:"termsOfService"`
	Contact        *Contact `json:"contact"`
	License        *License `json:"license"`
	Version        string   `json:"version"`
}

// Contact struct contains contact information for the API.
type Contact struct {
	Name  string `json:"name,omitempty"`
	URL   string `json:"url,omitempty"`
	Email string `json:"email,omitempty"`
}

// License struct contains license information for the API.
type License struct {
	Name       string `json:"name"`
	Identifier string `json:"identifier,omitempty"`
	Url        string `json:"url,omitempty"`
}

// Server struct represents a server object in OpenAPI.
type Server struct {
	URL         string                    `json:"url"`
	Description string                    `json:"description"`
	Variables   map[string]ServerVariable `json:"variables"`
}

// Components struct holds reusable objects for different aspects of the OAS.
type Components struct {
	Schemas         *SchemaDefinition `json:"schemas,omitempty"`
	Responses       map[string]any    `json:"responses,omitempty"`
	Parameters      map[string]any    `json:"parameters,omitempty"`
	Examples        map[string]any    `json:"examples,omitempty"`
	RequestBodies   map[string]any    `json:"requestBodies,omitempty"`
	Headers         map[string]any    `json:"headers,omitempty"`
	SecuritySchemes map[string]any    `json:"securitySchemes,omitempty"`
	Links           map[string]any    `json:"links,omitempty"`
	Callbacks       map[string]any    `json:"callbacks,omitempty"`
	PathItems       map[string]any    `json:"pathItems,omitempty"`
}

func NewComponents(a *PuffApp) *Components {
	return &Components{
		Schemas:         &Schemas,
		Responses:       make(map[string]any),
		Parameters:      make(map[string]any),
		Examples:        make(map[string]any),
		RequestBodies:   make(map[string]any),
		SecuritySchemes: make(map[string]any),
		Headers:         make(map[string]any),
		Callbacks:       make(map[string]any),
		PathItems:       make(map[string]any),
		Links:           make(map[string]any),
	}
}

// Tag struct represents a tag used by the OpenAPI document.
type Tag struct {
	Name         string                `json:"name"`
	Description  string                `json:"description"`
	ExternalDocs ExternalDocumentation `json:"externalDocs,omitempty"`
}

// ExternalDocumentation struct provides external documentation for the API.
type ExternalDocumentation struct {
	Description string `json:"description"`
	URL         string `json:"url"`
}

// aliases

// PathItem struct describes operations available on a single path.
type PathItem struct {
	Ref         string       `json:"$ref"`
	Summary     string       `json:"summary"`
	Description string       `json:"description"`
	Get         *Operation   `json:"get,omitempty"`
	Put         *Operation   `json:"put,omitempty"`
	Post        *Operation   `json:"post,omitempty"`
	Delete      *Operation   `json:"delete,omitempty"`
	Options     *Operation   `json:"options,omitempty"`
	Head        *Operation   `json:"head,omitempty"`
	Patch       *Operation   `json:"patch,omitempty"`
	Trace       *Operation   `json:"trace,omitempty"`
	Servers     *[]Server    `json:"servers,omitempty"`
	Parameters  *[]Parameter `json:"parameters,omitempty"`
}

// Operation struct describes an operation in a PathItem.
type Operation struct {
	Tags         []string                   `json:"tags"`
	Summary      string                     `json:"summary"`
	Description  string                     `json:"description"`
	ExternalDocs ExternalDocumentation      `json:"externalDocs"`
	OperationID  string                     `json:"operationId"`
	Parameters   []Parameter                `json:"parameters"`
	RequestBody  *RequestBodyOrReference    `json:"requestBody,omitempty"`
	Responses    map[string]OpenAPIResponse `json:"responses"`
	Callbacks    map[string]Callback        `json:"callbacks"`
	Deprecated   bool                       `json:"deprecated"`
	Security     *[]SecurityRequirement     `json:"security,omitempty"`
	Servers      *[]Server                  `json:"servers,omitempty"`
}

// Parameter struct describes a parameter in OpenAPI.
type Parameter struct {
	Name            string  `json:"name"`
	In              string  `json:"in"`
	Description     string  `json:"description"`
	Required        bool    `json:"required"`
	Type            string  `json:"type"`
	Deprecated      bool    `json:"deprecated"`
	AllowEmptyValue bool    `json:"allowEmptyValue"`
	Style           string  `json:"style"`
	Explode         bool    `json:"explode"`
	AllowReserved   bool    `json:"allowReserved"`
	Schema          *Schema `json:"schema"`
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
	Schema   *Schema        `json:"schema"`
	Example  any            `json:"example,omitempty"`
	Examples map[string]any `json:"examples,omitempty"`
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
	Required             []string           `json:"required,omitempty"`
	Examples             []any              `json:"examples,omitempty"`
}

// OpenAPIResponse struct describes possible responses in OpenAPI.
type OpenAPIResponse struct {
	Description string               `json:"description"`
	Headers     map[string]Header    `json:"headers,omitempty"`
	Content     map[string]MediaType `json:"content,omitempty"`
	Links       map[string]Link      `json:"links,omitempty"`
}

type Example struct {
	Summary       string `json:"summary,omitempty"`
	Description   string `json:"description,omitempty"`
	Value         any    `json:"value,omitempty"`
	ExternalValue string `json:"externalValue,omitempty"`
}

type Header struct {
	// TODO: Add header fields as needed
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

// SwaggerUIConfig is the subset of the SwaggerUI configurables that Puff supports.
// To learn more, please read [SwaggerDocs](https://swagger.io/docs/open-source-tools/swagger-ui/usage/configuration/).
type SwaggerUIConfig struct {
	// Title of the Swagger Page
	Title string
	// URL of OpenAPI JSON
	URL string
	// One of the 7 themes supported by Swagger, e.g 'nord'.
	Theme string
	// Filter controls whether to display a tag-based filter on the OpenAPI UI
	Filter bool
	// RequestDuration controls whether to display the request duration after firing a request.
	RequestDuration bool
	// FaviconURL is the location of favicon image to display
	FaviconURL string
}

// aliases
type Callback map[string]PathItem
type Paths map[string]PathItem
type SecurityRequirement map[string][]string
type SchemaDefinition map[string]*Schema
