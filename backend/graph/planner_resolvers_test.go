package graph_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"dayos/db"
	"dayos/graph"
	"dayos/graph/model"
	"dayos/planner"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func newPlannerResolver() (*graph.Resolver, *mockTaskStore, *mockContextStore, *mockRoutineStore, *mockDayPlanStore, *mockTaskConversationStore, *mockPlanner) {
	ts := newMockTaskStore()
	cs := newMockContextStore()
	rs := newMockRoutineStore()
	ds := newMockDayPlanStore()
	tcs := newMockTaskConversationStore()
	mp := &mockPlanner{}

	r := &graph.Resolver{
		TaskStore:             ts,
		ContextStore:          cs,
		RoutineStore:          rs,
		DayPlanStore:          ds,
		TaskConversationStore: tcs,
		Planner:               mp,
	}
	return r, ts, cs, rs, ds, tcs, mp
}

// Test anchor 1: sendPlanMessage creates plan + messages for new date
func TestSendPlanMessage_NewPlan(t *testing.T) {
	r, ts, cs, rs, _, _, mp := newPlannerResolver()
	_ = ts
	_ = rs

	// Seed context entries
	seedContextEntries(cs)

	// Seed routines (2 applicable for today)
	factoryRoutine(rs, "Daily exercise", func(p *db.UpsertRoutineParams) {
		p.Frequency = "daily"
		dur := int32(45)
		p.PreferredDurationMin = &dur
	})
	factoryRoutine(rs, "Interview prep", func(p *db.UpsertRoutineParams) {
		p.Frequency = "weekdays"
		p.Category = "interview"
		dur := int32(60)
		p.PreferredDurationMin = &dur
	})

	// Seed tasks (5 incomplete)
	for i := 0; i < 5; i++ {
		factoryTask(ts, "Task "+string(rune('A'+i)))
	}

	// Configure mock planner to return blocks — use tomorrow to avoid current-time filtering
	mp.planChatFn = func(_ context.Context, input planner.PlanChatInput) (*planner.PlanChatOutput, error) {
		// Verify context entries were passed
		if len(input.ContextEntries) == 0 {
			t.Error("expected context entries to be passed to planner")
		}
		return &planner.PlanChatOutput{
			Blocks: []planner.Block{
				{ID: "b1", Time: "09:00", Duration: 60, Title: "Interview prep", Category: "interview"},
				{ID: "b2", Time: "10:15", Duration: 45, Title: "Exercise", Category: "exercise"},
			},
			RawResponses: []string{`[{"id":"b1","time":"09:00","duration":60,"title":"Interview prep","category":"interview"},{"id":"b2","time":"10:15","duration":45,"title":"Exercise","category":"exercise"}]`},
		}, nil
	}

	tomorrow := model.Date{Time: time.Now().Truncate(24 * time.Hour).AddDate(0, 0, 1)}
	mutation := r.Mutation()
	result, err := mutation.SendPlanMessage(context.Background(), tomorrow, "Light day, just interview prep and exercise")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Plan should be draft
	if result.Status != model.PlanStatusDraft {
		t.Errorf("expected status DRAFT, got %v", result.Status)
	}

	// Should have 2 blocks
	if len(result.Blocks) != 2 {
		t.Errorf("expected 2 blocks, got %d", len(result.Blocks))
	}

	// Should have 2 messages (user + assistant)
	if len(result.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(result.Messages))
	}
	if result.Messages[0].Role != "user" {
		t.Errorf("expected first message role 'user', got %q", result.Messages[0].Role)
	}
	if result.Messages[1].Role != "assistant" {
		t.Errorf("expected second message role 'assistant', got %q", result.Messages[1].Role)
	}
}

// Test anchor 2: sendPlanMessage on draft includes history
func TestSendPlanMessage_DraftRefinement(t *testing.T) {
	r, _, cs, _, ds, _, mp := newPlannerResolver()
	seedContextEntries(cs)

	// Create existing draft plan with 2 messages
	plan := factoryDayPlan(ds, time.Now().Format("2006-01-02"), "draft", []byte(`[{"id":"b1","time":"09:00","duration":60,"title":"Work","category":"job"}]`))
	factoryPlanMessage(ds, plan.ID, "user", "Plan my day")
	factoryPlanMessage(ds, plan.ID, "assistant", `[{"id":"b1"}]`)

	var capturedInput planner.PlanChatInput
	mp.planChatFn = func(_ context.Context, input planner.PlanChatInput) (*planner.PlanChatOutput, error) {
		capturedInput = input
		return &planner.PlanChatOutput{
			Blocks:       []planner.Block{{ID: "b2", Time: "13:00", Duration: 60, Title: "After lunch prep", Category: "interview"}},
			RawResponses: []string{`[{"id":"b2"}]`},
		}, nil
	}

	today := model.Date{Time: time.Now().Truncate(24 * time.Hour)}
	mutation := r.Mutation()
	result, err := mutation.SendPlanMessage(context.Background(), today, "Move interview prep to after lunch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// History should include the 2 existing messages
	if len(capturedInput.History) != 2 {
		t.Errorf("expected 2 history messages, got %d", len(capturedInput.History))
	}

	// Should now have 4 messages total (2 old + 2 new)
	if len(result.Messages) != 4 {
		t.Errorf("expected 4 messages, got %d", len(result.Messages))
	}

	// Status should still be draft
	if result.Status != model.PlanStatusDraft {
		t.Errorf("expected status DRAFT, got %v", result.Status)
	}
}

// Test anchor 3: sendPlanMessage on accepted plan adds replanning context
func TestSendPlanMessage_Replanning(t *testing.T) {
	r, _, cs, _, ds, _, mp := newPlannerResolver()
	seedContextEntries(cs)

	blocksJSON := `[
		{"id":"b1","time":"09:00","duration":60,"title":"Morning work","category":"job","skipped":false},
		{"id":"b2","time":"11:00","duration":60,"title":"Mid-morning","category":"job","skipped":false},
		{"id":"b3","time":"14:00","duration":60,"title":"Afternoon","category":"job","skipped":false}
	]`
	plan := factoryDayPlan(ds, time.Now().Format("2006-01-02"), "accepted", []byte(blocksJSON))
	_ = plan

	var capturedInput planner.PlanChatInput
	mp.planChatFn = func(_ context.Context, input planner.PlanChatInput) (*planner.PlanChatOutput, error) {
		capturedInput = input
		return &planner.PlanChatOutput{
			Blocks:       []planner.Block{{ID: "b4", Time: "23:00", Duration: 60, Title: "Evening work", Category: "job"}},
			RawResponses: []string{`[{"id":"b4"}]`},
		}, nil
	}

	today := model.Date{Time: time.Now().Truncate(24 * time.Hour)}
	mutation := r.Mutation()
	result, err := mutation.SendPlanMessage(context.Background(), today, "Cancel the afternoon, baby is sick")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be replanning mode
	if !capturedInput.IsReplan {
		t.Error("expected IsReplan to be true")
	}
	// Current blocks should be included
	if capturedInput.CurrentBlocks == "" {
		t.Error("expected CurrentBlocks to be set for replanning")
	}
	// Status should be set back to draft
	if result.Status != model.PlanStatusDraft {
		t.Errorf("expected status DRAFT after replan, got %v", result.Status)
	}
}

// Test anchor 6: confirmTaskBreakdown creates parent + subtasks
func TestConfirmTaskBreakdown_CreatesTasksFromProposal(t *testing.T) {
	r, _, _, _, _, tcs, _ := newPlannerResolver()

	// Create a conversation with a proposal as the last assistant message
	conv, _ := tcs.CreateTaskConversation(context.Background())
	tcs.CreateTaskMessage(context.Background(), db.CreateTaskMessageParams{
		ConversationID: conv.ID,
		Role:           "user",
		Content:        "I want to build an API",
	})
	proposalJSON := `{
		"status": "proposal",
		"parent": {"title": "Build API", "category": "project", "priority": "high"},
		"subtasks": [
			{"title": "Design schema", "estimated_minutes": 60, "category": "project"},
			{"title": "Write endpoints", "estimated_minutes": 120, "category": "project"},
			{"title": "Write tests", "estimated_minutes": 90, "category": "project"}
		]
	}`
	tcs.CreateTaskMessage(context.Background(), db.CreateTaskMessageParams{
		ConversationID: conv.ID,
		Role:           "assistant",
		Content:        proposalJSON,
	})

	mutation := r.Mutation()
	convID := uuid.UUID(conv.ID.Bytes)
	result, err := mutation.ConfirmTaskBreakdown(context.Background(), convID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 4 tasks (1 parent + 3 subtasks)
	if len(result) != 4 {
		t.Fatalf("expected 4 tasks, got %d", len(result))
	}

	// First should be the parent
	parent := result[0]
	if parent.Title != "Build API" {
		t.Errorf("expected parent title 'Build API', got %q", parent.Title)
	}
	if parent.ParentID != nil {
		t.Error("expected parent to have no parent_id")
	}

	// Subtasks should reference parent
	for i := 1; i < len(result); i++ {
		sub := result[i]
		if sub.ParentID == nil || *sub.ParentID != parent.ID {
			t.Errorf("subtask %d should have parent_id %v", i, parent.ID)
		}
	}

	// Conversation should be marked completed
	updatedConv, _ := tcs.GetTaskConversation(context.Background(), conv.ID)
	if updatedConv.Status != "completed" {
		t.Errorf("expected conversation status 'completed', got %q", updatedConv.Status)
	}

	// Conversation should be linked to parent
	if !updatedConv.ParentTaskID.Valid {
		t.Error("expected conversation parent_task_id to be set")
	}
}

// Test anchor 7: confirmTaskBreakdown with question returns error
func TestConfirmTaskBreakdown_RejectsQuestion(t *testing.T) {
	r, _, _, _, _, tcs, _ := newPlannerResolver()

	conv, _ := tcs.CreateTaskConversation(context.Background())
	tcs.CreateTaskMessage(context.Background(), db.CreateTaskMessageParams{
		ConversationID: conv.ID,
		Role:           "user",
		Content:        "I want to build something",
	})
	tcs.CreateTaskMessage(context.Background(), db.CreateTaskMessageParams{
		ConversationID: conv.ID,
		Role:           "assistant",
		Content:        `{"status": "question", "message": "What's the scope?"}`,
	})

	mutation := r.Mutation()
	convID := uuid.UUID(conv.ID.Bytes)
	_, err := mutation.ConfirmTaskBreakdown(context.Background(), convID)
	if err == nil {
		t.Fatal("expected error for question status")
	}
	if !strings.Contains(err.Error(), "No valid task proposal found") {
		t.Errorf("expected 'No valid task proposal found', got %q", err.Error())
	}
}

// Test anchor 9: parent tasks excluded from backlog in buildPlanChatInput
func TestBuildPlanChatInput_ExcludesParentTasks(t *testing.T) {
	r, ts, cs, _, ds, _, mp := newPlannerResolver()
	seedContextEntries(cs)

	// Create parent task (no estimated_minutes)
	parent := factoryTask(ts, "Parent goal", func(p *db.CreateTaskParams) {
		p.EstimatedMinutes = nil // parent tasks have nil estimated_minutes
	})

	// Create 2 incomplete subtasks
	factoryTask(ts, "Subtask 1", func(p *db.CreateTaskParams) {
		p.ParentID = parent.ID
		est := int32(60)
		p.EstimatedMinutes = &est
	})
	factoryTask(ts, "Subtask 2", func(p *db.CreateTaskParams) {
		p.ParentID = parent.ID
		est := int32(90)
		p.EstimatedMinutes = &est
	})

	// Create 1 completed subtask
	completedSub := factoryTask(ts, "Subtask completed", func(p *db.CreateTaskParams) {
		p.ParentID = parent.ID
		est := int32(30)
		p.EstimatedMinutes = &est
	})
	ts.CompleteTask(context.Background(), completedSub.ID)

	var capturedInput planner.PlanChatInput
	mp.planChatFn = func(_ context.Context, input planner.PlanChatInput) (*planner.PlanChatOutput, error) {
		capturedInput = input
		return &planner.PlanChatOutput{
			Blocks:       []planner.Block{},
			RawResponses: []string{"[]"},
		}, nil
	}

	// Create a plan for today so we can call sendPlanMessage
	today := time.Now().Truncate(24 * time.Hour)
	factoryDayPlan(ds, today.Format("2006-01-02"), "draft", []byte("[]"))

	mutation := r.Mutation()
	_, err := mutation.SendPlanMessage(context.Background(), model.Date{Time: today}, "Plan my day")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify: parent should NOT be in tasks (it has nil estimated_minutes)
	// Only the 2 incomplete subtasks should appear
	if len(capturedInput.Tasks) != 2 {
		t.Errorf("expected 2 tasks in backlog (only incomplete subtasks), got %d", len(capturedInput.Tasks))
		for _, task := range capturedInput.Tasks {
			t.Logf("  task: %s", task.Title)
		}
	}
}

// Test: startTaskConversation creates conversation and returns messages
func TestStartTaskConversation(t *testing.T) {
	r, _, _, _, _, _, mp := newPlannerResolver()

	mp.taskChatFn = func(_ context.Context, history []planner.Message, msg string) (*planner.TaskChatOutput, error) {
		return &planner.TaskChatOutput{
			RawResponse: `{"status": "question", "message": "What's the deadline?"}`,
		}, nil
	}

	mutation := r.Mutation()
	result, err := mutation.StartTaskConversation(context.Background(), "I want to prep for Meta interviews")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "active" {
		t.Errorf("expected status 'active', got %q", result.Status)
	}
	if len(result.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(result.Messages))
	}
}

// Test: sendTaskMessage includes conversation history
func TestSendTaskMessage_IncludesHistory(t *testing.T) {
	r, _, _, _, _, tcs, mp := newPlannerResolver()

	conv, _ := tcs.CreateTaskConversation(context.Background())
	tcs.CreateTaskMessage(context.Background(), db.CreateTaskMessageParams{
		ConversationID: conv.ID,
		Role:           "user",
		Content:        "I want to build an API",
	})
	tcs.CreateTaskMessage(context.Background(), db.CreateTaskMessageParams{
		ConversationID: conv.ID,
		Role:           "assistant",
		Content:        `{"status": "question", "message": "What's the deadline?"}`,
	})

	var capturedHistory []planner.Message
	mp.taskChatFn = func(_ context.Context, history []planner.Message, msg string) (*planner.TaskChatOutput, error) {
		capturedHistory = history
		return &planner.TaskChatOutput{
			RawResponse: `{"status": "question", "message": "What framework?"}`,
		}, nil
	}

	mutation := r.Mutation()
	convID := uuid.UUID(conv.ID.Bytes)
	_, err := mutation.SendTaskMessage(context.Background(), convID, "Within 2 weeks")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have passed 2 history messages
	if len(capturedHistory) != 2 {
		t.Errorf("expected 2 history messages, got %d", len(capturedHistory))
	}
}

// Test: replanning saves previous state and allows revert
func TestRevertPlan_RestoresPreviousBlocks(t *testing.T) {
	r, _, cs, _, ds, _, mp := newPlannerResolver()
	seedContextEntries(cs)

	originalBlocks := `[
		{"id":"b1","time":"09:00","duration":60,"title":"Morning work","category":"job","skipped":false,"done":true},
		{"id":"b2","time":"11:00","duration":60,"title":"Mid-morning","category":"job","skipped":false}
	]`
	factoryDayPlan(ds, time.Now().Format("2006-01-02"), "accepted", []byte(originalBlocks))

	mp.planChatFn = func(_ context.Context, input planner.PlanChatInput) (*planner.PlanChatOutput, error) {
		return &planner.PlanChatOutput{
			Blocks:       []planner.Block{{ID: "b5", Time: "23:00", Duration: 60, Title: "New block", Category: "job"}},
			RawResponses: []string{`[{"id":"b5"}]`},
		}, nil
	}

	today := model.Date{Time: time.Now().Truncate(24 * time.Hour)}
	mutation := r.Mutation()

	// Replan
	result, err := mutation.SendPlanMessage(context.Background(), today, "Something came up")
	if err != nil {
		t.Fatalf("SendPlanMessage: %v", err)
	}
	if !result.CanRevert {
		t.Error("expected canRevert to be true after replanning")
	}

	// Revert
	reverted, err := mutation.RevertPlan(context.Background(), today)
	if err != nil {
		t.Fatalf("RevertPlan: %v", err)
	}
	if reverted.Status != model.PlanStatusAccepted {
		t.Errorf("expected ACCEPTED after revert, got %v", reverted.Status)
	}
	if len(reverted.Blocks) != 2 {
		t.Errorf("expected 2 blocks after revert, got %d", len(reverted.Blocks))
	}
	if reverted.CanRevert {
		t.Error("expected canRevert to be false after reverting")
	}
}

// Test: revert on plan with no previous state returns error
func TestRevertPlan_NoPreviousState(t *testing.T) {
	store := newMockDayPlanStore()
	factoryDayPlan(store, "2026-03-05", "draft", threeBlocksJSON())
	r := newDayPlanResolver(store)

	date, _ := model.UnmarshalDate("2026-03-05")
	_, err := r.Mutation().RevertPlan(context.Background(), date)
	if err == nil {
		t.Fatal("expected error when no previous state")
	}
}

// Suppress unused import warnings
var (
	_ = json.Marshal
	_ = pgtype.UUID{}
)
