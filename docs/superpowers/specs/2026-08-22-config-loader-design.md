# config: struct-based JSON/YAML/env loader — Design

Status: Approved for planning
Date: 2026-08-22

## Purpose

A public Go library package, `github.com/eftakhairul/lib-go/config`, that
populates a caller-provided struct from a JSON or YAML file and/or from
environment variables. Environment variables take precedence over file
values, which take precedence over `default` tag values.

## Repo/module setup

This repo currently has no `go.mod` (the existing `logger` and `secure`
packages are not modules). This work adds one:

- `module github.com/eftakhairul/lib-go`
- `go 1.22`
- dependency: `sigs.k8s.io/yaml`

New package lives at `config/` alongside `logger/` and `secure/`.

## Tag scheme

A single tag set drives both file and env parsing:

| Tag | Purpose |
|---|---|
| `json:"..."` | Field name for both JSON and YAML files (YAML is converted to JSON internally via `sigs.k8s.io/yaml`, so one tag set covers both formats). |
| `env:"NAME"` | Explicit environment variable name. When present, always used as-is (no prefix, no auto-derivation). |
| `default:"..."` | Fallback value, applied before file/env are considered. Parsed with the same type coercion as env values. |

If a field has no `env` tag, its env var name is auto-derived from the Go
field name by converting CamelCase to `SCREAMING_SNAKE_CASE`
(`LogLevel` → `LOG_LEVEL`, `HTTPPort` → `HTTP_PORT`).

Nested and embedded structs recurse. A nested field's auto-derived name is
`PARENT_CHILD` (parent field name + child field name, both
screaming-snake, joined with `_`). An explicit `env` tag on a nested field
overrides the whole derived path — it is used verbatim, not appended to
the parent prefix.

```go
type DBConfig struct {
    Host string `json:"host"`            // -> DB_HOST
    Port int    `json:"port" env:"PGPORT"` // -> PGPORT (verbatim, no DB_ prefix)
}

type Config struct {
    DB       DBConfig `json:"db"`
    LogLevel string   `json:"log_level" default:"info"` // -> LOG_LEVEL
}
```

An optional `WithEnvPrefix("APP")` load option prepends `APP_` to every
*auto-derived* name (`LOG_LEVEL` → `APP_LOG_LEVEL`). It does not affect
fields with an explicit `env` tag, since those are already fully
specified by the caller.

## Precedence semantics

High to low: **env var → file value → `default` tag → zero value.**

Implementation order (this order is what makes the precedence correct,
not just documentation):

1. Walk the struct and apply `default:"..."` tag values first, using the
   same type-coercion logic as env parsing.
2. If `path != ""`, unmarshal the file (JSON directly, or YAML via
   `sigs.k8s.io/yaml`, chosen by file extension) into the struct. Because
   `encoding/json` only writes fields whose keys are present in the
   source document, defaults on fields absent from the file survive this
   step untouched.
3. Walk the struct again; for each field whose resolved env var name is
   present via `os.LookupEnv` (presence, not non-empty check — an
   explicitly set empty string still counts as "set"), parse and
   overwrite the field.

This ordering sidesteps the classic "is zero-value unset-or-explicit"
ambiguity: defaults are seeded before the file is even read, and env
overrides are gated on `LookupEnv` presence rather than on the field's
current value.

## API surface

```go
package config

// Load populates cfg (a pointer to a struct) from the file at path (if
// path != ""), then applies environment variable overrides. Format is
// chosen from the file extension: .json, .yaml, .yml.
func Load(path string, cfg any, opts ...Option) error

// Option configures Load.
type Option func(*options)

// WithEnvPrefix prepends prefix + "_" to every auto-derived
// (non-explicit-tag) environment variable name.
func WithEnvPrefix(prefix string) Option
```

Behavior notes:
- `path == ""`: skip file loading; env vars and defaults still apply.
- `path != ""` and the file does not exist, or has an unsupported
  extension: `Load` returns an error (no silent skip).
- `cfg` must be a non-nil pointer to a struct; anything else is a
  programmer error and returns an error rather than panicking.

## Type coercion

Supported field kinds, applied identically whether the value comes from
an env var string or a `default` tag string:

- `string`, `bool`
- all signed/unsigned int sizes, `float32`/`float64`
- `time.Duration` (via `time.ParseDuration`)
- `[]string` (comma-split, e.g. `"a,b,c"` → `["a","b","c"]`)
- any type implementing `encoding.TextUnmarshaler` (covers `time.Time`
  and custom enum-like types) — checked before falling back to the
  built-in kinds above, so callers get an escape hatch without the
  library needing bespoke cases for every stdlib type.
- pointer fields: allocated on demand (`*int`, `*SubStruct`, etc.) before
  the value is written or recursion continues.

An env var or default string that fails to parse into the field's type
returns a wrapped error naming the field and the offending value.

## Errors

Plain wrapped errors via `fmt.Errorf("...: %w", err)`, plus sentinels for
programmatic checks:

- `ErrUnsupportedExtension` — file path has neither `.json`, `.yaml`, nor
  `.yml`.
- a wrapped `os.IsNotExist` condition for a missing file at a non-empty
  `path`.

No dependency on the existing `secure` package's `SafeError` — that type
models externally-safe API error responses, a different concern from
config-loading/startup errors, which are internal-only.

## Testing plan

Table-driven tests in `config/config_test.go` (and possibly a
`config/internal` helper for reflection walking, if the implementation
benefits from separating the tag-walk logic from `Load` itself):

- File-only load: valid `.json`, valid `.yaml`/`.yml`.
- Env-only load (`path == ""`).
- Precedence: file value overridden by env var; default overridden by
  file; default overridden by env directly (file absent).
- Nested/embedded struct env name derivation, including explicit `env`
  tag override on a nested field.
- `WithEnvPrefix` applied to auto-derived names, not applied to explicit
  `env` tags.
- `default` tag coercion across supported types.
- `encoding.TextUnmarshaler` field (e.g. a custom type or `time.Time`).
- Error cases: missing file, unsupported extension, non-pointer/non-struct
  `cfg`, unparsable env value for a typed field.

## Out of scope (v1)

- `required` tag / validation that a field ended up set.
- Config file watching / reload.
- Sources other than a single file + env (e.g. remote config, multiple
  layered files).
