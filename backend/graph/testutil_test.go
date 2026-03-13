package graph_test

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"dayos/db"
	"dayos/planner"

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

func (m *mockTaskStore) IncrementTimesDeferred(_ context.Context, id pgtype.UUID) (db.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	t, ok := m.tasks[id]
	if !ok {
		return db.Task{}, fmt.Errorf("no rows in result set")
	}
	if t.TimesDeferred == nil {
		zero := int32(0)
		t.TimesDeferred = &zero
	}
	v := *t.TimesDeferred + 1
	t.TimesDeferred = &v
	t.LastDeferredAt = pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
	m.tasks[id] = t
	return t, nil
}

func (m *mockTaskStore) ListSchedulableTasks(_ context.Context) ([]db.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []db.Task
	for _, id := range m.order {
		t := m.tasks[id]
		if t.IsCompleted != nil && *t.IsCompleted {
			continue
		}
		// subtasks OR standalone with estimated_minutes
		if t.ParentID.Valid || (!t.ParentID.Valid && t.EstimatedMinutes != nil) {
			result = append(result, t)
		}
	}
	return result, nil
}

// --- Mock ContextStore ---

type mockContextStore struct {
	mu      sync.Mutex
	entries map[pgtype.UUID]db.ContextEntry
	order   []pgtype.UUID
}

func newMockContextStore() *mockContextStore {
	return &mockContextStore{
		entries: make(map[pgtype.UUID]db.ContextEntry),
	}
}

func (m *mockContextStore) ListContextEntries(_ context.Context, category *string) ([]db.ContextEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []db.ContextEntry
	for _, id := range m.order {
		e := m.entries[id]
		if category != nil && e.Category != *category {
			continue
		}
		result = append(result, e)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Category != result[j].Category {
			return result[i].Category < result[j].Category
		}
		return result[i].Key < result[j].Key
	})
	return result, nil
}

func (m *mockContextStore) GetContextEntry(_ context.Context, id pgtype.UUID) (db.ContextEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	e, ok := m.entries[id]
	if !ok {
		return db.ContextEntry{}, fmt.Errorf("no rows in result set")
	}
	return e, nil
}

func (m *mockContextStore) UpsertContextEntry(_ context.Context, arg db.UpsertContextEntryParams) (db.ContextEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for existing entry with same category+key (upsert behavior)
	for _, id := range m.order {
		e := m.entries[id]
		if e.Category == arg.Category && e.Key == arg.Key {
			e.Value = arg.Value
			m.entries[id] = e
			return e, nil
		}
	}

	// Insert new
	id := pgtype.UUID{Bytes: uuid.New(), Valid: true}
	active := true
	e := db.ContextEntry{
		ID:       id,
		Category: arg.Category,
		Key:      arg.Key,
		Value:    arg.Value,
		IsActive: &active,
	}
	m.entries[id] = e
	m.order = append(m.order, id)
	return e, nil
}

func (m *mockContextStore) ToggleContextEntry(_ context.Context, arg db.ToggleContextEntryParams) (db.ContextEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	e, ok := m.entries[arg.ID]
	if !ok {
		return db.ContextEntry{}, fmt.Errorf("no rows in result set")
	}
	e.IsActive = arg.IsActive
	m.entries[arg.ID] = e
	return e, nil
}

func (m *mockContextStore) DeleteContextEntry(_ context.Context, id pgtype.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.entries[id]; !ok {
		return fmt.Errorf("no rows in result set")
	}
	delete(m.entries, id)
	for i, oid := range m.order {
		if oid == id {
			m.order = append(m.order[:i], m.order[i+1:]...)
			break
		}
	}
	return nil
}

// --- Mock DayPlanStore ---

type mockDayPlanStore struct {
	mu       sync.Mutex
	plans    map[pgtype.UUID]db.DayPlan
	messages map[pgtype.UUID][]db.PlanMessage // keyed by plan_id
	order    []pgtype.UUID
}

func newMockDayPlanStore() *mockDayPlanStore {
	return &mockDayPlanStore{
		plans:    make(map[pgtype.UUID]db.DayPlan),
		messages: make(map[pgtype.UUID][]db.PlanMessage),
	}
}

func (m *mockDayPlanStore) GetDayPlanByDate(_ context.Context, planDate pgtype.Date) (db.DayPlan, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, id := range m.order {
		p := m.plans[id]
		if p.PlanDate.Time.Equal(planDate.Time) {
			return p, nil
		}
	}
	return db.DayPlan{}, fmt.Errorf("no rows in result set")
}

func (m *mockDayPlanStore) GetDayPlanByID(_ context.Context, id pgtype.UUID) (db.DayPlan, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	p, ok := m.plans[id]
	if !ok {
		return db.DayPlan{}, fmt.Errorf("no rows in result set")
	}
	return p, nil
}

func (m *mockDayPlanStore) CreateDayPlan(_ context.Context, arg db.CreateDayPlanParams) (db.DayPlan, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check unique constraint on plan_date
	for _, id := range m.order {
		p := m.plans[id]
		if p.PlanDate.Time.Equal(arg.PlanDate.Time) {
			return db.DayPlan{}, fmt.Errorf("duplicate key value violates unique constraint")
		}
	}

	id := pgtype.UUID{Bytes: uuid.New(), Valid: true}
	p := db.DayPlan{
		ID:       id,
		PlanDate: arg.PlanDate,
		Status:   arg.Status,
		Blocks:   arg.Blocks,
	}
	m.plans[id] = p
	m.order = append(m.order, id)
	return p, nil
}

func (m *mockDayPlanStore) UpdateDayPlanBlocks(_ context.Context, arg db.UpdateDayPlanBlocksParams) (db.DayPlan, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	p, ok := m.plans[arg.ID]
	if !ok {
		return db.DayPlan{}, fmt.Errorf("no rows in result set")
	}
	p.Blocks = arg.Blocks
	m.plans[arg.ID] = p
	return p, nil
}

func (m *mockDayPlanStore) UpdateDayPlanStatus(_ context.Context, arg db.UpdateDayPlanStatusParams) (db.DayPlan, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	p, ok := m.plans[arg.ID]
	if !ok {
		return db.DayPlan{}, fmt.Errorf("no rows in result set")
	}
	p.Status = arg.Status
	m.plans[arg.ID] = p
	return p, nil
}

func (m *mockDayPlanStore) RecentPlans(_ context.Context, limit int32) ([]db.DayPlan, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Collect all plans and sort by plan_date DESC
	plans := make([]db.DayPlan, 0, len(m.plans))
	for _, id := range m.order {
		plans = append(plans, m.plans[id])
	}
	sort.Slice(plans, func(i, j int) bool {
		return plans[i].PlanDate.Time.After(plans[j].PlanDate.Time)
	})
	if int(limit) < len(plans) {
		plans = plans[:limit]
	}
	return plans, nil
}

func (m *mockDayPlanStore) GetPlanMessages(_ context.Context, planID pgtype.UUID) ([]db.PlanMessage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	msgs := m.messages[planID]
	return msgs, nil
}

func (m *mockDayPlanStore) CreatePlanMessage(_ context.Context, arg db.CreatePlanMessageParams) (db.PlanMessage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	msg := db.PlanMessage{
		ID:      pgtype.UUID{Bytes: uuid.New(), Valid: true},
		PlanID:  arg.PlanID,
		Role:    arg.Role,
		Content: arg.Content,
	}
	m.messages[arg.PlanID] = append(m.messages[arg.PlanID], msg)
	return msg, nil
}

// --- Mock TaskConversationStore ---

type mockTaskConversationStore struct {
	mu            sync.Mutex
	conversations map[pgtype.UUID]db.TaskConversation
	messages      map[pgtype.UUID][]db.TaskMessage // keyed by conversation_id
	order         []pgtype.UUID
}

func newMockTaskConversationStore() *mockTaskConversationStore {
	return &mockTaskConversationStore{
		conversations: make(map[pgtype.UUID]db.TaskConversation),
		messages:      make(map[pgtype.UUID][]db.TaskMessage),
	}
}

func (m *mockTaskConversationStore) CreateTaskConversation(_ context.Context) (db.TaskConversation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := pgtype.UUID{Bytes: uuid.New(), Valid: true}
	c := db.TaskConversation{
		ID:     id,
		Status: "active",
	}
	m.conversations[id] = c
	m.order = append(m.order, id)
	return c, nil
}

func (m *mockTaskConversationStore) GetTaskConversation(_ context.Context, id pgtype.UUID) (db.TaskConversation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	c, ok := m.conversations[id]
	if !ok {
		return db.TaskConversation{}, fmt.Errorf("no rows in result set")
	}
	return c, nil
}

func (m *mockTaskConversationStore) UpdateTaskConversationStatus(_ context.Context, arg db.UpdateTaskConversationStatusParams) (db.TaskConversation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	c, ok := m.conversations[arg.ID]
	if !ok {
		return db.TaskConversation{}, fmt.Errorf("no rows in result set")
	}
	c.Status = arg.Status
	m.conversations[arg.ID] = c
	return c, nil
}

func (m *mockTaskConversationStore) LinkTaskConversationParent(_ context.Context, arg db.LinkTaskConversationParentParams) (db.TaskConversation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	c, ok := m.conversations[arg.ID]
	if !ok {
		return db.TaskConversation{}, fmt.Errorf("no rows in result set")
	}
	c.ParentTaskID = arg.ParentTaskID
	m.conversations[arg.ID] = c
	return c, nil
}

func (m *mockTaskConversationStore) CreateTaskMessage(_ context.Context, arg db.CreateTaskMessageParams) (db.TaskMessage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	msg := db.TaskMessage{
		ID:             pgtype.UUID{Bytes: uuid.New(), Valid: true},
		ConversationID: arg.ConversationID,
		Role:           arg.Role,
		Content:        arg.Content,
	}
	m.messages[arg.ConversationID] = append(m.messages[arg.ConversationID], msg)
	return msg, nil
}

func (m *mockTaskConversationStore) GetTaskMessages(_ context.Context, conversationID pgtype.UUID) ([]db.TaskMessage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.messages[conversationID], nil
}

// --- Mock PlannerService ---

type mockPlanner struct {
	planChatFn func(ctx context.Context, input planner.PlanChatInput) (*planner.PlanChatOutput, error)
	taskChatFn func(ctx context.Context, history []planner.Message, userMessage string) (*planner.TaskChatOutput, error)
}

func (m *mockPlanner) PlanChat(ctx context.Context, input planner.PlanChatInput) (*planner.PlanChatOutput, error) {
	if m.planChatFn != nil {
		return m.planChatFn(ctx, input)
	}
	return &planner.PlanChatOutput{
		Blocks:       []planner.Block{},
		RawResponses: []string{"[]"},
	}, nil
}

func (m *mockPlanner) TaskChat(ctx context.Context, history []planner.Message, userMessage string) (*planner.TaskChatOutput, error) {
	if m.taskChatFn != nil {
		return m.taskChatFn(ctx, history, userMessage)
	}
	return &planner.TaskChatOutput{
		RawResponse: `{"status": "question", "message": "What's the scope?"}`,
	}, nil
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

func factoryContextEntry(store *mockContextStore, category, key, value string) db.ContextEntry {
	e, err := store.UpsertContextEntry(context.Background(), db.UpsertContextEntryParams{
		Category: category,
		Key:      key,
		Value:    value,
	})
	if err != nil {
		panic(fmt.Sprintf("factoryContextEntry: %v", err))
	}
	return e
}

func seedContextEntries(store *mockContextStore) {
	factoryContextEntry(store, "life", "baby", "6-month-old daughter.")
	factoryContextEntry(store, "life", "family", "Partner Elijah.")
	factoryContextEntry(store, "constraints", "work_window", "Deep focus work: 9am-4pm only.")
	factoryContextEntry(store, "constraints", "location", "Toronto, Canada.")
	factoryContextEntry(store, "constraints", "energy", "Cap deep cognitive work at 5h/day max.")
	factoryContextEntry(store, "constraints", "dinner_prep", "Dinner is always prepped after 16:00.")
	factoryContextEntry(store, "constraints", "evening_window", "Light evening window ~20:00-22:00.")
	factoryContextEntry(store, "equipment", "kitchen", "Full Indian kitchen setup.")
	factoryContextEntry(store, "preferences", "planning_style", "Time-blocked days.")
}

func factoryDayPlan(store *mockDayPlanStore, dateStr string, status string, blocksJSON []byte) db.DayPlan {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		panic(fmt.Sprintf("factoryDayPlan: bad date %q: %v", dateStr, err))
	}
	p, err := store.CreateDayPlan(context.Background(), db.CreateDayPlanParams{
		PlanDate: pgtype.Date{Time: t, Valid: true},
		Status:   status,
		Blocks:   blocksJSON,
	})
	if err != nil {
		panic(fmt.Sprintf("factoryDayPlan: %v", err))
	}
	return p
}

func factoryPlanMessage(store *mockDayPlanStore, planID pgtype.UUID, role, content string) db.PlanMessage {
	msg, err := store.CreatePlanMessage(context.Background(), db.CreatePlanMessageParams{
		PlanID:  planID,
		Role:    role,
		Content: content,
	})
	if err != nil {
		panic(fmt.Sprintf("factoryPlanMessage: %v", err))
	}
	return msg
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
