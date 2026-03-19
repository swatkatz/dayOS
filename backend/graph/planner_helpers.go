package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"dayos/db"
	"dayos/identity"
	"dayos/planner"
	"dayos/tz"

	"github.com/google/uuid"
)

func boolPtrVal(b bool) *bool { return &b }

// buildPlanChatInput gathers all context data from stores and builds the planner input.
func (r *mutationResolver) buildPlanChatInput(ctx context.Context, plan db.DayPlan, userMessage string, planDate time.Time, isReplan bool) (planner.PlanChatInput, error) {
	// Get user display name for personalized prompts
	userName := ""
	if u, ok := identity.FromContext(ctx); ok {
		userName = u.DisplayName
	}

	// Compute date label and whether this is a future plan.
	// planDate arrives as UTC midnight from the Date scalar — re-interpret in user's timezone
	// so day comparisons are correct for all timezones.
	userNow := time.Now().In(tz.FromContext(ctx))
	userToday := time.Date(userNow.Year(), userNow.Month(), userNow.Day(), 0, 0, 0, 0, userNow.Location())
	planDateLocal := planDate.In(userNow.Location())
	planDay := time.Date(planDateLocal.Year(), planDateLocal.Month(), planDateLocal.Day(), 0, 0, 0, 0, userNow.Location())
	isFuturePlan := planDay.After(userToday)

	var planDateLabel string
	switch {
	case planDay.Equal(userToday):
		planDateLabel = fmt.Sprintf("TODAY (%s, %s)", planDay.Weekday(), planDay.Format("January 2"))
	case planDay.Equal(userToday.AddDate(0, 0, 1)):
		planDateLabel = fmt.Sprintf("TOMORROW (%s, %s)", planDay.Weekday(), planDay.Format("January 2"))
	default:
		planDateLabel = strings.ToUpper(planDay.Format("Monday, January 2"))
	}

	input := planner.PlanChatInput{
		UserMessage:   userMessage,
		UserName:      userName,
		IsReplan:      isReplan,
		PlanDateLabel: planDateLabel,
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

	// Gather routines for the plan's day of week (use timezone-adjusted date)
	dow := int32(planDay.Weekday()) // Sunday=0, same as Go's convention
	routines, err := r.RoutineStore.ListRoutinesForDay(ctx, dow)
	if err != nil {
		return input, fmt.Errorf("listing routines: %w", err)
	}
	for _, r := range routines {
		ri := planner.RoutineInfo{
			RoutineID: uuid.UUID(r.ID.Bytes).String(),
			Title:     r.Title,
			Category:  r.Category,
		}
		if r.PreferredDurationMin != nil {
			ri.DurationMin = int(*r.PreferredDurationMin)
		}
		if r.PreferredExactTime != nil {
			ri.ExactTime = *r.PreferredExactTime
		} else if r.PreferredTimeOfDay != nil {
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

	// Index parent tasks so subtasks can inherit deadlines
	parentDeadlines := make(map[string]db.Task)
	for _, t := range tasks {
		if !t.ParentID.Valid {
			parentDeadlines[uuid.UUID(t.ID.Bytes).String()] = t
		}
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
		// Inherit deadline from parent if subtask has none
		if t.ParentID.Valid && td.DeadlineType == "" {
			parentID := uuid.UUID(t.ParentID.Bytes).String()
			if parent, ok := parentDeadlines[parentID]; ok {
				if parent.DeadlineType != nil {
					td.DeadlineType = *parent.DeadlineType
				}
				if parent.DeadlineDate.Valid {
					td.DeadlineDate = parent.DeadlineDate.Time.Format("2006-01-02")
				}
				if parent.DeadlineDays != nil {
					td.DeadlineDays = int(*parent.DeadlineDays)
				}
			}
		}
		if t.TimesDeferred != nil {
			td.TimesDeferred = int(*t.TimesDeferred)
		}
		taskData = append(taskData, td)
	}
	input.Tasks = planner.FormatTaskBacklog(taskData)

	// Gather calendar events if connected
	if r.Calendar != nil {
		calResult, calErr := r.Calendar.GetEvents(ctx, planDate)
		if calErr == nil && calResult.Connected {
			for _, e := range calResult.Events {
				input.CalendarEvents = append(input.CalendarEvents, planner.CalendarEventInfo{
					Title:         e.Title,
					StartTime:     e.StartTime,
					Duration:      e.Duration,
					AllDay:        e.AllDay,
					AttendeeCount: e.AttendeeCount,
					EventType:     e.EventType,
				})
			}
		}
	}

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

	// For replanning, split blocks into done/skipped/remaining so the AI only replans what's needed
	if isReplan {
		// Only inject current wall-clock time when replanning today's plan.
		// For a future date, current time is irrelevant — the AI plans the full day.
		if !isFuturePlan {
			input.CurrentTime = userNow.Format("15:04")
		}

		var allBlocks []json.RawMessage
		if err := json.Unmarshal(plan.Blocks, &allBlocks); err == nil {
			type blockFlags struct {
				Done    bool `json:"done"`
				Skipped bool `json:"skipped"`
			}
			var doneBlocks, remainingBlocks []json.RawMessage
			for _, raw := range allBlocks {
				var flags blockFlags
				json.Unmarshal(raw, &flags)
				switch {
				case flags.Done:
					doneBlocks = append(doneBlocks, raw)
				case flags.Skipped:
					// skipped blocks are preserved but not sent to AI
				default:
					remainingBlocks = append(remainingBlocks, raw)
				}
			}
			completed, _ := json.Marshal(doneBlocks)
			remaining, _ := json.Marshal(remainingBlocks)
			input.CompletedBlocks = string(completed)
			input.CurrentBlocks = string(remaining)
		} else {
			input.CurrentBlocks = string(plan.Blocks)
			input.CompletedBlocks = "[]"
		}
	}

	return input, nil
}
