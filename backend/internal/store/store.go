// Package store owns every database statement: pool setup, the
// transaction helper services use so a mutation and its audit row commit
// together (RNF05), and the repositories per entity. No business rules
// live here — SQL only. Constraint violations surface as plain error
// values (sentinels for the named constraints, ConstraintError for the
// rest) so no database driver type crosses into the service layer.
package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB is the executor subset both *pgxpool.Pool and pgx.Tx satisfy, so
// repository methods run unchanged against the pool or inside a
// transaction.
type DB interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Store executes statements against a DB: the pool for the root
// instance, the open transaction inside WithTx.
type Store struct {
	pool *pgxpool.Pool
	db   DB
}

// Open connects a pool to databaseURL (libpq URI) and verifies it answers.
// The session runs in UTC: storage and service-layer times are UTC, the
// timezone policy converts only at the UI edge
// (docs/spec/architecture.md, "Timezone policy").
func Open(ctx context.Context, databaseURL string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("store: parse config: %w", err)
	}
	cfg.ConnConfig.RuntimeParams["TimeZone"] = "UTC"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("store: connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("store: ping: %w", err)
	}
	return &Store{pool: pool, db: pool}, nil
}

// Close releases the pool.
func (s *Store) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}

// ErrNotFound reports a single-row lookup that matched nothing.
var ErrNotFound = errors.New("not found")

// Database-condition sentinels detected by constraint name
// (docs/spec/data-model.md names both constraints). The service layer
// aliases them into its operation error model.
var (
	ErrDriverDateConflict = errors.New("driver already has a route on this date")
	ErrDuplicateEmail     = errors.New("email already registered")
)

// ConstraintError reports a constraint violation without leaking driver
// types; the service layer maps it onto its error model.
type ConstraintError struct {
	Code       string // SQLSTATE, e.g. 23505
	Constraint string
	Table      string
}

func (e *ConstraintError) Error() string {
	return fmt.Sprintf("constraint violation %s (%s)", e.Constraint, e.Code)
}

// WithTx runs fn inside one transaction on the pool; the Store fn
// receives is bound to it. Whatever fn wrote commits only when fn
// returns nil — a mutation and its audit row land together or not at
// all. Transactions do not nest.
func (s *Store) WithTx(ctx context.Context, fn func(tx *Store) error) error {
	if s.pool == nil {
		return errors.New("store: WithTx inside a transaction is not supported")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return translate(err)
	}
	if err := fn(&Store{db: tx}); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return translate(err)
	}
	return nil
}

// WithSnapshot runs fn in one read-only REPEATABLE READ transaction, so
// several reads (a dashboard series and the parameter it was computed
// with) see the same database state.
func (s *Store) WithSnapshot(ctx context.Context, fn func(tx *Store) error) error {
	if s.pool == nil {
		return errors.New("store: WithSnapshot inside a transaction is not supported")
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return translate(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(&Store{db: tx}); err != nil {
		return err
	}
	return translate(tx.Commit(ctx))
}

// translate maps driver errors onto the store's error values; other
// errors pass through untouched.
func translate(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			switch pgErr.ConstraintName {
			case "route_driver_date_uniq":
				return ErrDriverDateConflict // RN05
			case "app_user_email_uniq":
				return ErrDuplicateEmail
			}
		case "23503", "23514":
			return &ConstraintError{
				Code:       pgErr.Code,
				Constraint: pgErr.ConstraintName,
				Table:      pgErr.TableName,
			}
		}
	}
	return err
}

// scanOne reports a single-row lookup result: no rows is ErrNotFound,
// other errors are translated.
func scanOne(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return translate(err)
}
