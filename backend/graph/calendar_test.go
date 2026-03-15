package graph_test

import (
	"context"
	"testing"

	"dayos/calendar"
	"dayos/graph"
	"dayos/graph/model"
)

func newCalendarResolver(calSvc *mockCalendarService) *graph.Resolver {
	return &graph.Resolver{
		RoutineStore:          newMockRoutineStore(),
		TaskStore:             newMockTaskStore(),
		ContextStore:          newMockContextStore(),
		DayPlanStore:          newMockDayPlanStore(),
		TaskConversationStore: newMockTaskConversationStore(),
		Planner:               &mockPlanner{},
		Calendar:              calSvc,
	}
}

// Test anchor 3: not connected returns empty events with connected=false
func TestCalendarEvents_NotConnected(t *testing.T) {
	calSvc := &mockCalendarService{connected: false}
	r := newCalendarResolver(calSvc)

	date := model.Date{}
	result, err := r.Query().CalendarEvents(context.Background(), date)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Connected {
		t.Error("expected connected=false")
	}
	if len(result.Events) != 0 {
		t.Errorf("expected 0 events, got %d", len(result.Events))
	}
	if result.Version != "" {
		t.Errorf("expected empty version, got %q", result.Version)
	}
}

// Test anchor 1/2: connected returns events
func TestCalendarEvents_Connected(t *testing.T) {
	calSvc := &mockCalendarService{
		connected: true,
		events: []calendar.Event{
			{Title: "Team standup", StartTime: "09:30", Duration: 30, AllDay: false},
			{Title: "Birthday", AllDay: true},
		},
		version: "abc123",
	}
	r := newCalendarResolver(calSvc)

	date := model.Date{}
	result, err := r.Query().CalendarEvents(context.Background(), date)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Connected {
		t.Error("expected connected=true")
	}
	if len(result.Events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(result.Events))
	}
	if result.Events[0].Title != "Team standup" {
		t.Errorf("expected 'Team standup', got %q", result.Events[0].Title)
	}
	if result.Events[0].Duration != 30 {
		t.Errorf("expected duration 30, got %d", result.Events[0].Duration)
	}
	if !result.Events[1].AllDay {
		t.Error("expected second event to be all-day")
	}
	if result.Version != "abc123" {
		t.Errorf("expected version 'abc123', got %q", result.Version)
	}
}

// Test anchor 4: connectGoogleCalendar stores tokens
func TestConnectGoogleCalendar(t *testing.T) {
	calSvc := &mockCalendarService{connected: false}
	r := newCalendarResolver(calSvc)

	ok, err := r.Mutation().ConnectGoogleCalendar(context.Background(), "auth-code-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected true")
	}
	if !calSvc.stored {
		t.Error("expected ExchangeCodeAndStore to be called")
	}
	if !calSvc.connected {
		t.Error("expected connected=true after connect")
	}
}

// Test anchor 5: disconnectGoogleCalendar deletes auth
func TestDisconnectGoogleCalendar(t *testing.T) {
	calSvc := &mockCalendarService{
		connected: true,
		events:    []calendar.Event{{Title: "Meeting"}},
	}
	r := newCalendarResolver(calSvc)

	ok, err := r.Mutation().DisconnectGoogleCalendar(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected true")
	}
	if calSvc.connected {
		t.Error("expected connected=false after disconnect")
	}
}

// Test: googleCalendarStatus when connected
func TestGoogleCalendarStatus_Connected(t *testing.T) {
	calSvc := &mockCalendarService{connected: true}
	r := newCalendarResolver(calSvc)

	result, err := r.Query().GoogleCalendarStatus(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Connected {
		t.Error("expected connected=true")
	}
	if result.CalendarName == nil || *result.CalendarName != "primary" {
		t.Error("expected calendarName='primary'")
	}
}

// Test: googleCalendarStatus when not connected
func TestGoogleCalendarStatus_NotConnected(t *testing.T) {
	calSvc := &mockCalendarService{connected: false}
	r := newCalendarResolver(calSvc)

	result, err := r.Query().GoogleCalendarStatus(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Connected {
		t.Error("expected connected=false")
	}
	if result.CalendarName != nil {
		t.Error("expected nil calendarName")
	}
}

// Test: calendarEvents returns gracefully when Calendar is nil (not configured)
func TestCalendarEvents_NilService(t *testing.T) {
	r := &graph.Resolver{
		RoutineStore:          newMockRoutineStore(),
		TaskStore:             newMockTaskStore(),
		ContextStore:          newMockContextStore(),
		DayPlanStore:          newMockDayPlanStore(),
		TaskConversationStore: newMockTaskConversationStore(),
		Planner:               &mockPlanner{},
		Calendar:              nil,
	}

	date := model.Date{}
	result, err := r.Query().CalendarEvents(context.Background(), date)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Connected {
		t.Error("expected connected=false when Calendar is nil")
	}
}
