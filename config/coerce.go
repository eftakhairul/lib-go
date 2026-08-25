// config/coerce.go
package config

import (
	"encoding"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var textUnmarshalerType = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()

// implementsTextUnmarshaler reports whether *t (a pointer to t) implements
// encoding.TextUnmarshaler, which is how types like time.Time expose it.
func implementsTextUnmarshaler(t reflect.Type) bool {
	return reflect.PointerTo(t).Implements(textUnmarshalerType)
}

var durationType = reflect.TypeOf(time.Duration(0))

// setFieldFromString parses s and writes it into the addressable value v,
// used for both `default` tag values and environment variable values.
func setFieldFromString(v reflect.Value, s string) error {
	if v.CanAddr() && v.CanSet() && implementsTextUnmarshaler(v.Type()) {
		return v.Addr().Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(s))
	}

	switch v.Kind() {
	case reflect.Ptr:
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		return setFieldFromString(v.Elem(), s)

	case reflect.String:
		v.SetString(s)
		return nil

	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return fmt.Errorf("parsing bool from %q: %w", s, err)
		}
		v.SetBool(b)
		return nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if v.Type() == durationType {
			d, err := time.ParseDuration(s)
			if err != nil {
				return fmt.Errorf("parsing duration from %q: %w", s, err)
			}
			v.SetInt(int64(d))
			return nil
		}
		n, err := strconv.ParseInt(s, 10, v.Type().Bits())
		if err != nil {
			return fmt.Errorf("parsing int from %q: %w", s, err)
		}
		v.SetInt(n)
		return nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(s, 10, v.Type().Bits())
		if err != nil {
			return fmt.Errorf("parsing uint from %q: %w", s, err)
		}
		v.SetUint(n)
		return nil

	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(s, v.Type().Bits())
		if err != nil {
			return fmt.Errorf("parsing float from %q: %w", s, err)
		}
		v.SetFloat(f)
		return nil

	case reflect.Slice:
		parts := strings.Split(s, ",")
		dst := reflect.MakeSlice(v.Type(), len(parts), len(parts))
		for i, part := range parts {
			if err := setFieldFromString(dst.Index(i), part); err != nil {
				return fmt.Errorf("parsing slice element %d from %q: %w", i, part, err)
			}
		}
		v.Set(dst)
		return nil

	default:
		return fmt.Errorf("unsupported field type %s", v.Type())
	}
}
