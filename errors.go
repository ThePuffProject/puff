package puff

import "fmt"

func FieldTypeError(value string, expectedType string) error {
	return fmt.Errorf(
		"type error: the value %s cant be used as the expected type %s",
		value,
		expectedType,
	)
}

func BadFieldType(k string, got string, expected string) error {
	return fmt.Errorf(
		"type error: the value for key %s: %s cannot be used for expected type %s",
		k,
		got,
		expected,
	)
}

func ExpectedButNotFound(k string) error {
	return fmt.Errorf("expected key %s but not found in json", k)
}

func UnexpectedJSONKey(k string) error {
	return fmt.Errorf("unexpected json key: %s", k)
}

func InvalidJSONError(v string) error {
	return fmt.Errorf("expected json, but got invalid json")
}
