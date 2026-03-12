package graph

import (
	"encoding/json"
	"strings"

	"dayos/db"
	"dayos/graph/model"

	"github.com/google/uuid"
)

type dayPlanConv struct{}

var dayPlanConverter dayPlanConv

type jsonBlock struct {
	ID        string  `json:"id"`
	Time      string  `json:"time"`
	Duration  int     `json:"duration"`
	Title     string  `json:"title"`
	Category  string  `json:"category"`
	TaskID    *string `json:"task_id"`
	RoutineID *string `json:"routine_id"`
	Notes     *string `json:"notes"`
	Skipped   bool    `json:"skipped"`
}

func (dayPlanConv) FromDB(p db.DayPlan, messages []db.PlanMessage) *model.DayPlan {
	out := &model.DayPlan{
		ID:       uuid.UUID(p.ID.Bytes),
		PlanDate: model.Date{Time: p.PlanDate.Time},
		Status:   model.PlanStatus(strings.ToUpper(p.Status)),
		Blocks:   parseBlocks(p.Blocks),
		Messages: make([]*model.PlanMessage, len(messages)),
	}
	if p.CreatedAt.Valid {
		out.CreatedAt = model.DateTime{Time: p.CreatedAt.Time}
	}
	if p.UpdatedAt.Valid {
		out.UpdatedAt = model.DateTime{Time: p.UpdatedAt.Time}
	}
	for i, m := range messages {
		out.Messages[i] = &model.PlanMessage{
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

func parseBlocks(data []byte) []*model.PlanBlock {
	if len(data) == 0 {
		return []*model.PlanBlock{}
	}
	var raw []jsonBlock
	if err := json.Unmarshal(data, &raw); err != nil {
		return []*model.PlanBlock{}
	}
	blocks := make([]*model.PlanBlock, 0, len(raw))
	for _, b := range raw {
		if b.ID == "" || b.Time == "" || b.Title == "" {
			continue
		}
		cat := model.Category(strings.ToUpper(b.Category))
		if !cat.IsValid() {
			cat = model.CategoryAdmin
		}
		block := &model.PlanBlock{
			ID:       b.ID,
			Time:     b.Time,
			Duration: b.Duration,
			Title:    b.Title,
			Category: cat,
			Notes:    b.Notes,
			Skipped:  b.Skipped,
		}
		if b.TaskID != nil {
			if uid, err := uuid.Parse(*b.TaskID); err == nil {
				block.TaskID = &uid
			}
		}
		if b.RoutineID != nil {
			if uid, err := uuid.Parse(*b.RoutineID); err == nil {
				block.RoutineID = &uid
			}
		}
		blocks = append(blocks, block)
	}
	return blocks
}

func marshalBlocks(blocks []*model.PlanBlock) ([]byte, error) {
	raw := make([]jsonBlock, len(blocks))
	for i, b := range blocks {
		raw[i] = jsonBlock{
			ID:       b.ID,
			Time:     b.Time,
			Duration: b.Duration,
			Title:    b.Title,
			Category: strings.ToLower(string(b.Category)),
			Notes:    b.Notes,
			Skipped:  b.Skipped,
		}
		if b.TaskID != nil {
			s := b.TaskID.String()
			raw[i].TaskID = &s
		}
		if b.RoutineID != nil {
			s := b.RoutineID.String()
			raw[i].RoutineID = &s
		}
	}
	return json.Marshal(raw)
}
