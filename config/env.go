package config

import (
	"fmt"
	"os"
	"reflect"
)

// applyEnv walks v and, for each leaf field, resolves an environment
// variable name (the field's explicit `env` tag verbatim, or its
// auto-derived PARENT_CHILD name with globalPrefix prepended) and, if that
// variable is set, parses it into the field. Fields whose variable is not
// set are left untouched.
func applyEnv(v reflect.Value, globalPrefix string) error {
	return walkFields(v, "", func(field reflect.StructField, fv reflect.Value, autoName string) error {
		name, explicit := field.Tag.Lookup("env")
		if !explicit {
			name = autoName
			if globalPrefix != "" {
				name = globalPrefix + "_" + name
			}
		}

		val, ok := os.LookupEnv(name)
		if !ok {
			return nil
		}
		if err := setFieldFromString(fv, val); err != nil {
			return fmt.Errorf("config: env var %s for field %s: %w", name, field.Name, err)
		}
		return nil
	})
}
