package graph

import (
	"fmt"
	"strings"

	"dayos/db"
	"dayos/graph/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type taskConv struct{}

var taskConverter taskConv

func (taskConv) FromDB(t db.Task) *model.Task {
	out := &model.Task{
		ID:       uuid.UUID(t.ID.Bytes),
		Title:    t.Title,
		Category: model.Category(strings.ToUpper(t.Category)),
		Priority: model.Priority(strings.ToUpper(t.Priority)),
		Subtasks: []*model.Task{},
	}
	if t.ParentID.Valid {
		pid := uuid.UUID(t.ParentID.Bytes)
		out.ParentID = &pid
	}
	if t.EstimatedMinutes != nil {
		v := int(*t.EstimatedMinutes)
		out.EstimatedMinutes = &v
	}
	if t.ActualMinutes != nil {
		out.ActualMinutes = int(*t.ActualMinutes)
	}
	if t.DeadlineType != nil {
		dt := model.DeadlineType(strings.ToUpper(*t.DeadlineType))
		out.DeadlineType = &dt
	}
	if t.DeadlineDate.Valid {
		d := model.Date{Time: t.DeadlineDate.Time}
		out.DeadlineDate = &d
	}
	if t.DeadlineDays != nil {
		v := int(*t.DeadlineDays)
		out.DeadlineDays = &v
	}
	out.Notes = t.Notes
	if t.IsRoutine != nil {
		out.IsRoutine = *t.IsRoutine
	}
	if t.RoutineID.Valid {
		rid := uuid.UUID(t.RoutineID.Bytes)
		out.Routine = &model.Routine{ID: rid}
	}
	if t.TimesDeferred != nil {
		out.TimesDeferred = int(*t.TimesDeferred)
	}
	if t.IsCompleted != nil {
		out.IsCompleted = *t.IsCompleted
	}
	if t.CompletedAt.Valid {
		ca := model.DateTime{Time: t.CompletedAt.Time}
		out.CompletedAt = &ca
	}
	if t.CreatedAt.Valid {
		out.CreatedAt = model.DateTime{Time: t.CreatedAt.Time}
	}
	if t.UpdatedAt.Valid {
		out.UpdatedAt = model.DateTime{Time: t.UpdatedAt.Time}
	}
	return out
}

func (taskConv) ToCreateParams(input model.CreateTaskInput) db.CreateTaskParams {
	estMin := int32(60)
	if input.EstimatedMinutes != nil {
		estMin = int32(*input.EstimatedMinutes)
	}

	isRoutine := input.RoutineID != nil

	params := db.CreateTaskParams{
		Title:            input.Title,
		Category:         strings.ToLower(string(input.Category)),
		Priority:         strings.ToLower(string(input.Priority)),
		EstimatedMinutes: &estMin,
		IsRoutine:        &isRoutine,
	}

	if input.ParentID != nil {
		params.ParentID = uuidToPgtype(*input.ParentID)
	}
	if input.DeadlineType != nil {
		s := strings.ToLower(string(*input.DeadlineType))
		params.DeadlineType = &s
	}
	if input.DeadlineDate != nil {
		params.DeadlineDate = pgtype.Date{Time: input.DeadlineDate.Time, Valid: true}
	}
	if input.DeadlineDays != nil {
		v := int32(*input.DeadlineDays)
		params.DeadlineDays = &v
	}
	params.Notes = input.Notes
	if input.RoutineID != nil {
		params.RoutineID = uuidToPgtype(*input.RoutineID)
	}
	return params
}

func (taskConv) ToUpdateParams(id uuid.UUID, input model.UpdateTaskInput) db.UpdateTaskParams {
	params := db.UpdateTaskParams{
		ID: uuidToPgtype(id),
	}
	if input.Title != nil {
		params.Title = input.Title
	}
	if input.Category != nil {
		s := strings.ToLower(string(*input.Category))
		params.Category = &s
	}
	if input.Priority != nil {
		s := strings.ToLower(string(*input.Priority))
		params.Priority = &s
	}
	if input.EstimatedMinutes != nil {
		v := int32(*input.EstimatedMinutes)
		params.EstimatedMinutes = &v
	}
	if input.DeadlineType != nil {
		s := strings.ToLower(string(*input.DeadlineType))
		params.DeadlineType = &s
	}
	if input.DeadlineDate != nil {
		params.DeadlineDate = pgtype.Date{Time: input.DeadlineDate.Time, Valid: true}
	}
	if input.DeadlineDays != nil {
		v := int32(*input.DeadlineDays)
		params.DeadlineDays = &v
	}
	if input.Notes != nil {
		params.Notes = input.Notes
	}
	if input.RoutineID != nil {
		params.RoutineID = uuidToPgtype(*input.RoutineID)
	}
	return params
}

func validateDeadline(deadlineType *model.DeadlineType, deadlineDate *model.Date, deadlineDays *int) error {
	if deadlineType == nil {
		return nil
	}
	switch *deadlineType {
	case model.DeadlineTypeHard:
		if deadlineDate == nil || deadlineDate.Time.IsZero() {
			return fmt.Errorf("deadlineDate required for HARD deadline")
		}
	case model.DeadlineTypeHorizon:
		if deadlineDays == nil {
			return fmt.Errorf("deadlineDays required for HORIZON deadline")
		}
	}
	return nil
}
