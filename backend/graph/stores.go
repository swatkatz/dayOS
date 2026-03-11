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

type ContextStore interface {
	ListContextEntries(ctx context.Context, category *string) ([]db.ContextEntry, error)
	GetContextEntry(ctx context.Context, id pgtype.UUID) (db.ContextEntry, error)
	UpsertContextEntry(ctx context.Context, arg db.UpsertContextEntryParams) (db.ContextEntry, error)
	ToggleContextEntry(ctx context.Context, arg db.ToggleContextEntryParams) (db.ContextEntry, error)
	DeleteContextEntry(ctx context.Context, id pgtype.UUID) error
}

type TaskStore interface {
	CreateTask(ctx context.Context, arg db.CreateTaskParams) (db.Task, error)
	GetTask(ctx context.Context, id pgtype.UUID) (db.Task, error)
	ListTasks(ctx context.Context, arg db.ListTasksParams) ([]db.Task, error)
	ListSubtasks(ctx context.Context, parentID pgtype.UUID) ([]db.Task, error)
	UpdateTask(ctx context.Context, arg db.UpdateTaskParams) (db.Task, error)
	CompleteTask(ctx context.Context, id pgtype.UUID) (db.Task, error)
	UncompleteTask(ctx context.Context, id pgtype.UUID) (db.Task, error)
	DeleteTask(ctx context.Context, id pgtype.UUID) error
	CountIncompleteSubtasks(ctx context.Context, parentID pgtype.UUID) (int64, error)
}
