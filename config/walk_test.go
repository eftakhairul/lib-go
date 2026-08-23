package config

import (
	"reflect"
	"testing"
	"time"
)

type walkInner struct {
	Host string
	Port int `env:"PGPORT"`
}

type walkEmbedded struct {
	Nickname string
}

type walkOuter struct {
	walkEmbedded
	Inner   walkInner
	Ptr     *walkInner
	LogTime time.Time
	skipped string //nolint:unused // exercises unexported-field skip
}

func TestWalkFields_LeafNamesAndRecursion(t *testing.T) {
	var out walkOuter
	var gotNames []string
	err := walkFields(reflect.ValueOf(&out).Elem(), "", func(field reflect.StructField, fv reflect.Value, autoName string) error {
		gotNames = append(gotNames, autoName)
		return nil
	})
	if err != nil {
		t.Fatalf("walkFields: %v", err)
	}

	want := []string{
		"NICKNAME",   // embedded struct field, no extra prefix segment beyond its own name
		"INNER_HOST",
		"INNER_PORT",
		"PTR_HOST",
		"PTR_PORT",
		"LOG_TIME",
	}
	if !reflect.DeepEqual(gotNames, want) {
		t.Errorf("visited leaf autoNames = %v, want %v", gotNames, want)
	}

	if out.Ptr == nil {
		t.Fatal("expected walkFields to allocate nil pointer-to-struct field Ptr")
	}
}

// privateHelper is an unexported struct type used to test error handling
// for anonymous pointer fields to unexported types.
type privateHelper struct {
	Value string
}

type structWithUnexportedAnonymousPtr struct {
	*privateHelper // anonymous pointer field to unexported type
}

func TestWalkFields_UnexportedAnonymousPtrReturnsError(t *testing.T) {
	var out structWithUnexportedAnonymousPtr
	err := walkFields(reflect.ValueOf(&out).Elem(), "", func(field reflect.StructField, fv reflect.Value, autoName string) error {
		return nil
	})

	if err == nil {
		t.Fatal("expected walkFields to return an error for unexported anonymous pointer field, got nil")
	}
	if err.Error() != "config: field privateHelper is not settable (reached through an unexported anonymous field)" {
		t.Errorf("unexpected error message: %v", err)
	}
}
