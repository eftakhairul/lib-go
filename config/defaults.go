package config

import (
	"fmt"
	"reflect"
)

// applyDefaults walks v and, for each leaf field carrying a `default` tag,
// parses the tag value into the field. Fields without a `default` tag are
// left untouched (zero value).
func applyDefaults(v reflect.Value) error {
	return walkFields(v, "", func(field reflect.StructField, fv reflect.Value, autoName string) error {
		def, ok := field.Tag.Lookup("default")
		if !ok {
			return nil
		}
		if err := setFieldFromString(fv, def); err != nil {
			return fmt.Errorf("config: default value for field %s: %w", field.Name, err)
		}
		return nil
	})
}
