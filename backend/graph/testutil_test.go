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
