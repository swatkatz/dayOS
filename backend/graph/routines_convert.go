package graph

import (
	"dayos/db"
	"dayos/graph/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type routineConv struct{}

var routineConverter routineConv

func (routineConv) FromDB(r db.Routine) *model.Routine {
	out := &model.Routine{
		ID:        uuid.UUID(r.ID.Bytes),
		Title:     r.Title,
		Category:  model.Category(r.Category),
		Frequency: model.Frequency(r.Frequency),
		Notes:     r.Notes,
	}
	if r.IsActive != nil {
		out.IsActive = *r.IsActive
	}
	if r.PreferredTimeOfDay != nil {
		tod := model.TimeOfDay(*r.PreferredTimeOfDay)
		out.PreferredTimeOfDay = &tod
	}
	if r.PreferredDurationMin != nil {
		d := int(*r.PreferredDurationMin)
		out.PreferredDurationMin = &d
	}
	out.PreferredExactTime = r.PreferredExactTime
	if len(r.DaysOfWeek) > 0 {
		days := make([]int, len(r.DaysOfWeek))
		for i, d := range r.DaysOfWeek {
			days[i] = int(d)
		}
		out.DaysOfWeek = days
	}
	return out
}

func (routineConv) ToDB(input model.CreateRoutineInput) db.UpsertRoutineParams {
	params := db.UpsertRoutineParams{
		Title:     input.Title,
		Category:  string(input.Category),
		Frequency: string(input.Frequency),
		Notes:     input.Notes,
	}
	for _, d := range input.DaysOfWeek {
		params.DaysOfWeek = append(params.DaysOfWeek, int32(d))
	}
	if input.PreferredTimeOfDay != nil {
		s := string(*input.PreferredTimeOfDay)
		params.PreferredTimeOfDay = &s
	}
	if input.PreferredDurationMin != nil {
		v := int32(*input.PreferredDurationMin)
		params.PreferredDurationMin = &v
	}
	params.PreferredExactTime = input.PreferredExactTime
	return params
}

func (routineConv) MergeParams(existing db.Routine, input model.UpdateRoutineInput) db.UpsertRoutineParams {
	params := db.UpsertRoutineParams{
		ID:                   existing.ID,
		Title:                existing.Title,
		Category:             existing.Category,
		Frequency:            existing.Frequency,
		DaysOfWeek:           existing.DaysOfWeek,
		PreferredTimeOfDay:   existing.PreferredTimeOfDay,
		PreferredDurationMin: existing.PreferredDurationMin,
		PreferredExactTime:   existing.PreferredExactTime,
		Notes:                existing.Notes,
		IsActive:             existing.IsActive,
	}
	if input.Title != nil {
		params.Title = *input.Title
	}
	if input.Category != nil {
		params.Category = string(*input.Category)
	}
	if input.Frequency != nil {
		params.Frequency = string(*input.Frequency)
	}
	if input.DaysOfWeek != nil {
		params.DaysOfWeek = nil
		for _, d := range input.DaysOfWeek {
			params.DaysOfWeek = append(params.DaysOfWeek, int32(d))
		}
	}
	if input.PreferredTimeOfDay != nil {
		s := string(*input.PreferredTimeOfDay)
		params.PreferredTimeOfDay = &s
	}
	if input.PreferredDurationMin != nil {
		v := int32(*input.PreferredDurationMin)
		params.PreferredDurationMin = &v
	}
	if input.PreferredExactTime != nil {
		params.PreferredExactTime = input.PreferredExactTime
	}
	if input.Notes != nil {
		params.Notes = input.Notes
	}
	if input.IsActive != nil {
		params.IsActive = input.IsActive
	}
	return params
}

func uuidToPgtype(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}
