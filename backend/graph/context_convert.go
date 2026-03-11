package graph

import (
	"strings"

	"dayos/db"
	"dayos/graph/model"

	"github.com/google/uuid"
)

type contextConv struct{}

var contextConverter contextConv

func (contextConv) FromDB(e db.ContextEntry) *model.ContextEntry {
	out := &model.ContextEntry{
		ID:       uuid.UUID(e.ID.Bytes),
		Category: model.ContextCategory(strings.ToUpper(e.Category)),
		Key:      e.Key,
		Value:    e.Value,
	}
	if e.IsActive != nil {
		out.IsActive = *e.IsActive
	}
	if e.CreatedAt.Valid {
		out.CreatedAt = model.DateTime{Time: e.CreatedAt.Time}
	}
	return out
}

func (contextConv) ToDB(input model.UpsertContextInput) db.UpsertContextEntryParams {
	return db.UpsertContextEntryParams{
		Category: strings.ToLower(string(input.Category)),
		Key:      input.Key,
		Value:    input.Value,
	}
}
