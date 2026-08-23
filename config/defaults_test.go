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
