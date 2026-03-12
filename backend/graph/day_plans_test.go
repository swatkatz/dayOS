package graph_test

import (
	"context"
	"encoding/json"
	"testing"

	"dayos/graph"
	"dayos/graph/model"

	"github.com/google/uuid"
)

func newDayPlanResolver(store *mockDayPlanStore) *graph.Resolver {
	return &graph.Resolver{DayPlanStore: store}
}

func threeBlocksJSON() []byte {
	blocks := []map[string]any{
		{"id": "block-1", "time": "09:00", "duration": 60, "title": "Interview prep", "category": "interview", "skipped": false},
		{"id": "block-2", "time": "10:00", "duration": 60, "title": "Coding", "category": "project", "skipped": false},
		{"id": "block-3", "time": "11:00", "duration": 45, "title": "Exercise", "category": "exercise", "skipped": false},
	}
	b, _ := json.Marshal(blocks)
	return b
}

// Test Anchor 1: Given no plan exists for 2026-03-05, when dayPlan(date: "2026-03-05")
// is called, then null is returned.
func TestDayPlan_NotFound(t *testing.T) {
	store := newMockDayPlanStore()
	r := newDayPlanResolver(store)

	date, _ := model.UnmarshalDate("2026-03-05")
	result, err := r.Query().DayPlan(context.Background(), date)
	if err != nil {
		t.Fatalf("DayPlan: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil, got %+v", result)
	}
}

// Test Anchor 2: Given a draft plan exists for 2026-03-05 with 3 blocks, when
// acceptPlan(date: "2026-03-05") is called, then the plan status changes to accepted
// and all 3 blocks are preserved.
func TestAcceptPlan_DraftToAccepted(t *testing.T) {
	store := newMockDayPlanStore()
	_ = factoryDayPlan(store, "2026-03-05", "draft", threeBlocksJSON())
	r := newDayPlanResolver(store)

	date, _ := model.UnmarshalDate("2026-03-05")
	result, err := r.Mutation().AcceptPlan(context.Background(), date)
	if err != nil {
		t.Fatalf("AcceptPlan: %v", err)
	}
	if result.Status != model.PlanStatusAccepted {
		t.Errorf("status = %s, want ACCEPTED", result.Status)
	}
	if len(result.Blocks) != 3 {
		t.Errorf("expected 3 blocks, got %d", len(result.Blocks))
	}
}

// Test Anchor 3: Given an accepted plan with block "block-1", when skipBlock(planId, "block-1")
// is called, then the block's skipped field is true and all other blocks remain unchanged.
func TestSkipBlock(t *testing.T) {
	store := newMockDayPlanStore()
	plan := factoryDayPlan(store, "2026-03-05", "accepted", threeBlocksJSON())
	r := newDayPlanResolver(store)

	planID := uuid.UUID(plan.ID.Bytes)
	result, err := r.Mutation().SkipBlock(context.Background(), planID, "block-1")
	if err != nil {
		t.Fatalf("SkipBlock: %v", err)
	}

	var skippedBlock *model.PlanBlock
	for _, b := range result.Blocks {
		if b.ID == "block-1" {
			skippedBlock = b
		}
	}
	if skippedBlock == nil {
		t.Fatal("block-1 not found in result")
	}
	if !skippedBlock.Skipped {
		t.Error("expected block-1 to be skipped")
	}
	// Other blocks unchanged
	for _, b := range result.Blocks {
		if b.ID != "block-1" && b.Skipped {
			t.Errorf("block %s should not be skipped", b.ID)
		}
	}
}

// Test Anchor 4: Given an accepted plan with block "block-2" having duration: 60,
// when updateBlock(planId, "block-2", { duration: 45 }) is called, then the block's
// duration is 45 and all other fields on that block are unchanged.
func TestUpdateBlock_Duration(t *testing.T) {
	store := newMockDayPlanStore()
	plan := factoryDayPlan(store, "2026-03-05", "accepted", threeBlocksJSON())
	r := newDayPlanResolver(store)

	planID := uuid.UUID(plan.ID.Bytes)
	dur := 45
	result, err := r.Mutation().UpdateBlock(context.Background(), planID, "block-2", model.UpdateBlockInput{
		Duration: &dur,
	})
	if err != nil {
		t.Fatalf("UpdateBlock: %v", err)
	}

	var block *model.PlanBlock
	for _, b := range result.Blocks {
		if b.ID == "block-2" {
			block = b
		}
	}
	if block == nil {
		t.Fatal("block-2 not found")
	}
	if block.Duration != 45 {
		t.Errorf("duration = %d, want 45", block.Duration)
	}
	if block.Title != "Coding" {
		t.Errorf("title changed unexpectedly: %s", block.Title)
	}
}

// Test Anchor 5: Given a draft plan exists, when skipBlock is called on it, then an error
// is returned indicating blocks can only be modified on accepted plans.
func TestSkipBlock_DraftPlanError(t *testing.T) {
	store := newMockDayPlanStore()
	plan := factoryDayPlan(store, "2026-03-05", "draft", threeBlocksJSON())
	r := newDayPlanResolver(store)

	planID := uuid.UUID(plan.ID.Bytes)
	_, err := r.Mutation().SkipBlock(context.Background(), planID, "block-1")
	if err == nil {
		t.Fatal("expected error for skipBlock on draft plan")
	}
	if err.Error() != "Can only skip blocks on an accepted plan" {
		t.Errorf("unexpected error: %v", err)
	}
}

// Test Anchor 6: Given an accepted plan, when updateBlock is called with a non-existent blockId,
// then an error "Block not found" is returned.
func TestUpdateBlock_NotFound(t *testing.T) {
	store := newMockDayPlanStore()
	plan := factoryDayPlan(store, "2026-03-05", "accepted", threeBlocksJSON())
	r := newDayPlanResolver(store)

	planID := uuid.UUID(plan.ID.Bytes)
	dur := 30
	_, err := r.Mutation().UpdateBlock(context.Background(), planID, "nonexistent", model.UpdateBlockInput{
		Duration: &dur,
	})
	if err == nil {
		t.Fatal("expected error for non-existent block")
	}
	if err.Error() != "Block not found" {
		t.Errorf("unexpected error: %v", err)
	}
}

// Test Anchor 7: Given 3 plans exist for different dates, when recentPlans(limit: 2) is called,
// then exactly 2 plans are returned, ordered by plan_date DESC.
func TestRecentPlans(t *testing.T) {
	store := newMockDayPlanStore()
	_ = factoryDayPlan(store, "2026-03-03", "accepted", threeBlocksJSON())
	_ = factoryDayPlan(store, "2026-03-05", "draft", threeBlocksJSON())
	_ = factoryDayPlan(store, "2026-03-04", "accepted", threeBlocksJSON())
	r := newDayPlanResolver(store)

	limit := 2
	result, err := r.Query().RecentPlans(context.Background(), &limit)
	if err != nil {
		t.Fatalf("RecentPlans: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 plans, got %d", len(result))
	}
	// Should be ordered by plan_date DESC: 2026-03-05, 2026-03-04
	if result[0].PlanDate.Format("2006-01-02") != "2026-03-05" {
		t.Errorf("first plan date = %s, want 2026-03-05", result[0].PlanDate.Format("2006-01-02"))
	}
	if result[1].PlanDate.Format("2006-01-02") != "2026-03-04" {
		t.Errorf("second plan date = %s, want 2026-03-04", result[1].PlanDate.Format("2006-01-02"))
	}
}

// Test Anchor 8: Given a plan with 2 messages (user + assistant), when dayPlan is queried,
// then both messages are returned in chronological order.
func TestDayPlan_WithMessages(t *testing.T) {
	store := newMockDayPlanStore()
	plan := factoryDayPlan(store, "2026-03-05", "draft", threeBlocksJSON())
	factoryPlanMessage(store, plan.ID, "user", "Plan my day")
	factoryPlanMessage(store, plan.ID, "assistant", "Here is your plan")
	r := newDayPlanResolver(store)

	date, _ := model.UnmarshalDate("2026-03-05")
	result, err := r.Query().DayPlan(context.Background(), date)
	if err != nil {
		t.Fatalf("DayPlan: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil plan")
	}
	if len(result.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result.Messages))
	}
	if result.Messages[0].Role != "user" {
		t.Errorf("first message role = %s, want user", result.Messages[0].Role)
	}
	if result.Messages[1].Role != "assistant" {
		t.Errorf("second message role = %s, want assistant", result.Messages[1].Role)
	}
}

// Test Anchor 9: Given an accepted plan with block "block-3", when updateBlock with
// duration: 0 is called, then error "Duration must be positive" is returned.
func TestUpdateBlock_ZeroDuration(t *testing.T) {
	store := newMockDayPlanStore()
	plan := factoryDayPlan(store, "2026-03-05", "accepted", threeBlocksJSON())
	r := newDayPlanResolver(store)

	planID := uuid.UUID(plan.ID.Bytes)
	dur := 0
	_, err := r.Mutation().UpdateBlock(context.Background(), planID, "block-3", model.UpdateBlockInput{
		Duration: &dur,
	})
	if err == nil {
		t.Fatal("expected error for zero duration")
	}
	if err.Error() != "Duration must be positive" {
		t.Errorf("unexpected error: %v", err)
	}
}

// Additional: AcceptPlan on non-existent plan returns error
func TestAcceptPlan_NoPlan(t *testing.T) {
	store := newMockDayPlanStore()
	r := newDayPlanResolver(store)

	date, _ := model.UnmarshalDate("2026-03-05")
	_, err := r.Mutation().AcceptPlan(context.Background(), date)
	if err == nil {
		t.Fatal("expected error for non-existent plan")
	}
	if err.Error() != "No plan exists for this date" {
		t.Errorf("unexpected error: %v", err)
	}
}

// Additional: AcceptPlan on already-accepted plan is idempotent
func TestAcceptPlan_Idempotent(t *testing.T) {
	store := newMockDayPlanStore()
	_ = factoryDayPlan(store, "2026-03-05", "accepted", threeBlocksJSON())
	r := newDayPlanResolver(store)

	date, _ := model.UnmarshalDate("2026-03-05")
	result, err := r.Mutation().AcceptPlan(context.Background(), date)
	if err != nil {
		t.Fatalf("AcceptPlan: %v", err)
	}
	if result.Status != model.PlanStatusAccepted {
		t.Errorf("status = %s, want ACCEPTED", result.Status)
	}
}

// Additional: updateBlock on draft plan returns error
func TestUpdateBlock_DraftPlanError(t *testing.T) {
	store := newMockDayPlanStore()
	plan := factoryDayPlan(store, "2026-03-05", "draft", threeBlocksJSON())
	r := newDayPlanResolver(store)

	planID := uuid.UUID(plan.ID.Bytes)
	dur := 30
	_, err := r.Mutation().UpdateBlock(context.Background(), planID, "block-1", model.UpdateBlockInput{
		Duration: &dur,
	})
	if err == nil {
		t.Fatal("expected error for updateBlock on draft plan")
	}
	if err.Error() != "Can only update blocks on an accepted plan" {
		t.Errorf("unexpected error: %v", err)
	}
}
