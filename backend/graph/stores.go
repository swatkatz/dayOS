package graph

import (
	"context"

	"dayos/db"

	"github.com/jackc/pgx/v5/pgtype"
)

type RoutineStore interface {
	UpsertRoutine(ctx context.Context, arg db.UpsertRoutineParams) (db.Routine, error)
	GetRoutine(ctx context.Context, id pgtype.UUID) (db.Routine, error)
	ListRoutines(ctx context.Context, activeOnly *bool) ([]db.Routine, error)
	DeleteRoutine(ctx context.Context, id pgtype.UUID) error
	ListRoutinesForDay(ctx context.Context, dayOfWeek int32) ([]db.Routine, error)
}
