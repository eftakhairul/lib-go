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
//
// Cycle detection: self-referential struct types (e.g. a linked-list-shaped
// config struct) would otherwise cause unbounded recursion. walkFields seeds
// an internal set of struct types on the current recursion path (starting
// with v's own type) and delegates to walkFieldsRec, which errors out if the
// same type is encountered again while still inside its own recursion.
func walkFields(v reflect.Value, prefix string, visit func(field reflect.StructField, fv reflect.Value, autoName string) error) error {
	seen := map[reflect.Type]bool{v.Type(): true}
	return walkFieldsRec(v, prefix, visit, seen)
}

// walkFieldsRec is the cycle-tracking implementation behind walkFields. seen
// holds the struct types currently being recursed into on the active path;
// it is mutated (added before recursing into a branch, removed once that
// branch returns) so that the same type may still legitimately appear in
// separate, non-overlapping branches of the struct tree.
func walkFieldsRec(v reflect.Value, prefix string, visit func(field reflect.StructField, fv reflect.Value, autoName string) error, seen map[reflect.Type]bool) error {
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

			tt := target.Type()
			if seen[tt] {
				return fmt.Errorf("config: circular struct reference detected at field %s (type %s)", field.Name, tt.String())
			}
			seen[tt] = true
			err := walkFieldsRec(target, childPrefix, visit, seen)
			delete(seen, tt)
			if err != nil {
				return err
			}
			continue
		}

		autoName := joinPrefix(prefix, toScreamingSnake(field.Name))
		if !fv.CanSet() {
			return fmt.Errorf("config: field %s is not settable (reached through an unexported anonymous field)", field.Name)
		}
		if err := visit(field, fv, autoName); err != nil {
			return err
		}
	}
	return nil
}
