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
