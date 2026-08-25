package config

import "errors"

// ErrUnsupportedExtension is returned by Load when the file path's
// extension is not one of .json, .yaml, or .yml.
var ErrUnsupportedExtension = errors.New("config: unsupported file extension")
