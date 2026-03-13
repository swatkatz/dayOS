package graph_test

import (
	"context"
	"testing"

	"dayos/db"
	"dayos/graph"
	"dayos/graph/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func newTaskResolver(store *mockTaskStore) *graph.Resolver {
	return &graph.Resolver{TaskStore: store}
}

// Test Anchor 1: Create task with defaults
func TestCreateTask_Defaults(t *testing.T) {
	store := newMockTaskStore()
	r := newTaskResolver(store)

	result, err := r.Mutation().CreateTask(context.Background(), model.CreateTaskInput{
		Title:    "Apply to Stripe",
		Category: model.CategoryJob,
		Priority: model.PriorityHigh,
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	if result.ID == uuid.Nil {
		t.Error("expected non-nil UUID")
	}
	if result.Title != "Apply to Stripe" {
		t.Errorf("title = %q, want %q", result.Title, "Apply to Stripe")
	}
	if result.EstimatedMinutes == nil || *result.EstimatedMinutes != 60 {
		t.Errorf("estimatedMinutes = %v, want 60", result.EstimatedMinutes)
	}
	if result.ActualMinutes != 0 {
		t.Errorf("actualMinutes = %d, want 0", result.ActualMinutes)
	}
	if result.TimesDeferred != 0 {
		t.Errorf("timesDeferred = %d, want 0", result.TimesDeferred)
	}
	if result.IsCompleted {
		t.Error("expected isCompleted = false")
	}
}

// Test Anchor 2: Complete subtask auto-completes parent when all siblings done
func TestCompleteTask_AutoCompletesParent(t *testing.T) {
	store := newMockTaskStore()
	r := newTaskResolver(store)
	ctx := context.Background()

	// Create parent (no estimated_minutes for parent tasks)
	parent := factoryTask(store, "Interview prep", func(p *db.CreateTaskParams) {
		p.EstimatedMinutes = nil
	})

	parentID := uuidToPgtypeTest(parent.ID)
	// Create 2 subtasks
	sub1 := factoryTask(store, "Subtask 1", func(p *db.CreateTaskParams) {
		p.ParentID = parentID
	})
	factoryTask(store, "Subtask 2", func(p *db.CreateTaskParams) {
		p.ParentID = parentID
	})

	// Complete sub1
	_, err := r.Mutation().CompleteTask(ctx, uuid.UUID(sub1.ID.Bytes))
	if err != nil {
		t.Fatalf("CompleteTask sub1: %v", err)
	}

	// Parent should still be incomplete (sub2 not done)
	parentTask, _ := store.GetTask(ctx, parent.ID)
	if parentTask.IsCompleted != nil && *parentTask.IsCompleted {
		t.Fatal("parent should not be completed yet")
	}

	// Now complete sub2 — find it
	subs, _ := store.ListSubtasks(ctx, parentID)
	var sub2ID pgtype.UUID
	for _, s := range subs {
		if s.Title == "Subtask 2" {
			sub2ID = s.ID
		}
	}

	result, err := r.Mutation().CompleteTask(ctx, uuid.UUID(sub2ID.Bytes))
	if err != nil {
		t.Fatalf("CompleteTask sub2: %v", err)
	}
	if !result.IsCompleted {
		t.Error("subtask 2 should be completed")
	}

	// Parent should now be auto-completed
	parentTask, _ = store.GetTask(ctx, parent.ID)
	if parentTask.IsCompleted == nil || !*parentTask.IsCompleted {
		t.Error("parent should be auto-completed after all subtasks done")
	}
}

// Test Anchor 3: CompleteTask on parent returns error
func TestCompleteTask_ParentReturnsError(t *testing.T) {
	store := newMockTaskStore()
	r := newTaskResolver(store)
	ctx := context.Background()

	parent := factoryTask(store, "Parent task", func(p *db.CreateTaskParams) {
		p.EstimatedMinutes = nil
	})
	factoryTask(store, "Child", func(p *db.CreateTaskParams) {
		p.ParentID = uuidToPgtypeTest(parent.ID)
	})

	_, err := r.Mutation().CompleteTask(ctx, uuid.UUID(parent.ID.Bytes))
	if err == nil {
		t.Fatal("expected error when completing parent task directly, got nil")
	}
}

// Test Anchor 4: Filter by category + includeCompleted
func TestTasks_FilterByCategoryAndCompleted(t *testing.T) {
	store := newMockTaskStore()
	r := newTaskResolver(store)
	ctx := context.Background()

	factoryTask(store, "Interview task 1", func(p *db.CreateTaskParams) {
		p.Category = "interview"
	})
	factoryTask(store, "Interview task 2 (done)", func(p *db.CreateTaskParams) {
		p.Category = "interview"
	})
	factoryTask(store, "Job task", func(p *db.CreateTaskParams) {
		p.Category = "job"
	})

	// Complete the second interview task
	subs, _ := store.ListTasks(ctx, db.ListTasksParams{})
	for _, s := range subs {
		if s.Title == "Interview task 2 (done)" {
			store.CompleteTask(ctx, s.ID)
		}
	}

	cat := model.CategoryInterview
	falseVal := false
	result, err := r.Query().Tasks(ctx, &cat, &falseVal)
	if err != nil {
		t.Fatalf("Tasks: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 incomplete interview task, got %d", len(result))
	}
	if result[0].Title != "Interview task 1" {
		t.Errorf("title = %q, want %q", result[0].Title, "Interview task 1")
	}
}

// Test Anchor 5: Delete parent cascades to subtasks
func TestDeleteTask_CascadesSubtasks(t *testing.T) {
	store := newMockTaskStore()
	r := newTaskResolver(store)
	ctx := context.Background()

	parent := factoryTask(store, "Parent", func(p *db.CreateTaskParams) {
		p.EstimatedMinutes = nil
	})
	parentID := uuidToPgtypeTest(parent.ID)
	factoryTask(store, "Sub 1", func(p *db.CreateTaskParams) {
		p.ParentID = parentID
	})
	factoryTask(store, "Sub 2", func(p *db.CreateTaskParams) {
		p.ParentID = parentID
	})
	factoryTask(store, "Sub 3", func(p *db.CreateTaskParams) {
		p.ParentID = parentID
	})

	ok, err := r.Mutation().DeleteTask(ctx, uuid.UUID(parent.ID.Bytes))
	if err != nil {
		t.Fatalf("DeleteTask: %v", err)
	}
	if !ok {
		t.Error("expected true from DeleteTask")
	}

	// Verify all gone
	_, err = store.GetTask(ctx, parent.ID)
	if err == nil {
		t.Error("parent should be deleted")
	}

	subs, _ := store.ListSubtasks(ctx, parentID)
	if len(subs) != 0 {
		t.Errorf("expected 0 subtasks after cascade delete, got %d", len(subs))
	}
}

// Test Anchor 6: Deadline validation — HARD without date
func TestCreateTask_DeadlineValidation(t *testing.T) {
	store := newMockTaskStore()
	r := newTaskResolver(store)

	hard := model.DeadlineTypeHard
	_, err := r.Mutation().CreateTask(context.Background(), model.CreateTaskInput{
		Title:        "Test deadline",
		Category:     model.CategoryJob,
		Priority:     model.PriorityHigh,
		DeadlineType: &hard,
	})
	if err == nil {
		t.Fatal("expected validation error for HARD deadline without date, got nil")
	}

	horizon := model.DeadlineTypeHorizon
	_, err = r.Mutation().CreateTask(context.Background(), model.CreateTaskInput{
		Title:        "Test horizon",
		Category:     model.CategoryJob,
		Priority:     model.PriorityHigh,
		DeadlineType: &horizon,
	})
	if err == nil {
		t.Fatal("expected validation error for HORIZON deadline without days, got nil")
	}
}

// Test Anchor 7: Nested subtask (parentId of a subtask) returns error
func TestCreateTask_NestedSubtaskError(t *testing.T) {
	store := newMockTaskStore()
	r := newTaskResolver(store)

	parent := factoryTask(store, "Parent")
	parentID := uuidToPgtypeTest(parent.ID)
	sub := factoryTask(store, "Subtask", func(p *db.CreateTaskParams) {
		p.ParentID = parentID
	})

	subID := uuid.UUID(sub.ID.Bytes)
	_, err := r.Mutation().CreateTask(context.Background(), model.CreateTaskInput{
		Title:    "Nested subtask",
		Category: model.CategoryJob,
		Priority: model.PriorityHigh,
		ParentID: &subID,
	})
	if err == nil {
		t.Fatal("expected error for nested subtask, got nil")
	}
}

// Test: updateTask with isCompleted applies completion logic
func TestUpdateTask_IsCompleted(t *testing.T) {
	store := newMockTaskStore()
	r := newTaskResolver(store)
	ctx := context.Background()

	task := factoryTask(store, "Standalone task")

	trueVal := true
	result, err := r.Mutation().UpdateTask(ctx, uuid.UUID(task.ID.Bytes), model.UpdateTaskInput{
		IsCompleted: &trueVal,
	})
	if err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}
	if !result.IsCompleted {
		t.Error("expected isCompleted = true after update")
	}

	// Uncomplete
	falseVal := false
	result, err = r.Mutation().UpdateTask(ctx, uuid.UUID(task.ID.Bytes), model.UpdateTaskInput{
		IsCompleted: &falseVal,
	})
	if err != nil {
		t.Fatalf("UpdateTask uncomplete: %v", err)
	}
	if result.IsCompleted {
		t.Error("expected isCompleted = false after uncomplete")
	}
}

// Test: deleteTask with non-existent ID returns error
func TestDeleteTask_NonExistent(t *testing.T) {
	store := newMockTaskStore()
	r := newTaskResolver(store)

	_, err := r.Mutation().DeleteTask(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error deleting non-existent task, got nil")
	}
}

// Test: createTask with routineId sets isRoutine=true
func TestCreateTask_WithRoutine(t *testing.T) {
	store := newMockTaskStore()
	r := newTaskResolver(store)

	routineID := uuid.New()
	result, err := r.Mutation().CreateTask(context.Background(), model.CreateTaskInput{
		Title:     "Morning exercise",
		Category:  model.CategoryExercise,
		Priority:  model.PriorityMedium,
		RoutineID: &routineID,
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if !result.IsRoutine {
		t.Error("expected isRoutine = true when routineId is set")
	}
}

// Test: Tasks query populates subtasks on parent tasks
func TestTasks_SubtasksPopulated(t *testing.T) {
	store := newMockTaskStore()
	r := newTaskResolver(store)
	ctx := context.Background()

	parent := factoryTask(store, "Interview prep", func(p *db.CreateTaskParams) {
		p.EstimatedMinutes = nil
	})
	parentID := uuidToPgtypeTest(parent.ID)
	factoryTask(store, "LC easy problems", func(p *db.CreateTaskParams) {
		p.ParentID = parentID
	})
	factoryTask(store, "System design review", func(p *db.CreateTaskParams) {
		p.ParentID = parentID
	})

	result, err := r.Query().Tasks(ctx, nil, nil)
	if err != nil {
		t.Fatalf("Tasks: %v", err)
	}

	// Find the parent in the result
	var parentTask *model.Task
	for _, task := range result {
		if task.Title == "Interview prep" {
			parentTask = task
			break
		}
	}
	if parentTask == nil {
		t.Fatal("parent task not found in result")
	}
	if len(parentTask.Subtasks) != 2 {
		t.Fatalf("expected 2 subtasks, got %d", len(parentTask.Subtasks))
	}

	titles := map[string]bool{}
	for _, s := range parentTask.Subtasks {
		titles[s.Title] = true
	}
	if !titles["LC easy problems"] || !titles["System design review"] {
		t.Errorf("unexpected subtask titles: %v", titles)
	}
}

func uuidToPgtypeTest(id pgtype.UUID) pgtype.UUID {
	return id
}
