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
