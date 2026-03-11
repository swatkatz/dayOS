package graph_test

import (
	"context"
	"fmt"
	"sync"

	"dayos/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// --- Mock RoutineStore ---

type mockRoutineStore struct {
	mu       sync.Mutex
	routines map[pgtype.UUID]db.Routine
	order    []pgtype.UUID
}

func newMockRoutineStore() *mockRoutineStore {
	return &mockRoutineStore{
		routines: make(map[pgtype.UUID]db.Routine),
	}
}

func boolPtr(b bool) *bool { return &b }

func (m *mockRoutineStore) UpsertRoutine(_ context.Context, arg db.UpsertRoutineParams) (db.Routine, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// If ID is set, this is an update — replace existing
	id := arg.ID
	if !id.Valid {
		id = pgtype.UUID{Bytes: uuid.New(), Valid: true}
	}

	r := db.Routine{
		ID:                   id,
		Title:                arg.Title,
		Category:             arg.Category,
		Frequency:            arg.Frequency,
		DaysOfWeek:           arg.DaysOfWeek,
		PreferredTimeOfDay:   arg.PreferredTimeOfDay,
		PreferredDurationMin: arg.PreferredDurationMin,
		Notes:                arg.Notes,
		IsActive:             arg.IsActive,
	}
	if r.IsActive == nil {
		r.IsActive = boolPtr(true)
	}

	if _, exists := m.routines[id]; !exists {
		m.order = append(m.order, id)
	}
	m.routines[id] = r
	return r, nil
}

func (m *mockRoutineStore) GetRoutine(_ context.Context, id pgtype.UUID) (db.Routine, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	r, ok := m.routines[id]
	if !ok {
		return db.Routine{}, fmt.Errorf("no rows in result set")
	}
	return r, nil
}

func (m *mockRoutineStore) ListRoutines(_ context.Context, activeOnly *bool) ([]db.Routine, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []db.Routine
	for _, id := range m.order {
		r := m.routines[id]
		if activeOnly != nil && *activeOnly && (r.IsActive == nil || !*r.IsActive) {
			continue
		}
		result = append(result, r)
	}
	return result, nil
}

func (m *mockRoutineStore) ListRoutinesForDay(_ context.Context, dayOfWeek int32) ([]db.Routine, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []db.Routine
	for _, id := range m.order {
		r := m.routines[id]
		if r.IsActive == nil || !*r.IsActive {
			continue
		}
		switch r.Frequency {
		case "daily":
			result = append(result, r)
		case "weekdays":
			if dayOfWeek >= 1 && dayOfWeek <= 5 {
				result = append(result, r)
			}
		case "weekly", "custom":
			for _, d := range r.DaysOfWeek {
				if d == dayOfWeek {
					result = append(result, r)
					break
				}
			}
		}
	}
	return result, nil
}

func (m *mockRoutineStore) DeleteRoutine(_ context.Context, id pgtype.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.routines[id]; !ok {
		return fmt.Errorf("no rows in result set")
	}
	delete(m.routines, id)
	return nil
}

// --- Mock TaskStore ---

type mockTaskStore struct {
	mu    sync.Mutex
	tasks map[pgtype.UUID]db.Task
	order []pgtype.UUID
}

func newMockTaskStore() *mockTaskStore {
	return &mockTaskStore{
		tasks: make(map[pgtype.UUID]db.Task),
	}
}

func int32Ptr(v int32) *int32 { return &v }

func (m *mockTaskStore) CreateTask(_ context.Context, arg db.CreateTaskParams) (db.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := pgtype.UUID{Bytes: uuid.New(), Valid: true}
	zero := int32(0)
	falseVal := false
	t := db.Task{
		ID:               id,
		Title:            arg.Title,
		Category:         arg.Category,
		Priority:         arg.Priority,
		ParentID:         arg.ParentID,
		EstimatedMinutes: arg.EstimatedMinutes,
		ActualMinutes:    &zero,
		DeadlineType:     arg.DeadlineType,
		DeadlineDate:     arg.DeadlineDate,
		DeadlineDays:     arg.DeadlineDays,
		Notes:            arg.Notes,
		IsRoutine:        arg.IsRoutine,
		RoutineID:        arg.RoutineID,
		TimesDeferred:    &zero,
		IsCompleted:      &falseVal,
	}

	m.tasks[id] = t
	m.order = append(m.order, id)
	return t, nil
}

func (m *mockTaskStore) GetTask(_ context.Context, id pgtype.UUID) (db.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	t, ok := m.tasks[id]
	if !ok {
		return db.Task{}, fmt.Errorf("no rows in result set")
	}
	return t, nil
}

func (m *mockTaskStore) ListTasks(_ context.Context, arg db.ListTasksParams) ([]db.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []db.Task
	for _, id := range m.order {
		t := m.tasks[id]
		// top-level only
		if t.ParentID.Valid {
			continue
		}
		if arg.Category != nil && t.Category != *arg.Category {
			continue
		}
		if arg.IncludeCompleted == nil || !*arg.IncludeCompleted {
			if t.IsCompleted != nil && *t.IsCompleted {
				continue
			}
		}
		result = append(result, t)
	}
	return result, nil
}

func (m *mockTaskStore) ListSubtasks(_ context.Context, parentID pgtype.UUID) ([]db.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []db.Task
	for _, id := range m.order {
		t := m.tasks[id]
		if t.ParentID == parentID {
			result = append(result, t)
		}
	}
	return result, nil
}

func (m *mockTaskStore) UpdateTask(_ context.Context, arg db.UpdateTaskParams) (db.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	t, ok := m.tasks[arg.ID]
	if !ok {
		return db.Task{}, fmt.Errorf("no rows in result set")
	}
	if arg.Title != nil {
		t.Title = *arg.Title
	}
	if arg.Category != nil {
		t.Category = *arg.Category
	}
	if arg.Priority != nil {
		t.Priority = *arg.Priority
	}
	if arg.EstimatedMinutes != nil {
		t.EstimatedMinutes = arg.EstimatedMinutes
	}
	if arg.DeadlineType != nil {
		t.DeadlineType = arg.DeadlineType
	}
	if arg.DeadlineDate.Valid {
		t.DeadlineDate = arg.DeadlineDate
	}
	if arg.DeadlineDays != nil {
		t.DeadlineDays = arg.DeadlineDays
	}
	if arg.Notes != nil {
		t.Notes = arg.Notes
	}
	if arg.RoutineID.Valid {
		t.RoutineID = arg.RoutineID
	}
	m.tasks[arg.ID] = t
	return t, nil
}

func (m *mockTaskStore) CompleteTask(_ context.Context, id pgtype.UUID) (db.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	t, ok := m.tasks[id]
	if !ok {
		return db.Task{}, fmt.Errorf("no rows in result set")
	}
	trueVal := true
	t.IsCompleted = &trueVal
	m.tasks[id] = t
	return t, nil
}

func (m *mockTaskStore) UncompleteTask(_ context.Context, id pgtype.UUID) (db.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	t, ok := m.tasks[id]
	if !ok {
		return db.Task{}, fmt.Errorf("no rows in result set")
	}
	falseVal := false
	t.IsCompleted = &falseVal
	m.tasks[id] = t
	return t, nil
}

func (m *mockTaskStore) DeleteTask(_ context.Context, id pgtype.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.tasks[id]; !ok {
		return fmt.Errorf("no rows in result set")
	}
	// cascade delete subtasks
	for _, oid := range m.order {
		t := m.tasks[oid]
		if t.ParentID == id {
			delete(m.tasks, oid)
		}
	}
	delete(m.tasks, id)
	return nil
}

func (m *mockTaskStore) CountIncompleteSubtasks(_ context.Context, parentID pgtype.UUID) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var count int64
	for _, id := range m.order {
		t := m.tasks[id]
		if t.ParentID == parentID && (t.IsCompleted == nil || !*t.IsCompleted) {
			count++
		}
	}
	return count, nil
}

// --- Factories ---

func factoryRoutine(store *mockRoutineStore, title string, overrides ...func(*db.UpsertRoutineParams)) db.Routine {
	params := db.UpsertRoutineParams{
		Title:     title,
		Category:  "exercise",
		Frequency: "daily",
	}
	for _, o := range overrides {
		o(&params)
	}
	r, err := store.UpsertRoutine(context.Background(), params)
	if err != nil {
		panic(fmt.Sprintf("factoryRoutine: %v", err))
	}
	return r
}

func factoryTask(store *mockTaskStore, title string, overrides ...func(*db.CreateTaskParams)) db.Task {
	estMin := int32(60)
	falseVal := false
	params := db.CreateTaskParams{
		Title:            title,
		Category:         "job",
		Priority:         "high",
		EstimatedMinutes: &estMin,
		IsRoutine:        &falseVal,
	}
	for _, o := range overrides {
		o(&params)
	}
	t, err := store.CreateTask(context.Background(), params)
	if err != nil {
		panic(fmt.Sprintf("factoryTask: %v", err))
	}
	return t
}
