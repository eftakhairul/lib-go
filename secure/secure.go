package secure

import "fmt"

type SafeError struct {

	// machine-readable code for clients (e.g., "RESOURCE_NOT_FOUND")
	Code string

	// human-readable message safe for public consumption
	UserMsg string

	// the raw, upstream error (DO NOT expose this via API)
	Internal error

	// context map for structured logging (sanitized)
	Metadata map[string]string
}

// Error satisfies the stdlib interface.
func (e *SafeError) Error() string {

	return e.UserMsg
}

// LogString returns the detailed string for your SRE team.
func (e *SafeError) LogString() string {

	return fmt.Sprintf("Code: %s | Msg: %s | Cause: %v | Meta: %v",
		e.Code,
		e.UserMsg,
		e.Internal,
		e.Metadata)
}
