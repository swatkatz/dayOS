package graph_test

import (
	"context"
	"encoding/json"
	"testing"

	"dayos/db"
	"dayos/graph"
	"dayos/graph/model"
	"dayos/planner"

	"github.com/google/uuid"
)

// Test Anchor 5: Given a task with deadline_type='horizon', deadline_days=14, times_deferred=3,
// when EffectiveDaysRemaining is computed, then the result is 11.
func TestEffectiveDaysRemaining_HorizonTask(t *testing.T) {
	days := 14
	result := graph.EffectiveDaysRemaining(&days, 3)
	if result != 11 {
		t.Errorf("EffectiveDaysRemaining = %d, want 11", result)
	}
}

func TestEffectiveDaysRemaining_NoDeadline(t *testing.T) {
	result := graph.EffectiveDaysRemaining(nil, 5)
	if result != -1 {
		t.Errorf("EffectiveDaysRemaining = %d, want -1", result)
	}
}

func TestEffectiveDaysRemaining_Negative(t *testing.T) {
	days := 2
	result := graph.EffectiveDaysRemaining(&days, 5)
	if result != -3 {
		t.Errorf("EffectiveDaysRemaining = %d, want -3", result)
	}
}

// Test Anchor 6: Given a task with times_deferred=1 and priority='low',
// when EffectivePriority is computed, then the result is 'medium'.
func TestEffectivePriority_OneDeferred_Low(t *testing.T) {
	result := graph.EffectivePriority("low", 1)
	if result != "medium" {
		t.Errorf("EffectivePriority = %s, want medium", result)
	}
}

// Test Anchor 7: Given a task with times_deferred=2 and priority='low',
// when EffectivePriority is computed, then the result is 'high'.
func TestEffectivePriority_TwoDeferred_Low(t *testing.T) {
	result := graph.EffectivePriority("low", 2)
	if result != "high" {
		t.Errorf("EffectivePriority = %s, want high", result)
	}
}

func TestEffectivePriority_Table(t *testing.T) {
	tests := []struct {
		name           string
		priority       string
		timesDeferred  int
		wantPriority   string
	}{
		{"zero deferred keeps priority", "low", 0, "low"},
		{"1 deferred low->medium", "low", 1, "medium"},
		{"1 deferred medium->high", "medium", 1, "high"},
		{"1 deferred high stays high", "high", 1, "high"},
		{"2 deferred any->high", "low", 2, "high"},
		{"2 deferred medium->high", "medium", 2, "high"},
		{"3 deferred any->high", "low", 3, "high"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := graph.EffectivePriority(tc.priority, tc.timesDeferred)
			if got != tc.wantPriority {
				t.Errorf("EffectivePriority(%q, %d) = %q, want %q",
					tc.priority, tc.timesDeferred, got, tc.wantPriority)
			}
		})
	}
}

// Test Anchor 8: Given a task with times_deferred=3, when IsOverdue is checked, then it returns true.
func TestIsOverdue(t *testing.T) {
	tests := []struct {
		timesDeferred int
		want          bool
	}{
		{0, false},
		{1, false},
		{2, false},
		{3, true},
		{5, true},
	}
	for _, tc := range tests {
		got := graph.IsOverdue(tc.timesDeferred)
		if got != tc.want {
			t.Errorf("IsOverdue(%d) = %v, want %v", tc.timesDeferred, got, tc.want)
		}
	}
}

// Test Anchor 1: Given a skipped block with task_id pointing to a task with times_deferred=0,
// when resolveSkippedBlock is called with intentional=false, then tasks.times_deferred becomes 1
// and last_deferred_at is set.
func TestResolveSkippedBlock_NotIntentional(t *testing.T) {
	taskStore := newMockTaskStore()
	planStore := newMockDayPlanStore()

	task := factoryTask(taskStore, "Interview prep")
	taskID := uuid.UUID(task.ID.Bytes).String()

	blocks := []map[string]any{
		{"id": "block-1", "time": "09:00", "duration": 60, "title": "Interview prep", "category": "interview", "skipped": true, "task_id": taskID},
	}
	blocksJSON, _ := json.Marshal(blocks)
	plan := factoryDayPlan(planStore, "2026-03-10", "accepted", blocksJSON)

	r := &graph.Resolver{
		TaskStore:    taskStore,
		DayPlanStore: planStore,
	}

	planID := uuid.UUID(plan.ID.Bytes)
	result, err := r.Mutation().ResolveSkippedBlock(context.Background(), planID, "block-1", false)
	if err != nil {
		t.Fatalf("ResolveSkippedBlock: %v", err)
	}
	if !result {
		t.Fatal("expected true")
	}

	// Verify task was updated
	updated, _ := taskStore.GetTask(context.Background(), task.ID)
	if updated.TimesDeferred == nil || *updated.TimesDeferred != 1 {
		t.Errorf("times_deferred = %v, want 1", updated.TimesDeferred)
	}
	if !updated.LastDeferredAt.Valid {
		t.Error("expected last_deferred_at to be set")
	}
}

// Test Anchor 2: Given a skipped block with task_id, when resolveSkippedBlock is called
// with intentional=true, then tasks.times_deferred remains unchanged.
func TestResolveSkippedBlock_Intentional(t *testing.T) {
	taskStore := newMockTaskStore()
	planStore := newMockDayPlanStore()

	task := factoryTask(taskStore, "Interview prep")
	taskID := uuid.UUID(task.ID.Bytes).String()

	blocks := []map[string]any{
		{"id": "block-1", "time": "09:00", "duration": 60, "title": "Interview prep", "category": "interview", "skipped": true, "task_id": taskID},
	}
	blocksJSON, _ := json.Marshal(blocks)
	plan := factoryDayPlan(planStore, "2026-03-10", "accepted", blocksJSON)

	r := &graph.Resolver{
		TaskStore:    taskStore,
		DayPlanStore: planStore,
	}

	planID := uuid.UUID(plan.ID.Bytes)
	result, err := r.Mutation().ResolveSkippedBlock(context.Background(), planID, "block-1", true)
	if err != nil {
		t.Fatalf("ResolveSkippedBlock: %v", err)
	}
	if !result {
		t.Fatal("expected true")
	}

	// Verify task was NOT updated
	updated, _ := taskStore.GetTask(context.Background(), task.ID)
	if updated.TimesDeferred == nil || *updated.TimesDeferred != 0 {
		t.Errorf("times_deferred = %v, want 0", updated.TimesDeferred)
	}
}

// Test Anchor 3: Given a block that is not skipped, when resolveSkippedBlock is called,
// then the system returns an error.
func TestResolveSkippedBlock_NotSkipped(t *testing.T) {
	planStore := newMockDayPlanStore()

	blocks := []map[string]any{
		{"id": "block-1", "time": "09:00", "duration": 60, "title": "Interview prep", "category": "interview", "skipped": false},
	}
	blocksJSON, _ := json.Marshal(blocks)
	plan := factoryDayPlan(planStore, "2026-03-10", "accepted", blocksJSON)

	r := &graph.Resolver{
		DayPlanStore: planStore,
	}

	planID := uuid.UUID(plan.ID.Bytes)
	_, err := r.Mutation().ResolveSkippedBlock(context.Background(), planID, "block-1", false)
	if err == nil {
		t.Fatal("expected error for non-skipped block")
	}
}

// Test Anchor 4: Given a skipped block with no task_id, when resolveSkippedBlock is called,
// then it returns true without any task modification.
func TestResolveSkippedBlock_NoTaskID(t *testing.T) {
	planStore := newMockDayPlanStore()

	blocks := []map[string]any{
		{"id": "block-1", "time": "09:00", "duration": 60, "title": "Exercise", "category": "exercise", "skipped": true},
	}
	blocksJSON, _ := json.Marshal(blocks)
	plan := factoryDayPlan(planStore, "2026-03-10", "accepted", blocksJSON)

	r := &graph.Resolver{
		DayPlanStore: planStore,
	}

	planID := uuid.UUID(plan.ID.Bytes)
	result, err := r.Mutation().ResolveSkippedBlock(context.Background(), planID, "block-1", false)
	if err != nil {
		t.Fatalf("ResolveSkippedBlock: %v", err)
	}
	if !result {
		t.Fatal("expected true")
	}
}

// Test Anchor 10: Given yesterday's accepted plan has 2 skipped blocks with task_ids
// (one task completed, one not), when GetCarryOverTasks is called, then only the
// non-completed task is returned.
func TestGetCarryOverTasks_FiltersCompletedTasks(t *testing.T) {
	taskStore := newMockTaskStore()
	planStore := newMockDayPlanStore()

	incompleteTask := factoryTask(taskStore, "Incomplete task")
	completedTask := factoryTask(taskStore, "Completed task")
	// Mark one as completed
	taskStore.CompleteTask(context.Background(), completedTask.ID)

	incompleteID := uuid.UUID(incompleteTask.ID.Bytes).String()
	completedID := uuid.UUID(completedTask.ID.Bytes).String()

	blocks := []map[string]any{
		{"id": "block-1", "time": "09:00", "duration": 60, "title": "Incomplete task", "category": "job", "skipped": true, "task_id": incompleteID},
		{"id": "block-2", "time": "10:00", "duration": 60, "title": "Completed task", "category": "job", "skipped": true, "task_id": completedID},
		{"id": "block-3", "time": "11:00", "duration": 45, "title": "Exercise", "category": "exercise", "skipped": false},
	}
	blocksJSON, _ := json.Marshal(blocks)
	// Yesterday's plan
	_ = factoryDayPlan(planStore, "2026-03-11", "accepted", blocksJSON)

	r := &graph.Resolver{
		TaskStore:    taskStore,
		DayPlanStore: planStore,
	}

	// Call sendPlanMessage indirectly won't work without planner, so test via exported method
	// We need to test getCarryOverTasks which is unexported. Test it through the planner input.
	// Instead, set up everything needed for buildPlanChatInput.
	contextStore := newMockContextStore()
	routineStore := newMockRoutineStore()
	r.ContextStore = contextStore
	r.RoutineStore = routineStore
	r.Planner = &mockPlanner{}

	// Create today's plan so SendPlanMessage works with existing plan
	_ = factoryDayPlan(planStore, "2026-03-12", "draft", []byte("[]"))

	date, _ := model.UnmarshalDate("2026-03-12")
	result, err := r.Mutation().SendPlanMessage(context.Background(), date, "Plan my day")
	if err != nil {
		t.Fatalf("SendPlanMessage: %v", err)
	}
	// The mockPlanner returns empty blocks, but we care about carry-over being populated
	// We can't directly inspect PlanChatInput from here, so let's use a custom mock
	_ = result

	// Use a planner mock that captures input
	var capturedInput planner.PlanChatInput
	r.Planner = &mockPlanner{
		planChatFn: func(_ context.Context, input planner.PlanChatInput) (*planner.PlanChatOutput, error) {
			capturedInput = input
			return &planner.PlanChatOutput{
				Blocks:       []planner.Block{},
				RawResponses: []string{"[]"},
			}, nil
		},
	}

	// Delete and recreate today's plan to send a fresh message
	// Actually, the plan already exists so SendPlanMessage will use it
	// But it already has a message stored. Let's just call it again.
	_, err = r.Mutation().SendPlanMessage(context.Background(), date, "Replan please")
	if err != nil {
		t.Fatalf("SendPlanMessage: %v", err)
	}

	if len(capturedInput.CarryOverTasks) != 1 {
		t.Fatalf("expected 1 carry-over task, got %d", len(capturedInput.CarryOverTasks))
	}
	if capturedInput.CarryOverTasks[0].Title != "Incomplete task" {
		t.Errorf("carry-over task title = %q, want %q", capturedInput.CarryOverTasks[0].Title, "Incomplete task")
	}
}

// Test Anchor 11: Given a task with deadline_type='horizon', deadline_days=5, times_deferred=3
// (effective=2), the carry-over data shows it as urgent.
func TestGetCarryOverTasks_EffectiveDeadline(t *testing.T) {
	taskStore := newMockTaskStore()
	planStore := newMockDayPlanStore()

	deadlineDays := int32(5)
	horizonType := "horizon"
	task := factoryTask(taskStore, "Urgent task", func(p *db.CreateTaskParams) {
		p.DeadlineDays = &deadlineDays
		p.DeadlineType = &horizonType
	})
	// Increment times_deferred 3 times
	taskStore.IncrementTimesDeferred(context.Background(), task.ID)
	taskStore.IncrementTimesDeferred(context.Background(), task.ID)
	taskStore.IncrementTimesDeferred(context.Background(), task.ID)

	taskID := uuid.UUID(task.ID.Bytes).String()
	blocks := []map[string]any{
		{"id": "block-1", "time": "09:00", "duration": 60, "title": "Urgent task", "category": "job", "skipped": true, "task_id": taskID},
	}
	blocksJSON, _ := json.Marshal(blocks)
	_ = factoryDayPlan(planStore, "2026-03-11", "accepted", blocksJSON)

	contextStore := newMockContextStore()
	routineStore := newMockRoutineStore()

	var capturedInput planner.PlanChatInput
	r := &graph.Resolver{
		TaskStore:    taskStore,
		DayPlanStore: planStore,
		ContextStore: contextStore,
		RoutineStore: routineStore,
		Planner: &mockPlanner{
			planChatFn: func(_ context.Context, input planner.PlanChatInput) (*planner.PlanChatOutput, error) {
				capturedInput = input
				return &planner.PlanChatOutput{
					Blocks:       []planner.Block{},
					RawResponses: []string{"[]"},
				}, nil
			},
		},
	}

	date, _ := model.UnmarshalDate("2026-03-12")
	_, err := r.Mutation().SendPlanMessage(context.Background(), date, "Plan my day")
	if err != nil {
		t.Fatalf("SendPlanMessage: %v", err)
	}

	if len(capturedInput.CarryOverTasks) != 1 {
		t.Fatalf("expected 1 carry-over task, got %d", len(capturedInput.CarryOverTasks))
	}
	ct := capturedInput.CarryOverTasks[0]
	if ct.TimesDeferred != 3 {
		t.Errorf("times_deferred = %d, want 3", ct.TimesDeferred)
	}
	// effective = 5 - 3 = 2, which is <= 3, so should be URGENT
	if ct.EffectiveDeadline != "URGENT — within 2 days" {
		t.Errorf("effective_deadline = %q, want %q", ct.EffectiveDeadline, "URGENT — within 2 days")
	}
}

// Test: blockId does not exist in the plan
func TestResolveSkippedBlock_BlockNotFound(t *testing.T) {
	planStore := newMockDayPlanStore()

	blocks := []map[string]any{
		{"id": "block-1", "time": "09:00", "duration": 60, "title": "Exercise", "category": "exercise", "skipped": true},
	}
	blocksJSON, _ := json.Marshal(blocks)
	plan := factoryDayPlan(planStore, "2026-03-10", "accepted", blocksJSON)

	r := &graph.Resolver{
		DayPlanStore: planStore,
	}

	planID := uuid.UUID(plan.ID.Bytes)
	_, err := r.Mutation().ResolveSkippedBlock(context.Background(), planID, "nonexistent", false)
	if err == nil {
		t.Fatal("expected error for non-existent block")
	}
}
