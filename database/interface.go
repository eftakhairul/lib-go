package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Connection = pgxpool.Conn

// GetSQLCQueriesObjectFunc returns an instance of the sqlc.Queries object
// This function acts as a wrapper so that we can use the generic type Q as the output
type GetSQLCQueriesObjectFunc[Q interface{}] func(ctx context.Context, conn *Connection) *Q

// QueryFunc is a function type that represents a query operation
type QueryFunc[Q interface{}] func(ctx context.Context, conn *Connection, queries *Q) error

type Config interface {
	PostgresDBHost() string
	PostgresDBPort() string
	PostgresDBUser() string
	PostgresDBName() string
	PostgresDBPassword() string
}
