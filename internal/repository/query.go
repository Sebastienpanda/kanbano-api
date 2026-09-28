package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func queryStruct[T any](ctx context.Context, q querier, sql string, args ...any) (T, error) {
	rows, err := q.Query(ctx, sql, args...)
	if err != nil {
		var zero T
		return zero, err
	}
	return pgx.CollectOneRow(rows, pgx.RowToStructByName[T])
}

// uuidStrings converts ids for a $n::uuid[] parameter: with
// QueryExecModeSimpleProtocol, pgx cannot encode a []uuid.UUID (unknown
// OID), but it can encode a []string.
func uuidStrings(ids []uuid.UUID) []string {
	s := make([]string, len(ids))
	for i, id := range ids {
		s[i] = id.String()
	}
	return s
}
