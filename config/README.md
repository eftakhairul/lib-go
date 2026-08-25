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
