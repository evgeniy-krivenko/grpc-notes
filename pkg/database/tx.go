package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Tx interface {
	Exec(ctx context.Context, sql string, arguments ...any) (commandTag pgconn.CommandTag, err error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error)
}

func (db *Database) RunInTx(ctx context.Context, f func(context.Context) error) error {
	tx := TxFromContext(ctx)
	if tx == nil {
		newTx, err := db.p.Begin(ctx)
		if err != nil {
			return err
		}

		tx = newTx
		ctx = NewTxContext(ctx, tx)
	}
	defer func() {
		if v := recover(); v != nil {
			if err := tx.Rollback(ctx); err != nil {
				v = fmt.Sprintf("%v: rolling back transaction: %v", v, err)
			}
			panic(v)
		}
	}()

	if err := f(ctx); err != nil {
		if rerr := tx.Rollback(ctx); rerr != nil {
			err = fmt.Errorf("%w: rolling back transaction: %v", err, rerr)
		}
		return err
	}

	return tx.Commit(ctx)
}

type txCtxKey struct{}

func TxFromContext(ctx context.Context) pgx.Tx {
	tx, _ := ctx.Value(txCtxKey{}).(pgx.Tx)

	return tx
}

func NewTxContext(parent context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(parent, txCtxKey{}, tx)
}

func (db *Database) loadDB(ctx context.Context) Tx {
	tx := TxFromContext(ctx)
	if tx != nil {
		return tx
	}

	return db.p
}
