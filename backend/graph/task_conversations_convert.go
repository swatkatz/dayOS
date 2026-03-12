package graph

import (
	"dayos/db"
	"dayos/graph/model"

	"github.com/google/uuid"
)

type taskConvConv struct{}

var taskConversationConverter taskConvConv

func (taskConvConv) FromDB(c db.TaskConversation, messages []db.TaskMessage) *model.TaskConversation {
	out := &model.TaskConversation{
		ID:       uuid.UUID(c.ID.Bytes),
		Status:   c.Status,
		Messages: make([]*model.TaskMessage, len(messages)),
	}
	if c.ParentTaskID.Valid {
		pid := uuid.UUID(c.ParentTaskID.Bytes)
		out.ParentTaskID = &pid
	}
	if c.CreatedAt.Valid {
		out.CreatedAt = model.DateTime{Time: c.CreatedAt.Time}
	}
	for i, m := range messages {
		out.Messages[i] = &model.TaskMessage{
			ID:      uuid.UUID(m.ID.Bytes),
			Role:    m.Role,
			Content: m.Content,
		}
		if m.CreatedAt.Valid {
			out.Messages[i].CreatedAt = model.DateTime{Time: m.CreatedAt.Time}
		}
	}
	return out
}
