package puff

import (
	"fmt"
	"log/slog"
	"reflect"
	"strconv"

	"github.com/ThePuffProject/puff/openapi"
)

// NoFields indicates that the route takes in no fields. It is
// equivlent to an empty struct.
type NoFields struct{}

// handleBasicType returns a Schema for the following kinds,
// string, int, int8, int16, int32, int64, uint, uint8, uint16
// uint32, uint64, float32, float64, bool.
func handleBasicType(k reflect.Kind) *openapi.Schema {
	switch k {
	case reflect.String:
		// json data type
		return &Schema{
			Format:   "string",
			Examples: []any{"string"},
		}
	case reflect.Int:
		// json data type
		return &Schema{
			Format:   "integer",
			Examples: []any{"int"},
		}
	case reflect.Int8:
		return &Schema{
			// https://spec.openapis.org/registry/format/int8
			Format:   "int8",
			Examples: []any{"-128"},
		}
	case reflect.Int16:
		return &Schema{
			// https://spec.openapis.org/registry/format/int16
			Format:   "int16",
			Examples: []any{"-32,768"},
		}
	case reflect.Int32:
		return &Schema{
			// https://spec.openapis.org/registry/format/int32
			Format:   "int32",
			Examples: []any{"-2,147,483,648"},
		}
	case reflect.Int64:
		return &Schema{
			// https://spec.openapis.org/registry/format/int64
			Format:   "int64",
			Examples: []any{"-9,223,372,036,854,775,808"},
		}
	case reflect.Uint:
		return &Schema{
			Format:   "int",
			Minimum:  "0",
			Examples: []any{"uint"},
		}
	case reflect.Uint8:
		return &Schema{
			Format:   "int8",
			Examples: []any{"255"},
			Minimum:  "0",
		}
	case reflect.Uint16:
		return &Schema{
			Format:   "int16",
			Examples: []any{"65,535"},
			Minimum:  "0",
		}
	case reflect.Uint32:
		return &Schema{
			Format:   "int32",
			Examples: []any{"4,294,967,295"},
			Minimum:  "0",
		}
	case reflect.Uint64:
		return &Schema{
			Format:   "int64",
			Examples: []any{"18,446,744,073,709,551,615"},
			Minimum:  "0",
		}
	case reflect.Float32:
		return &Schema{
			Format:   "float",
			Examples: []any{"3.4e+38"},
		}
	case reflect.Float64:
		return &Schema{
			Format:   "double",
			Examples: []any{"1.7e+308"},
		}
	case reflect.Bool:
		return &Schema{
			Format:   "bool",
			Examples: []any{false},
		}
	}
	return nil
}

// newSchemaDefinition creates a new OpenAPI Schema definition from a reflect.Type.
func newSchemaDefinition(t reflect.Type) (*openapi.Schema, error) {
	var err error
	switch t.Kind() {
	case reflect.Map:
		// https://swagger.io/docs/specification/v3_0/data-models/dictionaries/
		// https://stackoverflow.com/a/73626840/16467184
		if t.Key().Kind() != reflect.String {
			slog.Warn("As of OpenAPI 3.1, dictionary (map) key types are not able to be specified, since OpenAPI expects map key types to always be strings. puff will continue to work as normal.")
		}
		s, err := newSchemaDefinition(t.Elem())
		if err != nil {
			return nil, fmt.Errorf("map value type error: %v", err)
		}
		return &Schema{
			Type:                 "object",
			AdditionalProperties: s,
		}, nil
	case reflect.Array, reflect.Slice:
		// https://spec.openapis.org/oas/v3.0.3.html#parameter-object-examples
		s, err := newSchemaDefinition(t.Elem())
		if err != nil {
			return nil, fmt.Errorf("handling array/slice element type error: %v", err)
		}
		return &Schema{
			Type:  "array",
			Items: s,
		}, nil
	case reflect.Struct:
		// https://spec.openapis.org/oas/v3.0.3.html#simple-model
		s := &openapi.Schema{
			Type:       "object",
			Properties: map[string]*openapi.Schema{},
			Required:   []string{},
		}
		for i := range t.NumField() {
			f := t.Field(i)
			s.Properties[f.Name], err = newSchemaDefinition(f.Type)
			if err != nil {
				return nil, fmt.Errorf("handling field %s encountered an unexpected error: %s", f.Name, t.Name(), err.Error())
			}
			required, err := boolFromString(f.Tag.Get("required"), true)
			if err != nil {
				return nil, fmt.Errorf("struct tag `required` for field %s on struct %s is not a boolean: %s", f.Name, t.Name(), err.Error())
			}
			if required {
				s.Required = append(s.Required, f.Name)
			}
		}
	case
		reflect.String,
		reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Float32,
		reflect.Float64,
		reflect.Bool:
		return handleBasicType(t.Kind()), nil
	}
	return nil, fmt.Errorf("unsupported kind: %s. Documentation: %s/fields.", t.Kind(), documentationURL)
}

// newParameterDefinition creates a new OpenAPI Parameter definition from a reflect.StructField.
func newParameterDefinition(field reflect.StructField) (Parameter, error) {
	p := new(Parameter)
	var err error

	// name
	p.Name = field.Name

	// in
	p.In = field.Tag.Get("kind")
	switch p.In {
	case "header", "path", "query", "cookie", "body", "formdata":
	default:
		return Parameter{}, fmt.Errorf("struct tag `kind` on field %s expected `header`, `path`, `query`, `cookie`, `body`, or `formdata`")
	}

	// schema
	p.Schema, err = newSchemaDefinition(field.Type)
	if err != nil {
		return Parameter{}, fmt.Errorf("handling the field type on field %s encountered an unexpected error: %v", err)
	}
	// schema.format
	specifiedformat := field.Tag.Get("format")
	if specifiedformat != "" {
		slog.Warn(fmt.Sprintf("specified format %s overrides the format created by puff %s", specifiedformat, p.Schema.Format))
		p.Schema.Format = specifiedformat
	}

	// description
	p.Description = field.Tag.Get("description")

	// required
	if p.Required, err = boolFromString("required", true); err != nil {
		return Parameter{}, fmt.Errorf("struct tag `required` on field %s expected either `true` or `false`")
	}

	// deprecated
	if p.Deprecated, err = boolFromString("deprecated", false); err != nil {
		return Parameter{}, fmt.Errorf("struct tag `deprecated` on field %s expected either `true` or `false`")
	}

	return *p, nil
}

// handleInputSchema handles a request input schema.
func handleInputSchema(route *Route) error {
	if route.fieldsType == nil {
		route.params = []Parameter{}
		return nil
	}

	schema := route.fieldsType
	params := make([]Parameter, schema.NumField())
	var err error
	for i := range schema.NumField() {
		field := schema.Field(i)
		params[i], err = newParameterDefinition(field)
		if err != nil {
			return err
		}
	}

	route.params = params
	return nil
}

// boolFromString evaluates a boolean value from a string. If s is empty,
// it fallsback to def. Otherwise, it attempts to get the boolean value,
// and returns an error if s is otherwise invalid.
func boolFromString(s string, def bool) (bool, error) {
	if s == "" { // not specified, go to default value
		return def, nil
	}
	b, err := strconv.ParseBool(s) // specified, parse bool
	if err != nil {                // not a parsable bool
		return false, err
	}
	return b, nil
}
