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
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json", ".yaml", ".yml":
	default:
		return fmt.Errorf("config: file %s: %w", path, ErrUnsupportedExtension)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("config: file %s does not exist: %w", path, err)
		}
		return fmt.Errorf("config: reading file %s: %w", path, err)
	}

	switch ext {
	case ".json":
		if err := json.Unmarshal(data, cfg); err != nil {
			return fmt.Errorf("config: parsing json file %s: %w", path, err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return fmt.Errorf("config: parsing yaml file %s: %w", path, err)
		}
	}
	return nil
}
