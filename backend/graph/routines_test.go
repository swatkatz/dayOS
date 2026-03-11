package graph_test

import (
	"context"
	"testing"

	"dayos/db"
	"dayos/graph"
	"dayos/graph/model"

	"github.com/google/uuid"
)

func newRoutineResolver(store *mockRoutineStore) *graph.Resolver {
	return &graph.Resolver{RoutineStore: store}
}

// Test Anchor 1: Create routine returns isActive=true and a generated UUID
func TestCreateRoutine_Defaults(t *testing.T) {
	store := newMockRoutineStore()
	r := newRoutineResolver(store)

	morning := model.TimeOfDayMorning
	dur := 45
	result, err := r.Mutation().CreateRoutine(context.Background(), model.CreateRoutineInput{
		Title:                "Daily exercise",
		Category:             model.CategoryExercise,
		Frequency:            model.FrequencyDaily,
		PreferredTimeOfDay:   &morning,
		PreferredDurationMin: &dur,
	})
	if err != nil {
		t.Fatalf("CreateRoutine: %v", err)
	}

	if result.ID == uuid.Nil {
		t.Error("expected non-nil UUID")
	}
	if result.Title != "Daily exercise" {
		t.Errorf("title = %q, want %q", result.Title, "Daily exercise")
	}
	if !result.IsActive {
		t.Error("expected isActive = true")
	}
	if result.PreferredDurationMin == nil || *result.PreferredDurationMin != 45 {
		t.Errorf("preferredDurationMin = %v, want 45", result.PreferredDurationMin)
	}
}

// Test Anchor 2: WEEKLY with null daysOfWeek returns validation error
func TestCreateRoutine_WeeklyNoDays(t *testing.T) {
	store := newMockRoutineStore()
	r := newRoutineResolver(store)

	_, err := r.Mutation().CreateRoutine(context.Background(), model.CreateRoutineInput{
		Title:     "Weekend prep",
		Category:  model.CategoryMeal,
		Frequency: model.FrequencyWeekly,
	})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

// Test Anchor 3: ListRoutinesForDay filters by day-of-week
func TestListRoutinesForDay(t *testing.T) {
	store := newMockRoutineStore()
	ctx := context.Background()

	// daily routine
	factoryRoutine(store, "Daily standup")

	// weekdays routine
	factoryRoutine(store, "Code review", func(p *db.UpsertRoutineParams) {
		p.Frequency = "weekdays"
	})

	// custom weekend-only routine
	factoryRoutine(store, "Weekend hike", func(p *db.UpsertRoutineParams) {
		p.Frequency = "custom"
		p.DaysOfWeek = []int32{0, 6}
	})

	// Wednesday = day 3
	routines, err := store.ListRoutinesForDay(ctx, 3)
	if err != nil {
		t.Fatalf("ListRoutinesForDay: %v", err)
	}

	names := make([]string, len(routines))
	for i, r := range routines {
		names[i] = r.Title
	}

	if len(routines) != 2 {
		t.Fatalf("expected 2 routines for Wednesday, got %d: %v", len(routines), names)
	}

	// Verify weekend-only not included
	for _, r := range routines {
		if r.Title == "Weekend hike" {
			t.Error("weekend-only routine should not appear on Wednesday")
		}
	}
}

// Test Anchor 4: Deactivate routine, then activeOnly query excludes it
func TestUpdateRoutine_Deactivate(t *testing.T) {
	store := newMockRoutineStore()
	r := newRoutineResolver(store)
	ctx := context.Background()

	created := factoryRoutine(store, "Morning yoga")

	falseVal := false
	updated, err := r.Mutation().UpdateRoutine(ctx, uuid.UUID(created.ID.Bytes), model.UpdateRoutineInput{
		IsActive: &falseVal,
	})
	if err != nil {
		t.Fatalf("UpdateRoutine: %v", err)
	}
	if updated.IsActive {
		t.Error("expected isActive = false after update")
	}

	trueVal := true
	list, err := r.Query().Routines(ctx, &trueVal)
	if err != nil {
		t.Fatalf("Routines: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected 0 active routines, got %d", len(list))
	}
}

// Test Anchor 5: Delete routine succeeds; non-existent ID returns error
// Note: ON DELETE SET NULL behavior is tested in db/migrations_test.go (TestSetNullOnRoutineDelete)
func TestDeleteRoutine(t *testing.T) {
	store := newMockRoutineStore()
	r := newRoutineResolver(store)
	ctx := context.Background()

	created := factoryRoutine(store, "Throwaway routine")

	ok, err := r.Mutation().DeleteRoutine(ctx, uuid.UUID(created.ID.Bytes))
	if err != nil {
		t.Fatalf("DeleteRoutine: %v", err)
	}
	if !ok {
		t.Error("expected true from DeleteRoutine")
	}

	// Deleting again should error (non-existent)
	_, err = r.Mutation().DeleteRoutine(ctx, uuid.UUID(created.ID.Bytes))
	if err == nil {
		t.Fatal("expected error deleting non-existent routine, got nil")
	}
}
