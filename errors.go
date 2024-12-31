package puff

import "fmt"

func fieldTypeError(value string, expectedType string) error {
	return fmt.Errorf(
		"type error: the value %s cant be used as the expected type %s",
		value,
		expectedType,
	)
}

func badFieldType(k string, got string, expected string) error {
	return fmt.Errorf(
		"type error: the value for key %s: %s cannot be used for expected type %s",
		k,
		got,
		expected,
	)
}

func regexpError(s string, e error) error {
	return fmt.Errorf("regexp error: creating regexp for route with fullpath %s encountered an error: %v", s, e)
}

func schemaError(e error) error {
	return fmt.Errorf("schema error: %v", e)
}

func expectedButNotFound(k string) error {
	return fmt.Errorf("expected key %s but not found in json", k)
}

func unexpectedJSONKey(k string) error {
	return fmt.Errorf("unexpected json key: %s", k)
}

func invalidJSONError(v string) error {
	return fmt.Errorf("got invalid json")
}
