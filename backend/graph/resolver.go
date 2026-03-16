package graph

import (
	"context"

	"dayos/planner"
)

// PlannerService abstracts the planner for testability.
type PlannerService interface {
	PlanChat(ctx context.Context, input planner.PlanChatInput) (*planner.PlanChatOutput, error)
	TaskChat(ctx context.Context, history []planner.Message, userMessage string, userName string) (*planner.TaskChatOutput, error)
}

type Resolver struct {
	RoutineStore          RoutineStore
	TaskStore             TaskStore
	ContextStore          ContextStore
	DayPlanStore          DayPlanStore
	TaskConversationStore TaskConversationStore
	Planner               PlannerService
	Calendar              CalendarService
}
