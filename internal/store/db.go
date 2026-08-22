// Package store is the SQLite persistence layer for the reliability engine.
// It owns the schema, the connection and transaction primitives. Derived
// results (cut sets, probabilities, RPN, fits) are persisted but recomputable
// from the authoritative inputs (trees, diagrams, event logs); ReconcileAll
// re-Solves every analysis on startup so a crash converges to the same state.
//
// Open uses _txlock=immediate so every BEGIN is BEGIN IMMEDIATE (serialized
// writers). SetMaxOpenConns(1) avoids "database is locked" from interleaved
// write transactions on extra connections. With a single pooled connection,
// any read inside an InTx transaction MUST go through the *sql.Tx (passed as
// DBTX) — reaching for s.db while a tx holds the connection deadlocks.
package store

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite" // pure-Go SQLite driver; CGO_ENABLED=0 compatible
)

// Store wraps a *sql.DB connection to the reliability database.
type Store struct {
	db *sql.DB
}

// Open creates or opens the SQLite file at path, applies the schema and tunes
// pragmatic options for durability and concurrency (WAL, busy_timeout,
// synchronous=NORMAL, foreign_keys=ON, _txlock=immediate).
func Open(path string) (*Store, error) {
	dsn := fmt.Sprintf(
		"file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(ON)&_txlock=immediate",
		path,
	)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", path, err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	return &Store{db: db}, nil
}

// Close releases the database handle.
func (s *Store) Close() error { return s.db.Close() }

// DBTX is the union of *sql.DB and *sql.Tx query/exec methods. Every store
// method that may run inside an InTx transaction takes this interface so the
// caller can pass either the pooled connection's tx (inside a tx) or the bare
// store (outside a tx). This is critical with SetMaxOpenConns(1): a method
// that reaches for s.db while a transaction holds the single connection
// deadlocks.
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// ExecContext / QueryContext / QueryRowContext forward to the pooled connection
// so *Store satisfies DBTX for outside-tx read paths. Never call these from
// inside an InTx transaction (deadlock with SetMaxOpenConns(1)).
func (s *Store) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return s.db.ExecContext(ctx, query, args...)
}
func (s *Store) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return s.db.QueryContext(ctx, query, args...)
}
func (s *Store) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return s.db.QueryRowContext(ctx, query, args...)
}

// BeginTx starts an IMMEDIATE transaction. Callers must Commit or Rollback.
func (s *Store) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return s.db.BeginTx(ctx, nil)
}

// InTx runs fn inside a single IMMEDIATE transaction. On error the transaction
// is rolled back; on success it is committed.
func (s *Store) InTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := s.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// Count runs a query returning a single INTEGER.
func (s *Store) Count(ctx context.Context, query string, args ...any) (int, error) {
	var n int
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}
