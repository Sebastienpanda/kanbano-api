package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// querier est satisfait par *pgxpool.Pool comme par pgx.Tx, ce qui permet
// d'utiliser les helpers aussi bien hors transaction que dans une transaction.
type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// queryStruct exécute une requête et mappe l'unique ligne retournée vers T
// (via RowToStructByName). Il factorise le pattern Query + CollectOneRow.
func queryStruct[T any](ctx context.Context, q querier, sql string, args ...any) (T, error) {
	rows, err := q.Query(ctx, sql, args...)
	if err != nil {
		var zero T
		return zero, err
	}
	return pgx.CollectOneRow(rows, pgx.RowToStructByName[T])
}
