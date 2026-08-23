// config/coerce_test.go
package config

import (
	"reflect"
	"testing"
	"time"
)

type coerceTarget struct {
	S   string
	B   bool
	I   int
	I64 int64
	U   uint
	F64 float64
	D   time.Duration
	SS  []string
	PI  *int
	T   time.Time
}

func fieldOf(t *testing.T, target *coerceTarget, name string) reflect.Value {
	t.Helper()
	return reflect.ValueOf(target).Elem().FieldByName(name)
}

func TestSetFieldFromString_Scalars(t *testing.T) {
	var target coerceTarget

	if err := setFieldFromString(fieldOf(t, &target, "S"), "hello"); err != nil {
		t.Fatalf("string: %v", err)
	}
	if target.S != "hello" {
		t.Errorf("S = %q, want hello", target.S)
	}

	if err := setFieldFromString(fieldOf(t, &target, "B"), "true"); err != nil {
		t.Fatalf("bool: %v", err)
	}
	if !target.B {
		t.Errorf("B = false, want true")
	}

	if err := setFieldFromString(fieldOf(t, &target, "I"), "42"); err != nil {
		t.Fatalf("int: %v", err)
	}
	if target.I != 42 {
		t.Errorf("I = %d, want 42", target.I)
	}

	if err := setFieldFromString(fieldOf(t, &target, "U"), "7"); err != nil {
		t.Fatalf("uint: %v", err)
	}
	if target.U != 7 {
		t.Errorf("U = %d, want 7", target.U)
	}

	if err := setFieldFromString(fieldOf(t, &target, "F64"), "3.5"); err != nil {
		t.Fatalf("float: %v", err)
	}
	if target.F64 != 3.5 {
		t.Errorf("F64 = %v, want 3.5", target.F64)
	}
}

func TestSetFieldFromString_Duration(t *testing.T) {
	var target coerceTarget
	if err := setFieldFromString(fieldOf(t, &target, "D"), "1500ms"); err != nil {
		t.Fatalf("duration: %v", err)
	}
	if target.D != 1500*time.Millisecond {
		t.Errorf("D = %v, want 1500ms", target.D)
	}
}

func TestSetFieldFromString_StringSlice(t *testing.T) {
	var target coerceTarget
	if err := setFieldFromString(fieldOf(t, &target, "SS"), "a,b,c"); err != nil {
		t.Fatalf("slice: %v", err)
	}
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(target.SS, want) {
		t.Errorf("SS = %v, want %v", target.SS, want)
	}
}

func TestSetFieldFromString_Pointer(t *testing.T) {
	var target coerceTarget
	if err := setFieldFromString(fieldOf(t, &target, "PI"), "9"); err != nil {
		t.Fatalf("pointer: %v", err)
	}
	if target.PI == nil || *target.PI != 9 {
		t.Errorf("PI = %v, want pointer to 9", target.PI)
	}
}

func TestSetFieldFromString_TextUnmarshaler(t *testing.T) {
	var target coerceTarget
	if err := setFieldFromString(fieldOf(t, &target, "T"), "2026-08-22T00:00:00Z"); err != nil {
		t.Fatalf("time.Time: %v", err)
	}
	want, _ := time.Parse(time.RFC3339, "2026-08-22T00:00:00Z")
	if !target.T.Equal(want) {
		t.Errorf("T = %v, want %v", target.T, want)
	}
}

func TestSetFieldFromString_UnsupportedKind(t *testing.T) {
	var target struct {
		M map[string]string
	}
	err := setFieldFromString(reflect.ValueOf(&target).Elem().FieldByName("M"), "x")
	if err == nil {
		t.Fatal("expected error for unsupported kind, got nil")
	}
}

func TestSetFieldFromString_BadValue(t *testing.T) {
	var target coerceTarget
	err := setFieldFromString(fieldOf(t, &target, "I"), "not-a-number")
	if err == nil {
		t.Fatal("expected error for unparsable int, got nil")
	}
}

func TestSetFieldFromString_Uint8Overflow(t *testing.T) {
	var target struct {
		U8 uint8
	}
	err := setFieldFromString(reflect.ValueOf(&target).Elem().FieldByName("U8"), "300")
	if err == nil {
		t.Fatal("expected error for uint8 overflow (300 > 255), got nil")
	}
	if target.U8 != 0 {
		t.Errorf("U8 = %d, want 0 (unchanged after error)", target.U8)
	}
}

func TestSetFieldFromString_Int8Overflow(t *testing.T) {
	var target struct {
		I8 int8
	}
	err := setFieldFromString(reflect.ValueOf(&target).Elem().FieldByName("I8"), "200")
	if err == nil {
		t.Fatal("expected error for int8 overflow (200 > 127), got nil")
	}
	if target.I8 != 0 {
		t.Errorf("I8 = %d, want 0 (unchanged after error)", target.I8)
	}
}

func TestSetFieldFromString_Int8Underflow(t *testing.T) {
	var target struct {
		I8 int8
	}
	err := setFieldFromString(reflect.ValueOf(&target).Elem().FieldByName("I8"), "-200")
	if err == nil {
		t.Fatal("expected error for int8 underflow (-200 < -128), got nil")
	}
	if target.I8 != 0 {
		t.Errorf("I8 = %d, want 0 (unchanged after error)", target.I8)
	}
}

func TestSetFieldFromString_Uint16Overflow(t *testing.T) {
	var target struct {
		U16 uint16
	}
	err := setFieldFromString(reflect.ValueOf(&target).Elem().FieldByName("U16"), "100000")
	if err == nil {
		t.Fatal("expected error for uint16 overflow (100000 > 65535), got nil")
	}
	if target.U16 != 0 {
		t.Errorf("U16 = %d, want 0 (unchanged after error)", target.U16)
	}
}
