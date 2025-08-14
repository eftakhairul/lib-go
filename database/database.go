package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TODO: this presents a good case as a code-libraries component
// the pieces that need to be hooked-in from the consumer side are
// - NewDatabase() since it defines the concrete SQLC types which only exist consumer side
// - ErrNotFound, since error types are typically defined in the consumer models package

type Impl[Q interface{}] struct {
	connectionPool *pgxpool.Pool
	getQueries     GetSQLCQueriesObjectFunc[Q]
}

func New[Q interface{}](
	ctx context.Context,
	dbConfig Config,
	getQueries GetSQLCQueriesObjectFunc[Q],
) (*Impl[Q], error) {
	connectionPool, err := pgxpool.NewWithConfig(
		ctx,
		dbConfig,
	)
	if err != nil {
		return nil, errors.New("failed to create connection pool against database")
	}

	return &Impl[Q]{
		connectionPool: connectionPool,
		getQueries:     getQueries,
	}, nil
}

func (s *Impl[Q]) Close() {
	if s.connectionPool != nil {
		s.connectionPool.Close()
	}
}

func (s *Impl[Q]) Raw(ctx context.Context, f QueryFunc[Q]) error {
	// acquire a connection from the pool
	conn, err := s.connectionPool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection from pool")
	}
	defer conn.Release()

	// get the queries object for SQLC
	queries := s.getQueries(ctx, conn)

	// execute the query
	err = f(ctx, conn, queries)
	if errors.Is(err, pgx.ErrNoRows) {
		// return our own error type to encapsulate the driver-level error
		err = errors.Join(ErrNotFound)
	}
	if err != nil {
		return ErrFailedToExecuteQuery
	}

	return nil
}

func (s *Impl[Q]) TX(ctx context.Context, f QueryFunc[Q]) error {

	return s.Raw(ctx, func(ctx context.Context, conn *Connection, queries *Q) error {
		// begin the transaction
		tx, err := conn.Begin(ctx)
		if err != nil {
			return ErrFailedToBeginTX
		}

		// this will ensure that the transaction is rolled back if f() panics
		// if the TX has already been committed, this will not do anything
		defer func() {
			handleRollback(ctx, tx)
		}()

		// execute the inner query
		err = f(ctx, conn, queries)
		if err != nil {
			handleRollback(ctx, tx)

			// return the query error
			return ErrFailedToExecuteTX
		}

		// commit the transaction, since there's no errors
		err = tx.Commit(ctx)
		if err != nil {
			return ErrFailedToCommitTX
		}

		return nil
	})
}

func handleRollback(ctx context.Context, tx pgx.Tx) {
	rollbackErr := tx.Rollback(ctx)
	if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
		return
	}
}
