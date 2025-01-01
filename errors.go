package puff

import "fmt"

func regexpError(s string, e error) error {
	return fmt.Errorf("regexp error: creating regexp for route with fullpath %s encountered an error: %v", s, e)
}

func schemaError(e error) error {
	return fmt.Errorf("schema error: %v", e)
}

func valueCannotBeSet(v string, f string) error {
	return fmt.Errorf("value %s cannot be set into field %s", v, f)
}
