package db

import (
	"context"
	"fmt"
	"time"

	"dayos/identity"

	"github.com/jackc/pgx/v5/pgtype"
)

// ScopedQueries wraps *Queries and injects user_id from context into every call.
// It satisfies all store interfaces in graph/stores.go and calendar.GoogleAuthStore.
// Constructed once at startup — user ID is read per-request from context.
type ScopedQueries struct {
	q *Queries
}

// NewScopedQueries creates a ScopedQueries wrapping the given Queries.
func NewScopedQueries(q *Queries) *ScopedQueries {
	return &ScopedQueries{q: q}
}

func (s *ScopedQueries) userID(ctx context.Context) (pgtype.UUID, error) {
	u, ok := identity.FromContext(ctx)
	if !ok {
		return pgtype.UUID{}, fmt.Errorf("scoped query: no user in context")
	}
	return u.ID, nil
}

// --- RoutineStore ---

func (s *ScopedQueries) UpsertRoutine(ctx context.Context, arg UpsertRoutineParams) (Routine, error) {
	uid, err := s.userID(ctx)
	if err != nil {
		return Routine{}, err
	}
	arg.UserID = uid
	return s.q.UpsertRoutine(ctx, arg)
}

func (s *ScopedQueries) GetRoutine(ctx context.Context, id pgtype.UUID) (Routine, error) {
	uid, err := s.userID(ctx)
	if err != nil {
		return Routine{}, err
	}
	return s.q.GetRoutine(ctx, GetRoutineParams{ID: id, UserID: uid})
}

func (s *ScopedQueries) ListRoutines(ctx context.Context, activeOnly *bool) ([]Routine, error) {
	uid, err := s.userID(ctx)
	if err != nil {
		return nil, err
	}
	return s.q.ListRoutines(ctx, ListRoutinesParams{UserID: uid, ActiveOnly: activeOnly})
}

func (s *ScopedQueries) DeleteRoutine(ctx context.Context, id pgtype.UUID) error {
	uid, err := s.userID(ctx)
	if err != nil {
		return err
	}
	return s.q.DeleteRoutine(ctx, DeleteRoutineParams{ID: id, UserID: uid})
}

func (s *ScopedQueries) ListRoutinesForDay(ctx context.Context, dayOfWeek int32) ([]Routine, error) {
	uid, err := s.userID(ctx)
	if err != nil {
		return nil, err
	}
	return s.q.ListRoutinesForDay(ctx, ListRoutinesForDayParams{UserID: uid, DayOfWeek: dayOfWeek})
}

// --- TaskStore ---

func (s *ScopedQueries) CreateTask(ctx context.Context, arg CreateTaskParams) (Task, error) {
	uid, err := s.userID(ctx)
	if err != nil {
		return Task{}, err
	}
	arg.UserID = uid
	return s.q.CreateTask(ctx, arg)
}

func (s *ScopedQueries) GetTask(ctx context.Context, id pgtype.UUID) (Task, error) {
	uid, err := s.userID(ctx)
	if err != nil {
		return Task{}, err
	}
	return s.q.GetTask(ctx, GetTaskParams{ID: id, UserID: uid})
}

func (s *ScopedQueries) ListTasks(ctx context.Context, arg ListTasksParams) ([]Task, error) {
	uid, err := s.userID(ctx)
	if err != nil {
		return nil, err
	}
	arg.UserID = uid
	return s.q.ListTasks(ctx, arg)
}

func (s *ScopedQueries) ListSubtasks(ctx context.Context, parentID pgtype.UUID) ([]Task, error) {
	uid, err := s.userID(ctx)
	if err != nil {
		return nil, err
	}
	return s.q.ListSubtasks(ctx, ListSubtasksParams{ParentID: parentID, UserID: uid})
}

func (s *ScopedQueries) ListSchedulableTasks(ctx context.Context) ([]Task, error) {
	uid, err := s.userID(ctx)
	if err != nil {
		return nil, err
	}
	return s.q.ListSchedulableTasks(ctx, uid)
}

func (s *ScopedQueries) UpdateTask(ctx context.Context, arg UpdateTaskParams) (Task, error) {
	uid, err := s.userID(ctx)
	if err != nil {
		return Task{}, err
	}
	arg.UserID = uid
	return s.q.UpdateTask(ctx, arg)
}

func (s *ScopedQueries) CompleteTask(ctx context.Context, id pgtype.UUID) (Task, error) {
	uid, err := s.userID(ctx)
	if err != nil {
		return Task{}, err
	}
	return s.q.CompleteTask(ctx, CompleteTaskParams{ID: id, UserID: uid})
}

func (s *ScopedQueries) UncompleteTask(ctx context.Context, id pgtype.UUID) (Task, error) {
	uid, err := s.userID(ctx)
	if err != nil {
		return Task{}, err
	}
	return s.q.UncompleteTask(ctx, UncompleteTaskParams{ID: id, UserID: uid})
}

func (s *ScopedQueries) DeleteTask(ctx context.Context, id pgtype.UUID) error {
	uid, err := s.userID(ctx)
	if err != nil {
		return err
	}
	return s.q.DeleteTask(ctx, DeleteTaskParams{ID: id, UserID: uid})
}

func (s *ScopedQueries) CountIncompleteSubtasks(ctx context.Context, parentID pgtype.UUID) (int64, error) {
	uid, err := s.userID(ctx)
	if err != nil {
		return 0, err
	}
	return s.q.CountIncompleteSubtasks(ctx, CountIncompleteSubtasksParams{ParentID: parentID, UserID: uid})
}

func (s *ScopedQueries) IncrementTimesDeferred(ctx context.Context, id pgtype.UUID) (Task, error) {
	uid, err := s.userID(ctx)
	if err != nil {
		return Task{}, err
	}
	return s.q.IncrementTimesDeferred(ctx, IncrementTimesDeferredParams{ID: id, UserID: uid})
}

// --- ContextStore ---

func (s *ScopedQueries) ListContextEntries(ctx context.Context, category *string) ([]ContextEntry, error) {
	uid, err := s.userID(ctx)
	if err != nil {
		return nil, err
	}
	return s.q.ListContextEntries(ctx, ListContextEntriesParams{UserID: uid, Category: category})
}

func (s *ScopedQueries) GetContextEntry(ctx context.Context, id pgtype.UUID) (ContextEntry, error) {
	uid, err := s.userID(ctx)
	if err != nil {
		return ContextEntry{}, err
	}
	return s.q.GetContextEntry(ctx, GetContextEntryParams{ID: id, UserID: uid})
}

func (s *ScopedQueries) UpsertContextEntry(ctx context.Context, arg UpsertContextEntryParams) (ContextEntry, error) {
	uid, err := s.userID(ctx)
	if err != nil {
		return ContextEntry{}, err
	}
	arg.UserID = uid
	return s.q.UpsertContextEntry(ctx, arg)
}

func (s *ScopedQueries) ToggleContextEntry(ctx context.Context, arg ToggleContextEntryParams) (ContextEntry, error) {
	uid, err := s.userID(ctx)
	if err != nil {
		return ContextEntry{}, err
	}
	arg.UserID = uid
	return s.q.ToggleContextEntry(ctx, arg)
}

func (s *ScopedQueries) DeleteContextEntry(ctx context.Context, id pgtype.UUID) error {
	uid, err := s.userID(ctx)
	if err != nil {
		return err
	}
	return s.q.DeleteContextEntry(ctx, DeleteContextEntryParams{ID: id, UserID: uid})
}

// --- DayPlanStore ---

func (s *ScopedQueries) GetDayPlanByDate(ctx context.Context, planDate pgtype.Date) (DayPlan, error) {
	uid, err := s.userID(ctx)
	if err != nil {
		return DayPlan{}, err
	}
	return s.q.GetDayPlanByDate(ctx, GetDayPlanByDateParams{PlanDate: planDate, UserID: uid})
}

func (s *ScopedQueries) GetDayPlanByID(ctx context.Context, id pgtype.UUID) (DayPlan, error) {
	uid, err := s.userID(ctx)
	if err != nil {
		return DayPlan{}, err
	}
	return s.q.GetDayPlanByID(ctx, GetDayPlanByIDParams{ID: id, UserID: uid})
}

func (s *ScopedQueries) CreateDayPlan(ctx context.Context, arg CreateDayPlanParams) (DayPlan, error) {
	uid, err := s.userID(ctx)
	if err != nil {
		return DayPlan{}, err
	}
	arg.UserID = uid
	return s.q.CreateDayPlan(ctx, arg)
}

func (s *ScopedQueries) UpdateDayPlanBlocks(ctx context.Context, arg UpdateDayPlanBlocksParams) (DayPlan, error) {
	uid, err := s.userID(ctx)
	if err != nil {
		return DayPlan{}, err
	}
	arg.UserID = uid
	return s.q.UpdateDayPlanBlocks(ctx, arg)
}

func (s *ScopedQueries) UpdateDayPlanStatus(ctx context.Context, arg UpdateDayPlanStatusParams) (DayPlan, error) {
	uid, err := s.userID(ctx)
	if err != nil {
		return DayPlan{}, err
	}
	arg.UserID = uid
	return s.q.UpdateDayPlanStatus(ctx, arg)
}

func (s *ScopedQueries) RecentPlans(ctx context.Context, limit int32) ([]DayPlan, error) {
	uid, err := s.userID(ctx)
	if err != nil {
		return nil, err
	}
	return s.q.RecentPlans(ctx, RecentPlansParams{Limit: limit, UserID: uid})
}

func (s *ScopedQueries) GetPlanMessages(ctx context.Context, planID pgtype.UUID) ([]PlanMessage, error) {
	// Plan messages are scoped transitively via plan_id — no user_id needed
	return s.q.GetPlanMessages(ctx, planID)
}

func (s *ScopedQueries) CreatePlanMessage(ctx context.Context, arg CreatePlanMessageParams) (PlanMessage, error) {
	// Plan messages are scoped transitively via plan_id — no user_id needed
	return s.q.CreatePlanMessage(ctx, arg)
}

// --- TaskConversationStore ---

func (s *ScopedQueries) CreateTaskConversation(ctx context.Context) (TaskConversation, error) {
	uid, err := s.userID(ctx)
	if err != nil {
		return TaskConversation{}, err
	}
	return s.q.CreateTaskConversation(ctx, uid)
}

func (s *ScopedQueries) GetTaskConversation(ctx context.Context, id pgtype.UUID) (TaskConversation, error) {
	uid, err := s.userID(ctx)
	if err != nil {
		return TaskConversation{}, err
	}
	return s.q.GetTaskConversation(ctx, GetTaskConversationParams{ID: id, UserID: uid})
}

func (s *ScopedQueries) UpdateTaskConversationStatus(ctx context.Context, arg UpdateTaskConversationStatusParams) (TaskConversation, error) {
	uid, err := s.userID(ctx)
	if err != nil {
		return TaskConversation{}, err
	}
	arg.UserID = uid
	return s.q.UpdateTaskConversationStatus(ctx, arg)
}

func (s *ScopedQueries) LinkTaskConversationParent(ctx context.Context, arg LinkTaskConversationParentParams) (TaskConversation, error) {
	uid, err := s.userID(ctx)
	if err != nil {
		return TaskConversation{}, err
	}
	arg.UserID = uid
	return s.q.LinkTaskConversationParent(ctx, arg)
}

func (s *ScopedQueries) CreateTaskMessage(ctx context.Context, arg CreateTaskMessageParams) (TaskMessage, error) {
	// Task messages are scoped transitively via conversation_id — no user_id needed
	return s.q.CreateTaskMessage(ctx, arg)
}

func (s *ScopedQueries) GetTaskMessages(ctx context.Context, conversationID pgtype.UUID) ([]TaskMessage, error) {
	// Task messages are scoped transitively via conversation_id — no user_id needed
	return s.q.GetTaskMessages(ctx, conversationID)
}

// --- calendar.GoogleAuthStore ---

func (s *ScopedQueries) GetGoogleAuth(ctx context.Context) (GoogleAuth, error) {
	uid, err := s.userID(ctx)
	if err != nil {
		return GoogleAuth{}, err
	}
	return s.q.GetGoogleAuth(ctx, uid)
}

func (s *ScopedQueries) UpsertGoogleAuth(ctx context.Context, arg UpsertGoogleAuthParams) (GoogleAuth, error) {
	uid, err := s.userID(ctx)
	if err != nil {
		return GoogleAuth{}, err
	}
	arg.UserID = uid
	return s.q.UpsertGoogleAuth(ctx, arg)
}

func (s *ScopedQueries) DeleteGoogleAuth(ctx context.Context) error {
	uid, err := s.userID(ctx)
	if err != nil {
		return err
	}
	return s.q.DeleteGoogleAuth(ctx, uid)
}

// --- Carryover queries (used by resolvers) ---

func (s *ScopedQueries) ComputeActualMinutesForTask(ctx context.Context, taskID string) (int32, error) {
	uid, err := s.userID(ctx)
	if err != nil {
		return 0, err
	}
	return s.q.ComputeActualMinutesForTask(ctx, ComputeActualMinutesForTaskParams{UserID: uid, TaskID: taskID})
}

func (s *ScopedQueries) GetSkippedBlocksFromLastPlan(ctx context.Context, today time.Time) ([]GetSkippedBlocksFromLastPlanRow, error) {
	uid, err := s.userID(ctx)
	if err != nil {
		return nil, err
	}
	return s.q.GetSkippedBlocksFromLastPlan(ctx, GetSkippedBlocksFromLastPlanParams{
		UserID: uid,
		Today:  pgtype.Date{Time: today, Valid: true},
	})
}

// --- ListActiveContextEntries (used by planner) ---

func (s *ScopedQueries) ListActiveContextEntries(ctx context.Context) ([]ContextEntry, error) {
	uid, err := s.userID(ctx)
	if err != nil {
		return nil, err
	}
	return s.q.ListActiveContextEntries(ctx, uid)
}
