package graph

import (
	"context"
	"fmt"
	"strings"
	"time"

	"dayos/graph/model"
	"dayos/planner"
)

// EffectiveDaysRemaining computes deadline pressure for horizon tasks.
// Returns -1 if the task has no horizon deadline.
func EffectiveDaysRemaining(deadlineDays *int, timesDeferred int) int {
	if deadlineDays == nil {
		return -1
	}
	return *deadlineDays - timesDeferred
}

// EffectivePriority computes the escalated priority based on deferral count.
func EffectivePriority(assignedPriority string, timesDeferred int) string {
	switch {
	case timesDeferred >= 2:
		return "high"
	case timesDeferred == 1:
		switch assignedPriority {
		case "low":
			return "medium"
		case "medium":
			return "high"
		default:
			return assignedPriority
		}
	default:
		return assignedPriority
	}
}

// IsOverdue returns true if a task has been deferred 3+ times.
func IsOverdue(timesDeferred int) bool {
	return timesDeferred >= 3
}

// getCarryOverTasks finds skipped blocks from the most recent past accepted plan
// whose linked tasks are not completed, and builds carry-over task entries.
func (r *mutationResolver) getCarryOverTasks(ctx context.Context, today time.Time) []planner.CarryOverTask {
	plans, err := r.DayPlanStore.RecentPlans(ctx, 10)
	if err != nil {
		return nil
	}

	// Find the most recent accepted plan before today
	var pastPlanBlocks []*model.PlanBlock
	for _, p := range plans {
		if p.Status == "accepted" && p.PlanDate.Time.Before(today) {
			pastPlanBlocks = parseBlocks(p.Blocks)
			break
		}
	}
	if pastPlanBlocks == nil {
		return nil
	}

	var result []planner.CarryOverTask
	for _, b := range pastPlanBlocks {
		if !b.Skipped || b.TaskID == nil {
			continue
		}

		task, err := r.TaskStore.GetTask(ctx, uuidToPgtype(*b.TaskID))
		if err != nil {
			continue
		}
		if task.IsCompleted != nil && *task.IsCompleted {
			continue
		}

		timesDeferred := 0
		if task.TimesDeferred != nil {
			timesDeferred = int(*task.TimesDeferred)
		}

		var deadlineDays *int
		if task.DeadlineDays != nil {
			v := int(*task.DeadlineDays)
			deadlineDays = &v
		}

		effectiveDays := EffectiveDaysRemaining(deadlineDays, timesDeferred)
		effectiveDeadline := ""
		if effectiveDays >= 0 {
			if effectiveDays <= 3 {
				effectiveDeadline = fmt.Sprintf("URGENT — within %d days", effectiveDays)
			} else {
				effectiveDeadline = fmt.Sprintf("within %d days", effectiveDays)
			}
		}

		result = append(result, planner.CarryOverTask{
			Title:             b.Title,
			Category:          strings.ToLower(string(b.Category)),
			TimesDeferred:     timesDeferred,
			EffectiveDeadline: effectiveDeadline,
			TaskID:            b.TaskID.String(),
		})
	}

	return result
}
