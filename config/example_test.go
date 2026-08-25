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
