package sqlc

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBTX is satisfied by *pgxpool.Pool and pgx.Tx.
type DBTX interface {
	Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error)
	Query(context.Context, string, ...interface{}) (pgx.Rows, error)
	QueryRow(context.Context, string, ...interface{}) pgx.Row
}

// Queries wraps a database connection.
type Queries struct {
	db DBTX
}

// New creates a Queries instance. Accepts *pgxpool.Pool or pgx.Tx.
func New(db interface{}) *Queries {
	switch d := db.(type) {
	case *pgxpool.Pool:
		return &Queries{db: d}
	case pgx.Tx:
		return &Queries{db: d}
	default:
		panic("sqlc.New: unsupported db type")
	}
}
