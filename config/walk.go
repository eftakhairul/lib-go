package config

import (
	"fmt"
	"reflect"
)

// isLeafType reports whether t should be treated as a leaf value (parsed
// directly from a string) rather than recursed into as a nested struct.
func isLeafType(t reflect.Type) bool {
	elem := t
	if elem.Kind() == reflect.Ptr {
		elem = elem.Elem()
	}
	if elem.Kind() != reflect.Struct {
		return true
	}
	return implementsTextUnmarshaler(elem)
}

// walkFields recurses through v's exported fields. Nested/embedded struct
// fields (that don't implement encoding.TextUnmarshaler) are recursed into;
// every other exported field is passed to visit along with its
// PARENT_CHILD-joined auto-derived name. prefix is the accumulated
// screaming-snake path of enclosing named struct fields (embedded fields do
// not add a segment).
func walkFields(v reflect.Value, prefix string, visit func(field reflect.StructField, fv reflect.Value, autoName string) error) error {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Process exported fields or anonymous (embedded) fields
		if !field.IsExported() && !field.Anonymous {
			continue
		}

		fv := v.Field(i)

		isLeaf := isLeafType(field.Type)

		if !isLeaf {
			childPrefix := prefix
			if !field.Anonymous {
				childPrefix = joinPrefix(prefix, toScreamingSnake(field.Name))
			}

			target := fv
			if target.Kind() == reflect.Ptr {
				if target.IsNil() {
					if !target.CanSet() {
						return fmt.Errorf("config: field %s is not settable (reached through an unexported anonymous field)", field.Name)
					}
					target.Set(reflect.New(target.Type().Elem()))
				}
				target = target.Elem()
			}

			if err := walkFields(target, childPrefix, visit); err != nil {
				return err
			}
			continue
		}

		autoName := joinPrefix(prefix, toScreamingSnake(field.Name))
		if err := visit(field, fv, autoName); err != nil {
			return err
		}
	}
	return nil
}
