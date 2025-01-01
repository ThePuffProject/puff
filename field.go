package puff

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"strconv"

	"github.com/ThePuffProject/puff/openapi"
)

// NoFields indicates that the route takes in no fields. It is
// equivlent to an empty struct.
type NoFields struct{}

// DEFINITIONS (at start of application)

// handleInputSchema handles a request input schema.
func handleInputSchema(parameters *[]openapi.Parameter, fieldsType reflect.Type) error {
	if fieldsType == nil {
		*parameters = []openapi.Parameter{}
		return nil
	}

	schema := fieldsType
	params := []openapi.Parameter{}
	for i := range schema.NumField() {
		field := schema.Field(i)
		if field.Anonymous {
			handleInputSchema(&params, field.Type)
			continue
		}
		param, err := newParameterDefinition(field)
		if err != nil {
			return err
		}
		params = append(params, param)
	}

	*parameters = params
	return nil
}

// newParameterDefinition creates a new OpenAPI Parameter definition from a reflect.StructField.
func newParameterDefinition(field reflect.StructField) (openapi.Parameter, error) {
	p := new(openapi.Parameter)
	var err error

	// name
	p.Name = field.Name

	// in
	p.In = field.Tag.Get("kind")
	switch p.In {
	case "header", "path", "query", "cookie", "body", "formdata":
	default:
		if field.Type == reflect.TypeFor[*File]() {
			p.In = "file"
			break
		}
		return openapi.Parameter{}, fmt.Errorf("struct tag `kind` on field %s expected `header`, `path`, `query`, `cookie`, `body`, or `formdata`", p.Name)
	}

	// schema
	p.Schema, err = newSchemaDefinition(field.Type)
	if err != nil {
		return openapi.Parameter{}, fmt.Errorf("handling the field type on field %s encountered an unexpected error: %v", err, p.Name)
	}
	// schema.format
	specifiedformat := field.Tag.Get("format")
	if specifiedformat != "" {
		p.Schema.Format = specifiedformat
	}

	// description
	p.Description = field.Tag.Get("description")

	// required
	if p.Required, err = boolFromString(field.Tag.Get("required"), true); err != nil {
		return openapi.Parameter{}, fmt.Errorf("struct tag `required` on field %s expected either `true` or `false`", p.Name)
	}
	// FIXME: set allow empty values - https://swagger.io/docs/specification/v3_0/describing-parameters/#Empty-Valued%20and%20Nullable%20Parameters

	// deprecated
	if p.Deprecated, err = boolFromString(field.Tag.Get("deprecated"), false); err != nil {
		return openapi.Parameter{}, fmt.Errorf("struct tag `deprecated` on field %s expected either `true` or `false`", p.Name)
	}

	return *p, nil
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
		return &openapi.Schema{
			Type:                 "object",
			AdditionalProperties: s,
		}, nil
	case reflect.Array, reflect.Slice:
		// https://spec.openapis.org/oas/v3.0.3.html#parameter-object-examples
		s, err := newSchemaDefinition(t.Elem())
		if err != nil {
			return nil, fmt.Errorf("handling array/slice element type error: %v", err)
		}
		return &openapi.Schema{
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
				return nil, fmt.Errorf("handling field %s encountered an unexpected error: %s", f.Name, err.Error())
			}
			required, err := boolFromString(f.Tag.Get("required"), true)
			if err != nil {
				return nil, fmt.Errorf("struct tag `required` for field %s on struct %s is not a boolean: %s", f.Name, t.Name(), err.Error())
			}
			if required {
				s.Required = append(s.Required, f.Name)
			}
		}
		return s, nil
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

// handleBasicType returns a Schema for the following kinds,
// string, int, int8, int16, int32, int64, uint, uint8, uint16
// uint32, uint64, float32, float64, bool.
func handleBasicType(k reflect.Kind) *openapi.Schema {
	switch k {
	case reflect.String:
		// json data type
		return &openapi.Schema{
			Type:   "string",
			Format: "string",
			Examples: map[string]openapi.Example{
				"Example1": {
					Value: "string",
				},
			},
		}
	case reflect.Int:
		// json data type
		return &openapi.Schema{
			Type:   "integer",
			Format: "int32",
			Examples: map[string]openapi.Example{
				"Example1": {
					Value: 123,
				},
			},
		}
	case reflect.Int8:
		return &openapi.Schema{
			Type:   "integer",
			Format: "int8",
			Examples: map[string]openapi.Example{
				"Example1": {
					Value: -128,
				},
			},
		}
	case reflect.Int16:
		return &openapi.Schema{
			Type:   "integer",
			Format: "int16",
			Examples: map[string]openapi.Example{
				"Example1": {
					Value: -32768,
				},
			},
		}
	case reflect.Int32:
		return &openapi.Schema{
			Type:   "integer",
			Format: "int32",
			Examples: map[string]openapi.Example{
				"Example1": {
					Value: -2147483648,
				},
			},
		}
	case reflect.Int64:
		return &openapi.Schema{
			Type:   "integer",
			Format: "int64",
			Examples: map[string]openapi.Example{
				"Example1": {
					Value: -9223372036854775808,
				},
			},
		}
	case reflect.Uint:
		return &openapi.Schema{
			Type:    "integer",
			Format:  "int32",
			Minimum: "0",
			Examples: map[string]openapi.Example{
				"Example1": {
					Value: 2 ^ 64,
				},
			},
		}
	case reflect.Uint8:
		return &openapi.Schema{
			Type:    "integer",
			Format:  "int8",
			Minimum: "0",
			Examples: map[string]openapi.Example{
				"Example1": {
					Value: 2 ^ 8 - 1,
				},
			},
		}
	case reflect.Uint16:
		return &openapi.Schema{
			Type:    "integer",
			Format:  "int16",
			Minimum: "0",
			Examples: map[string]openapi.Example{
				"Example1": {
					Value: 2 ^ 16 - 1,
				},
			},
		}
	case reflect.Uint32:
		return &openapi.Schema{
			Type:    "integer",
			Format:  "int32",
			Minimum: "0",
			Examples: map[string]openapi.Example{
				"Example1": {
					Value: 2 ^ 32 - 1,
				},
			},
		}
	case reflect.Uint64:
		return &openapi.Schema{
			Type:    "integer",
			Format:  "int64",
			Minimum: "0",
			Examples: map[string]openapi.Example{
				"Example1": {
					Value: 2 ^ 64 - 1,
				},
			},
		}
	case reflect.Float32:
		return &openapi.Schema{
			Type:   "number",
			Format: "float",
			Examples: map[string]openapi.Example{
				"Example1": {
					Value: 3.4e+38,
				},
			},
		}
	case reflect.Float64:
		return &openapi.Schema{
			Type:   "number",
			Format: "double",
			Examples: map[string]openapi.Example{
				"Example1": {
					Value: 1.7e+308,
				},
			},
		}
	case reflect.Bool:
		return &openapi.Schema{
			Type: "boolean",
			Examples: map[string]openapi.Example{
				"Example1": {
					Value: true,
				},
			},
		}
	}
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

// POPULATION (on route serving)

func fieldsFromIncoming(c *Context, r *Route, m []string) (any, error) {
	if r.fieldsType == nil {
		return NoFields{}, nil
	}

	i := 0 // tracks index for m
	v := reflect.New(r.fieldsType).Elem()

	for _, param := range r.params {
		field := v.FieldByName(param.Name)
		value := ""

		switch param.In {
		case "file":
			file, err := getFileParam(c, &param)
			if err != nil {
				return nil, err
			}
			field.Set(reflect.ValueOf(file))
			continue
		case "header":
			value = c.GetRequestHeader(param.Name)
		case "cookie":
			value = c.GetCookie(param.Name)
		case "path":
			if i >= len(m) {
				return nil, fmt.Errorf("not enough matches")
			}
			value = m[i]
			i += 1
		case "body":
			val, err := c.GetBody()
			if err != nil {
				return nil, fmt.Errorf("read body error: %v", err)
			}
			value = string(val)
		case "query":
			value = c.GetQueryParam(param.Name)
		case "formdata":
			value = c.GetFormValue(param.Name)
		}

		if param.Required && value == "" {
			return nil, fmt.Errorf("required field %s not provided", param.Name)
		} else if value == "" {
			continue
		}
		switch field.Type().Kind() {
		case reflect.String:
			field.Set(reflect.ValueOf(value))
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			i, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return nil, valueCannotBeSet(value, param.Name)
			}
			field.SetInt(i)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			u, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return nil, valueCannotBeSet(value, param.Name)
			}
			field.SetUint(u)
		case reflect.Float32, reflect.Float64:
			f, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return nil, valueCannotBeSet(value, param.Name)
			}
			field.SetFloat(f)
		case reflect.Bool:
			b, err := strconv.ParseBool(value)
			if err != nil {
				return nil, valueCannotBeSet(value, param.Name)
			}
			field.SetBool(b)
		default:
			z := reflect.New(field.Type()).Interface()
			err := json.Unmarshal([]byte(value), z)
			if err != nil {
				return nil, valueCannotBeSet(value, param.Name)
			}
			field.Set(reflect.ValueOf(z))
		}
	}

	val := v.Interface()
	return val, nil
}
