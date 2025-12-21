package database

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Database struct {
	p *pgxpool.Pool
}

func (db *Database) Exec(ctx context.Context, sql string, arguments ...any) (commandTag pgconn.CommandTag, err error) {
	return db.loadDB(ctx).Exec(ctx, sql, arguments...)
}

func (db *Database) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return db.loadDB(ctx).Query(ctx, sql, args...)
}

func (db *Database) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return db.loadDB(ctx).QueryRow(ctx, sql, args...)
}

func (db *Database) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return db.loadDB(ctx).CopyFrom(ctx, tableName, columnNames, rowSrc)
}

func NewDatabase(pool *pgxpool.Pool) *Database {
	return &Database{p: pool}
}
