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
