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
