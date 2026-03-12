package graph

import (
	"context"
	"fmt"
	"time"

	"dayos/db"
	"dayos/planner"

	"github.com/google/uuid"
)

func boolPtrVal(b bool) *bool { return &b }

// buildPlanChatInput gathers all context data from stores and builds the planner input.
func (r *mutationResolver) buildPlanChatInput(ctx context.Context, plan db.DayPlan, userMessage string, planDate time.Time, isReplan bool) (planner.PlanChatInput, error) {
	input := planner.PlanChatInput{
		UserMessage: userMessage,
		IsReplan:    isReplan,
	}

	// Gather active context entries
	allEntries, err := r.ContextStore.ListContextEntries(ctx, nil)
	if err != nil {
		return input, fmt.Errorf("listing context entries: %w", err)
	}
	for _, e := range allEntries {
		if e.IsActive != nil && *e.IsActive {
			input.ContextEntries = append(input.ContextEntries, planner.ContextEntry{
				Key:   e.Key,
				Value: e.Value,
			})
		}
	}

	// Gather routines for today's day of week
	dow := int32(planDate.Weekday()) // Sunday=0, same as Go's convention
	routines, err := r.RoutineStore.ListRoutinesForDay(ctx, dow)
	if err != nil {
		return input, fmt.Errorf("listing routines: %w", err)
	}
	for _, r := range routines {
		ri := planner.RoutineInfo{
			Title:    r.Title,
			Category: r.Category,
		}
		if r.PreferredDurationMin != nil {
			ri.DurationMin = int(*r.PreferredDurationMin)
		}
		if r.PreferredTimeOfDay != nil {
			ri.PreferredTime = *r.PreferredTimeOfDay
		} else {
			ri.PreferredTime = "any"
		}
		input.Routines = append(input.Routines, ri)
	}

	// Gather schedulable tasks
	tasks, err := r.TaskStore.ListSchedulableTasks(ctx)
	if err != nil {
		return input, fmt.Errorf("listing tasks: %w", err)
	}
	var taskData []planner.TaskData
	for _, t := range tasks {
		td := planner.TaskData{
			Title:    t.Title,
			Category: t.Category,
			Priority: t.Priority,
			TaskID:   uuid.UUID(t.ID.Bytes).String(),
		}
		if t.EstimatedMinutes != nil {
			td.EstimatedMinutes = int(*t.EstimatedMinutes)
		}
		if t.ActualMinutes != nil {
			td.ActualMinutes = int(*t.ActualMinutes)
		}
		if t.DeadlineType != nil {
			td.DeadlineType = *t.DeadlineType
		}
		if t.DeadlineDate.Valid {
			td.DeadlineDate = t.DeadlineDate.Time.Format("2006-01-02")
		}
		if t.DeadlineDays != nil {
			td.DeadlineDays = int(*t.DeadlineDays)
		}
		if t.TimesDeferred != nil {
			td.TimesDeferred = int(*t.TimesDeferred)
		}
		taskData = append(taskData, td)
	}
	input.Tasks = planner.FormatTaskBacklog(taskData)

	// Gather carry-over tasks from most recent past plan
	input.CarryOverTasks = r.getCarryOverTasks(ctx, planDate)

	// Load conversation history
	existingMessages, err := r.DayPlanStore.GetPlanMessages(ctx, plan.ID)
	if err != nil {
		return input, fmt.Errorf("fetching plan messages: %w", err)
	}
	for _, m := range existingMessages {
		input.History = append(input.History, planner.Message{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	// For replanning, include current blocks and time
	if isReplan {
		input.CurrentBlocks = string(plan.Blocks)
		input.CurrentTime = time.Now().Format("15:04")
	}

	return input, nil
}
