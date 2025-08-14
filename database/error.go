package database

import (
	"errors"
)

var (
	ErrFailedToExecuteQuery = errors.New("failed to execute query")
	ErrFailedToCommitTX     = errors.New("failed to commit tx")
	ErrFailedToBeginTX      = errors.New("failed to begin tx")
	ErrFailedToExecuteTX    = errors.New("failed to execute tx")
	ErrNotFound             = errors.New("not found")
	ErrInvalidArgument      = errors.New("invalid argument")
)
