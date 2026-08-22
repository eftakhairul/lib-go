# Config Loader Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `github.com/eftakhairul/lib-go/config`, a package that loads a struct from a JSON/YAML file and/or environment variables, with env taking precedence over file, which takes precedence over `default` tags.

**Architecture:** A small reflection-based walker (`walkFields`) recurses through a struct once per pass (defaults, then file unmarshal, then env), invoking a per-pass leaf handler. Type coercion from string (env values, default tag values) is centralized in one `setFieldFromString` function shared by both the defaults pass and the env pass. File parsing reuses `encoding/json` directly for `.json` and `sigs.k8s.io/yaml` (which itself converts YAML to JSON then calls `encoding/json`) for `.yaml`/`.yml`, so a single `json` tag set drives both formats.

**Tech Stack:** Go 1.22 (toolchain available: go1.26.5), stdlib `reflect`/`encoding/json`/`strconv`/`time`/`os`, external dependency `sigs.k8s.io/yaml`.

**Spec:** `docs/superpowers/specs/2026-08-22-config-loader-design.md`

## Global Constraints

- Module: `github.com/eftakhairul/lib-go`, directive `go 1.22`.
- New package lives at `config/` (import path `github.com/eftakhairul/lib-go/config`), alongside existing `logger/` and `secure/` packages — do not modify those.
- Only new dependency: `sigs.k8s.io/yaml`.
- Precedence, high to low: env var → file value → `default` tag → zero value.
- Env var name: explicit `env:"NAME"` tag used verbatim; otherwise auto-derived from the Go field name via CamelCase→SCREAMING_SNAKE_CASE, with nested/embedded struct fields joined as `PARENT_CHILD`.
- `WithEnvPrefix(prefix)` prepends `prefix + "_"` only to auto-derived names, never to explicit `env` tags.
- Do not import or depend on the existing `secure` package.
- No `required` tag, no file watching, no multi-file layering — out of scope for this plan.

---

### Task 1: Module bootstrap + naming/prefix helpers

**Files:**
- Create: `go.mod`
- Create: `config/names.go`
- Test: `config/names_test.go`

**Interfaces:**
- Produces: `toScreamingSnake(name string) string`, `joinPrefix(parent, child string) string` — used by Task 3's walker to build env var names.

- [ ] **Step 1: Initialize the Go module**

Run:
```bash
go mod init github.com/eftakhairul/lib-go
```
Then edit `go.mod` so the `go` directive reads `go 1.22` (the `go mod init` command may write a different version based on the installed toolchain; set it explicitly to `1.22`).

- [ ] **Step 2: Write the failing test for name conversion**

```go
// config/names_test.go
package config

import "testing"

func TestToScreamingSnake(t *testing.T) {
	cases := map[string]string{
		"Port":     "PORT",
		"LogLevel": "LOG_LEVEL",
		"HTTPPort": "HTTP_PORT",
		"DB":       "DB",
		"ID":       "ID",
	}
	for in, want := range cases {
		if got := toScreamingSnake(in); got != want {
			t.Errorf("toScreamingSnake(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestJoinPrefix(t *testing.T) {
	cases := []struct {
		parent, child, want string
	}{
		{"", "HOST", "HOST"},
		{"DB", "HOST", "DB_HOST"},
	}
	for _, c := range cases {
		if got := joinPrefix(c.parent, c.child); got != c.want {
			t.Errorf("joinPrefix(%q, %q) = %q, want %q", c.parent, c.child, got, c.want)
		}
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./config/... -run 'TestToScreamingSnake|TestJoinPrefix' -v`
Expected: FAIL — `toScreamingSnake` and `joinPrefix` are undefined (package doesn't compile yet).

- [ ] **Step 4: Write minimal implementation**

```go
// config/names.go
package config

import (
	"regexp"
	"strings"
)

var (
	matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCap   = regexp.MustCompile("([a-z0-9])([A-Z])")
)

// toScreamingSnake converts a Go identifier like "HTTPPort" or "LogLevel"
// into "HTTP_PORT" / "LOG_LEVEL" for use as an environment variable name.
func toScreamingSnake(name string) string {
	s := matchFirstCap.ReplaceAllString(name, "${1}_${2}")
	s = matchAllCap.ReplaceAllString(s, "${1}_${2}")
	return strings.ToUpper(s)
}

// joinPrefix joins a parent env-name prefix and a child segment with "_",
// omitting the separator when parent is empty.
func joinPrefix(parent, child string) string {
	if parent == "" {
		return child
	}
	return parent + "_" + child
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./config/... -run 'TestToScreamingSnake|TestJoinPrefix' -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add go.mod config/names.go config/names_test.go
git commit -m "feat(config): bootstrap module and env-name helpers"
```

---

### Task 2: String-to-field type coercion

**Files:**
- Create: `config/coerce.go`
- Test: `config/coerce_test.go`

**Interfaces:**
- Consumes: nothing from Task 1.
- Produces: `setFieldFromString(v reflect.Value, s string) error` — used by Task 4 (defaults) and Task 5 (env) to write a parsed string into a struct field.

- [ ] **Step 1: Write the failing tests**

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./config/... -run TestSetFieldFromString -v`
Expected: FAIL — `setFieldFromString` is undefined.

- [ ] **Step 3: Write minimal implementation**

```go
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
	if v.CanAddr() && implementsTextUnmarshaler(v.Type()) {
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
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return fmt.Errorf("parsing int from %q: %w", s, err)
		}
		v.SetInt(n)
		return nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return fmt.Errorf("parsing uint from %q: %w", s, err)
		}
		v.SetUint(n)
		return nil

	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return fmt.Errorf("parsing float from %q: %w", s, err)
		}
		v.SetFloat(f)
		return nil

	case reflect.Slice:
		if v.Type().Elem().Kind() != reflect.String {
			return fmt.Errorf("unsupported slice element type %s", v.Type().Elem())
		}
		parts := strings.Split(s, ",")
		v.Set(reflect.ValueOf(parts))
		return nil

	default:
		return fmt.Errorf("unsupported field type %s", v.Type())
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./config/... -run TestSetFieldFromString -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add config/coerce.go config/coerce_test.go
git commit -m "feat(config): add string-to-field type coercion"
```

---

### Task 3: Generic struct walker

**Files:**
- Create: `config/walk.go`
- Test: `config/walk_test.go`

**Interfaces:**
- Consumes: `toScreamingSnake`, `joinPrefix` (Task 1); `implementsTextUnmarshaler` (Task 2).
- Produces: `walkFields(v reflect.Value, prefix string, visit func(field reflect.StructField, fv reflect.Value, autoName string) error) error` — used by Task 4 (`applyDefaults`) and Task 5 (`applyEnv`).

Behavior contract for `walkFields`:
- Iterates exported fields of the struct value `v` only (unexported fields are skipped).
- A field whose type is a struct (or pointer-to-struct) and does NOT implement `encoding.TextUnmarshaler` is treated as a nested container: nil pointers are allocated, then the walker recurses into it with `prefix` extended by `joinPrefix(prefix, toScreamingSnake(field.Name))`. This applies equally to named nested struct fields and embedded (anonymous) struct fields.
- Every other field (scalars, `[]string`, pointers-to-scalar, and anything implementing `encoding.TextUnmarshaler` such as `time.Time`) is a leaf: the walker calls `visit(field, fv, autoName)` where `autoName = joinPrefix(prefix, toScreamingSnake(field.Name))`.

- [ ] **Step 1: Write the failing test**

```go
// config/walk_test.go
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
		"NICKNAME",   // embedded struct field, no extra prefix segment beyond its own name... see note below
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
```

Note embedded in the test: `walkEmbedded`'s single field `Nickname` is expected as `NICKNAME` with no `WALK_EMBEDDED_` segment, because the embedded field's Go name in `reflect.StructField` for an anonymous field is the *type* name (`walkEmbedded`), and this plan's walker uses `field.Name` uniformly for the prefix segment — for an anonymous field `field.Name` is `"walkEmbedded"`, which would screaming-snake to `WALK_EMBEDDED`. To match common expectations that embedding doesn't add a synthetic prefix, `walkFields` must special-case `field.Anonymous`: pass through `prefix` unchanged (do not extend it) when recursing into an anonymous field, while still extending `prefix` normally for named nested struct fields like `Inner` and `Ptr`.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./config/... -run TestWalkFields_LeafNamesAndRecursion -v`
Expected: FAIL — `walkFields` is undefined.

- [ ] **Step 3: Write minimal implementation**

```go
// config/walk.go
package config

import "reflect"

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
		if !field.IsExported() {
			continue
		}
		fv := v.Field(i)

		if !isLeafType(field.Type) {
			childPrefix := prefix
			if !field.Anonymous {
				childPrefix = joinPrefix(prefix, toScreamingSnake(field.Name))
			}

			target := fv
			if target.Kind() == reflect.Ptr {
				if target.IsNil() {
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./config/... -run TestWalkFields_LeafNamesAndRecursion -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add config/walk.go config/walk_test.go
git commit -m "feat(config): add generic struct field walker"
```

---

### Task 4: Apply `default` tags

**Files:**
- Create: `config/defaults.go`
- Test: `config/defaults_test.go`

**Interfaces:**
- Consumes: `walkFields` (Task 3), `setFieldFromString` (Task 2).
- Produces: `applyDefaults(v reflect.Value) error` — called by `Load` (Task 6) before file/env are applied.

- [ ] **Step 1: Write the failing test**

```go
// config/defaults_test.go
package config

import (
	"reflect"
	"testing"
)

type defaultsTarget struct {
	LogLevel string `default:"info"`
	Retries  int    `default:"3"`
	NoTag    string
	Nested   struct {
		Timeout int `default:"30"`
	}
}

func TestApplyDefaults(t *testing.T) {
	var target defaultsTarget
	if err := applyDefaults(reflect.ValueOf(&target).Elem()); err != nil {
		t.Fatalf("applyDefaults: %v", err)
	}

	if target.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want info", target.LogLevel)
	}
	if target.Retries != 3 {
		t.Errorf("Retries = %d, want 3", target.Retries)
	}
	if target.NoTag != "" {
		t.Errorf("NoTag = %q, want empty (no default tag)", target.NoTag)
	}
	if target.Nested.Timeout != 30 {
		t.Errorf("Nested.Timeout = %d, want 30", target.Nested.Timeout)
	}
}

func TestApplyDefaults_BadValue(t *testing.T) {
	var target struct {
		Retries int `default:"not-a-number"`
	}
	if err := applyDefaults(reflect.ValueOf(&target).Elem()); err == nil {
		t.Fatal("expected error for unparsable default value, got nil")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./config/... -run TestApplyDefaults -v`
Expected: FAIL — `applyDefaults` is undefined.

- [ ] **Step 3: Write minimal implementation**

```go
// config/defaults.go
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./config/... -run TestApplyDefaults -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add config/defaults.go config/defaults_test.go
git commit -m "feat(config): apply default tag values"
```

---

### Task 5: Apply environment variable overrides

**Files:**
- Create: `config/env.go`
- Test: `config/env_test.go`

**Interfaces:**
- Consumes: `walkFields` (Task 3), `setFieldFromString` (Task 2).
- Produces: `applyEnv(v reflect.Value, globalPrefix string) error` — called by `Load` (Task 6) after file loading.

- [ ] **Step 1: Write the failing test**

```go
// config/env_test.go
package config

import (
	"reflect"
	"testing"
)

type envDB struct {
	Host string
	Port int `env:"PGPORT"`
}

type envTarget struct {
	LogLevel string `default:"info"`
	DB       envDB
}

func TestApplyEnv_ExplicitAndAutoDerived(t *testing.T) {
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("DB_HOST", "db.internal")
	t.Setenv("PGPORT", "6543")

	var target envTarget
	if err := applyEnv(reflect.ValueOf(&target).Elem(), ""); err != nil {
		t.Fatalf("applyEnv: %v", err)
	}

	if target.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want debug", target.LogLevel)
	}
	if target.DB.Host != "db.internal" {
		t.Errorf("DB.Host = %q, want db.internal", target.DB.Host)
	}
	if target.DB.Port != 6543 {
		t.Errorf("DB.Port = %d, want 6543 (from PGPORT, not DB_PORT)", target.DB.Port)
	}
}

func TestApplyEnv_GlobalPrefixAppliesOnlyToAutoDerived(t *testing.T) {
	t.Setenv("APP_LOG_LEVEL", "warn")
	t.Setenv("PGPORT", "5432")

	var target envTarget
	if err := applyEnv(reflect.ValueOf(&target).Elem(), "APP"); err != nil {
		t.Fatalf("applyEnv: %v", err)
	}

	if target.LogLevel != "warn" {
		t.Errorf("LogLevel = %q, want warn (from APP_LOG_LEVEL)", target.LogLevel)
	}
	if target.DB.Port != 5432 {
		t.Errorf("DB.Port = %d, want 5432 (explicit env tag ignores prefix)", target.DB.Port)
	}
}

func TestApplyEnv_UnsetVarsLeaveFieldUntouched(t *testing.T) {
	target := envTarget{LogLevel: "info"}
	if err := applyEnv(reflect.ValueOf(&target).Elem(), ""); err != nil {
		t.Fatalf("applyEnv: %v", err)
	}
	if target.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want unchanged info", target.LogLevel)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./config/... -run TestApplyEnv -v`
Expected: FAIL — `applyEnv` is undefined.

- [ ] **Step 3: Write minimal implementation**

```go
// config/env.go
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./config/... -run TestApplyEnv -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add config/env.go config/env_test.go
git commit -m "feat(config): apply environment variable overrides"
```

---

### Task 6: `Load`, file parsing, options, and errors

**Files:**
- Create: `config/errors.go`
- Create: `config/options.go`
- Create: `config/config.go`
- Test: `config/config_test.go`

**Interfaces:**
- Consumes: `applyDefaults` (Task 4), `applyEnv` (Task 5).
- Produces: `Load(path string, cfg any, opts ...Option) error`, `Option`, `WithEnvPrefix(prefix string) Option`, `ErrUnsupportedExtension` — the full public API of the package.

- [ ] **Step 1: Add the yaml dependency**

Run:
```bash
go get sigs.k8s.io/yaml
```

- [ ] **Step 2: Write the failing tests**

```go
// config/config_test.go
package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type dbConfig struct {
	Host string `json:"host"`
	Port int    `json:"port" env:"PGPORT"`
}

type appConfig struct {
	LogLevel string        `json:"log_level" default:"info"`
	Timeout  time.Duration `json:"timeout" default:"5s"`
	DB       dbConfig      `json:"db"`
}

func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}
	return path
}

func TestLoad_JSONFile(t *testing.T) {
	path := writeTempFile(t, "config.json", `{"log_level":"warn","db":{"host":"h1","port":1111}}`)

	var cfg appConfig
	if err := Load(path, &cfg); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.LogLevel != "warn" || cfg.DB.Host != "h1" || cfg.DB.Port != 1111 {
		t.Errorf("cfg = %+v", cfg)
	}
	if cfg.Timeout != 5*time.Second {
		t.Errorf("Timeout = %v, want default 5s", cfg.Timeout)
	}
}

func TestLoad_YAMLFile(t *testing.T) {
	path := writeTempFile(t, "config.yaml", "log_level: error\ndb:\n  host: h2\n  port: 2222\n")

	var cfg appConfig
	if err := Load(path, &cfg); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.LogLevel != "error" || cfg.DB.Host != "h2" || cfg.DB.Port != 2222 {
		t.Errorf("cfg = %+v", cfg)
	}
}

func TestLoad_EnvOverridesFile(t *testing.T) {
	path := writeTempFile(t, "config.json", `{"log_level":"warn","db":{"host":"h1","port":1111}}`)
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("PGPORT", "9999")

	var cfg appConfig
	if err := Load(path, &cfg); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want debug (env should win over file)", cfg.LogLevel)
	}
	if cfg.DB.Port != 9999 {
		t.Errorf("DB.Port = %d, want 9999 (env should win over file)", cfg.DB.Port)
	}
	if cfg.DB.Host != "h1" {
		t.Errorf("DB.Host = %q, want h1 (no env override, file should win over default)", cfg.DB.Host)
	}
}

func TestLoad_EnvOnly_NoPath(t *testing.T) {
	t.Setenv("LOG_LEVEL", "trace")
	t.Setenv("DB_HOST", "envhost")
	t.Setenv("PGPORT", "1")

	var cfg appConfig
	if err := Load("", &cfg); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.LogLevel != "trace" || cfg.DB.Host != "envhost" || cfg.DB.Port != 1 {
		t.Errorf("cfg = %+v", cfg)
	}
	if cfg.Timeout != 5*time.Second {
		t.Errorf("Timeout = %v, want default 5s", cfg.Timeout)
	}
}

func TestLoad_WithEnvPrefix(t *testing.T) {
	t.Setenv("APP_LOG_LEVEL", "warn")
	t.Setenv("PGPORT", "4242")

	var cfg appConfig
	if err := Load("", &cfg, WithEnvPrefix("APP")); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.LogLevel != "warn" {
		t.Errorf("LogLevel = %q, want warn", cfg.LogLevel)
	}
	if cfg.DB.Port != 4242 {
		t.Errorf("DB.Port = %d, want 4242 (explicit env tag ignores prefix)", cfg.DB.Port)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	var cfg appConfig
	err := Load(filepath.Join(t.TempDir(), "missing.json"), &cfg)
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestLoad_UnsupportedExtension(t *testing.T) {
	path := writeTempFile(t, "config.toml", "log_level = \"warn\"")

	var cfg appConfig
	err := Load(path, &cfg)
	if !errors.Is(err, ErrUnsupportedExtension) {
		t.Fatalf("Load error = %v, want ErrUnsupportedExtension", err)
	}
}

func TestLoad_RequiresPointerToStruct(t *testing.T) {
	var cfg appConfig
	if err := Load("", cfg); err == nil {
		t.Fatal("expected error when cfg is not a pointer, got nil")
	}
	if err := Load("", nil); err == nil {
		t.Fatal("expected error when cfg is nil, got nil")
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./config/... -run TestLoad -v`
Expected: FAIL — `Load`, `WithEnvPrefix`, `ErrUnsupportedExtension` are undefined.

- [ ] **Step 4: Write minimal implementation**

```go
// config/errors.go
package config

import "errors"

// ErrUnsupportedExtension is returned by Load when the file path's
// extension is not one of .json, .yaml, or .yml.
var ErrUnsupportedExtension = errors.New("config: unsupported file extension")
```

```go
// config/options.go
package config

type options struct {
	envPrefix string
}

// Option configures Load.
type Option func(*options)

// WithEnvPrefix prepends prefix + "_" to every auto-derived (non-explicit
// `env` tag) environment variable name Load looks up.
func WithEnvPrefix(prefix string) Option {
	return func(o *options) {
		o.envPrefix = prefix
	}
}
```

```go
// config/config.go
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"sigs.k8s.io/yaml"
)

// Load populates cfg (a non-nil pointer to a struct) from the file at path
// (if path != ""), then applies `default` tag values and environment
// variable overrides. Precedence, high to low: env var, file value,
// default tag, zero value. Format is chosen from the file extension:
// .json, .yaml, or .yml.
func Load(path string, cfg any, opts ...Option) error {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}

	rv := reflect.ValueOf(cfg)
	if rv.Kind() != reflect.Ptr || rv.IsNil() || rv.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("config: Load requires a non-nil pointer to a struct, got %T", cfg)
	}
	elem := rv.Elem()

	if err := applyDefaults(elem); err != nil {
		return err
	}

	if path != "" {
		if err := loadFile(path, cfg); err != nil {
			return err
		}
	}

	if err := applyEnv(elem, o.envPrefix); err != nil {
		return err
	}

	return nil
}

func loadFile(path string, cfg any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("config: file %s does not exist: %w", path, err)
		}
		return fmt.Errorf("config: reading file %s: %w", path, err)
	}

	switch ext := strings.ToLower(filepath.Ext(path)); ext {
	case ".json":
		if err := json.Unmarshal(data, cfg); err != nil {
			return fmt.Errorf("config: parsing json file %s: %w", path, err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return fmt.Errorf("config: parsing yaml file %s: %w", path, err)
		}
	default:
		return fmt.Errorf("config: file %s: %w", path, ErrUnsupportedExtension)
	}
	return nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./config/... -v`
Expected: PASS for every test in the package (Tasks 1-6 combined).

- [ ] **Step 6: Run go vet and go build across the module**

Run: `go build ./... && go vet ./...`
Expected: no errors.

- [ ] **Step 7: Commit**

```bash
git add go.mod go.sum config/errors.go config/options.go config/config.go config/config_test.go
git commit -m "feat(config): add Load with file parsing, options, and errors"
```

---

### Task 7: README with a compiled, runnable example

**Files:**
- Create: `config/README.md`
- Create: `config/example_test.go`

**Interfaces:**
- Consumes: `Load`, `WithEnvPrefix` (Task 6). No new production code.

This task's example must actually compile and run as part of `go test`, so the README can't drift from the real API: write it as a Go `Example` function first, confirm it passes, then base the README's code samples on that verified function.

- [ ] **Step 1: Write the example test**

```go
// config/example_test.go
package config_test

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/eftakhairul/lib-go/config"
)

type ServerConfig struct {
	Host string `json:"host" default:"0.0.0.0"`
	Port int    `json:"port" default:"8080" env:"SERVER_PORT"`
}

type AppConfig struct {
	LogLevel string       `json:"log_level" default:"info"`
	Server   ServerConfig `json:"server"`
}

func Example() {
	dir, err := os.MkdirTemp("", "config-example")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("log_level: warn\nserver:\n  host: 127.0.0.1\n"), 0o600); err != nil {
		panic(err)
	}

	os.Setenv("SERVER_PORT", "9090")
	defer os.Unsetenv("SERVER_PORT")

	var cfg AppConfig
	if err := config.Load(path, &cfg); err != nil {
		panic(err)
	}

	fmt.Println(cfg.LogLevel)
	fmt.Println(cfg.Server.Host)
	fmt.Println(cfg.Server.Port)
	// Output:
	// warn
	// 127.0.0.1
	// 9090
}
```

- [ ] **Step 2: Run the example to verify it passes**

Run: `go test ./config/... -run Example -v`
Expected: PASS, with output matching the `// Output:` block exactly.

- [ ] **Step 3: Write the README**

Create `config/README.md` with exactly this content (note: this is the
literal file content — write real triple-backtick fences in the file
itself, not escaped ones):

~~~
# config

Load a Go struct from a JSON or YAML file and/or environment variables.
Environment variables always win over the file, which wins over `default`
tag values.

## Install

```bash
go get github.com/eftakhairul/lib-go/config
```

## Usage

```go
package main

import (
	"fmt"
	"log"

	"github.com/eftakhairul/lib-go/config"
)

type ServerConfig struct {
	Host string `json:"host" default:"0.0.0.0"`
	Port int    `json:"port" default:"8080" env:"SERVER_PORT"`
}

type AppConfig struct {
	LogLevel string       `json:"log_level" default:"info"`
	Server   ServerConfig `json:"server"`
}

func main() {
	var cfg AppConfig
	if err := config.Load("config.yaml", &cfg); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%+v\n", cfg)
}
```

Given `config.yaml`:

```yaml
log_level: warn
server:
  host: 127.0.0.1
```

and `SERVER_PORT=9090` in the environment, `cfg` ends up as
`LogLevel: "warn"`, `Server.Host: "127.0.0.1"`, `Server.Port: 9090` — the
file supplies `log_level` and `server.host`, the `SERVER_PORT` env var
overrides `server.port`, and `Server.Host`'s `default:"0.0.0.0"` never
applies because the file already set it.

## Tags

- `json:"name"` — field name in both JSON and YAML files (YAML is
  converted to JSON internally, so one tag drives both formats).
- `env:"NAME"` — explicit environment variable name, used verbatim. Without
  this tag, the name is auto-derived from the Go field name
  (`LogLevel` -> `LOG_LEVEL`, `HTTPPort` -> `HTTP_PORT`), and nested struct
  fields are joined as `PARENT_CHILD` (e.g. a `Host` field inside a `DB`
  field auto-derives to `DB_HOST`). An explicit `env` tag is always used
  as-is, ignoring any parent prefix.
- `default:"value"` — fallback used only when the field is set by neither
  the file nor an environment variable.

## Precedence

Environment variable > file value > `default` tag > zero value.

## Loading from environment only

Pass an empty path to skip file loading entirely:

```go
err := config.Load("", &cfg)
```

## Global env var prefix

```go
err := config.Load("config.yaml", &cfg, config.WithEnvPrefix("APP"))
```

This prepends `APP_` to every auto-derived environment variable name
(`LOG_LEVEL` -> `APP_LOG_LEVEL`). It has no effect on fields with an
explicit `env` tag, since those are already fully specified.

## Supported field types

`string`, `bool`, all signed/unsigned integer sizes, `float32`/`float64`,
`time.Duration` (e.g. `"1500ms"`), `[]string` (comma-separated), pointers
to any of the above, and any type implementing `encoding.TextUnmarshaler`
(e.g. `time.Time`, parsed as RFC 3339).
~~~

- [ ] **Step 4: Verify the README's install path and package name match go.mod**

Run: `head -1 go.mod`
Expected: `module github.com/eftakhairul/lib-go` — confirms the README's `go get github.com/eftakhairul/lib-go/config` line is correct.

- [ ] **Step 5: Run the full test suite once more**

Run: `go build ./... && go vet ./... && go test ./... -v`
Expected: all packages build, vet is clean, all tests (including `Example`) pass.

- [ ] **Step 6: Commit**

```bash
git add config/README.md config/example_test.go
git commit -m "docs(config): add README with runnable example"
```
