package calendar

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"dayos/db"

	"github.com/jackc/pgx/v5/pgtype"
)

// --- Mock GoogleAuthStore ---

type mockGoogleAuthStore struct {
	auth *db.GoogleAuth
}

func (m *mockGoogleAuthStore) GetGoogleAuth(_ context.Context) (db.GoogleAuth, error) {
	if m.auth == nil {
		return db.GoogleAuth{}, fmt.Errorf("no rows in result set")
	}
	return *m.auth, nil
}

func (m *mockGoogleAuthStore) UpsertGoogleAuth(_ context.Context, arg db.UpsertGoogleAuthParams) (db.GoogleAuth, error) {
	m.auth = &db.GoogleAuth{
		AccessToken:  arg.AccessToken,
		RefreshToken: arg.RefreshToken,
		TokenExpiry:  arg.TokenExpiry,
		CalendarID:   arg.CalendarID,
	}
	return *m.auth, nil
}

func (m *mockGoogleAuthStore) DeleteGoogleAuth(_ context.Context) error {
	m.auth = nil
	return nil
}

// --- Mock CalendarAPI ---

type mockCalendarAPI struct {
	events []RawEvent
	err    error
}

func (m *mockCalendarAPI) FetchEvents(_ context.Context, accessToken string, calendarID string, date time.Time) ([]RawEvent, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.events, nil
}

// --- Mock TokenRefresher ---

type mockTokenRefresher struct {
	newToken  string
	newExpiry time.Time
	err       error
}

func (m *mockTokenRefresher) RefreshAccessToken(_ context.Context, refreshToken string) (string, time.Time, error) {
	if m.err != nil {
		return "", time.Time{}, m.err
	}
	return m.newToken, m.newExpiry, nil
}

// --- Mock Cache ---

type mockCache struct {
	data map[string][]byte
}

func newMockCache() *mockCache {
	return &mockCache{data: make(map[string][]byte)}
}

func (m *mockCache) Get(key string) ([]byte, error) {
	v, ok := m.data[key]
	if !ok {
		return nil, fmt.Errorf("cache miss")
	}
	return v, nil
}

func (m *mockCache) Set(key string, value []byte, ttl time.Duration) error {
	m.data[key] = value
	return nil
}

// --- Tests ---

func TestGetEvents_NotConnected(t *testing.T) {
	svc := &Service{
		Store: &mockGoogleAuthStore{auth: nil},
		Cache: newMockCache(),
	}

	result, err := svc.GetEvents(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Connected {
		t.Error("expected connected=false when not connected")
	}
	if len(result.Events) != 0 {
		t.Error("expected empty events when not connected")
	}
	if result.Version != "" {
		t.Error("expected empty version when not connected")
	}
}

func TestGetEvents_CacheHit(t *testing.T) {
	cache := newMockCache()

	// Pre-populate cache
	cached := &cachedEvents{
		Events: []Event{
			{Title: "Standup", StartTime: "09:30", Duration: 30, AllDay: false},
		},
		Version: "abc123",
	}
	data, _ := json.Marshal(cached)
	date := time.Date(2026, 3, 13, 0, 0, 0, 0, time.UTC)
	_ = cache.Set(cacheKey(date), data, 0)

	store := &mockGoogleAuthStore{
		auth: &db.GoogleAuth{
			AccessToken:  "valid-token",
			RefreshToken: "refresh-token",
			TokenExpiry:  pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true},
		},
	}

	api := &mockCalendarAPI{} // Should not be called

	svc := &Service{
		Store: store,
		Cache: cache,
		API:   api,
	}

	result, err := svc.GetEvents(context.Background(), date)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Connected {
		t.Error("expected connected=true")
	}
	if len(result.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(result.Events))
	}
	if result.Events[0].Title != "Standup" {
		t.Errorf("expected title 'Standup', got %q", result.Events[0].Title)
	}
	if result.Version != "abc123" {
		t.Errorf("expected version 'abc123', got %q", result.Version)
	}
}

func TestGetEvents_CacheMiss_FetchesFromAPI(t *testing.T) {
	store := &mockGoogleAuthStore{
		auth: &db.GoogleAuth{
			AccessToken:  "valid-token",
			RefreshToken: "refresh-token",
			TokenExpiry:  pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true},
		},
	}

	api := &mockCalendarAPI{
		events: []RawEvent{
			{Title: "Team standup", Start: time.Date(2026, 3, 13, 9, 30, 0, 0, time.UTC), End: time.Date(2026, 3, 13, 10, 0, 0, 0, time.UTC), AllDay: false},
			{Title: "Design review", Start: time.Date(2026, 3, 13, 13, 0, 0, 0, time.UTC), End: time.Date(2026, 3, 13, 14, 0, 0, 0, time.UTC), AllDay: false},
			{Title: "Birthday", AllDay: true},
		},
	}

	cache := newMockCache()

	svc := &Service{
		Store: store,
		Cache: cache,
		API:   api,
	}

	date := time.Date(2026, 3, 13, 0, 0, 0, 0, time.UTC)
	result, err := svc.GetEvents(context.Background(), date)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Connected {
		t.Error("expected connected=true")
	}
	if len(result.Events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(result.Events))
	}

	// Check data minimization: only title, startTime, duration, allDay
	if result.Events[0].Title != "Team standup" {
		t.Errorf("expected 'Team standup', got %q", result.Events[0].Title)
	}
	if result.Events[0].StartTime != "09:30" {
		t.Errorf("expected startTime '09:30', got %q", result.Events[0].StartTime)
	}
	if result.Events[0].Duration != 30 {
		t.Errorf("expected duration 30, got %d", result.Events[0].Duration)
	}

	// All-day event
	if !result.Events[2].AllDay {
		t.Error("expected Birthday to be all-day")
	}

	// Check cache was populated
	if _, err := cache.Get(cacheKey(date)); err != nil {
		t.Error("expected cache to be populated after API fetch")
	}

	// Check version is non-empty
	if result.Version == "" {
		t.Error("expected non-empty version")
	}
}

func TestGetEvents_ExpiredToken_RefreshesSuccessfully(t *testing.T) {
	store := &mockGoogleAuthStore{
		auth: &db.GoogleAuth{
			AccessToken:  "expired-token",
			RefreshToken: "valid-refresh",
			TokenExpiry:  pgtype.Timestamptz{Time: time.Now().Add(-time.Hour), Valid: true},
			CalendarID:   "primary",
		},
	}

	api := &mockCalendarAPI{
		events: []RawEvent{
			{Title: "Meeting", Start: time.Date(2026, 3, 13, 10, 0, 0, 0, time.UTC), End: time.Date(2026, 3, 13, 11, 0, 0, 0, time.UTC)},
		},
	}

	refresher := &mockTokenRefresher{
		newToken:  "new-access-token",
		newExpiry: time.Now().Add(time.Hour),
	}

	svc := &Service{
		Store:     store,
		Cache:     newMockCache(),
		API:       api,
		Refresher: refresher,
	}

	result, err := svc.GetEvents(context.Background(), time.Date(2026, 3, 13, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Connected {
		t.Error("expected connected=true after token refresh")
	}
	if len(result.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(result.Events))
	}

	// Verify token was updated in store
	if store.auth.AccessToken != "new-access-token" {
		t.Errorf("expected store to have new access token, got %q", store.auth.AccessToken)
	}
}

func TestGetEvents_ExpiredToken_RefreshFails(t *testing.T) {
	store := &mockGoogleAuthStore{
		auth: &db.GoogleAuth{
			AccessToken:  "expired-token",
			RefreshToken: "invalid-refresh",
			TokenExpiry:  pgtype.Timestamptz{Time: time.Now().Add(-time.Hour), Valid: true},
			CalendarID:   "primary",
		},
	}

	refresher := &mockTokenRefresher{
		err: fmt.Errorf("token revoked"),
	}

	svc := &Service{
		Store:     store,
		Cache:     newMockCache(),
		Refresher: refresher,
	}

	result, err := svc.GetEvents(context.Background(), time.Date(2026, 3, 13, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Connected {
		t.Error("expected connected=false when refresh fails")
	}

	// Verify google_auth row was deleted
	if store.auth != nil {
		t.Error("expected google_auth to be deleted after refresh failure")
	}
}

func TestVersionDeterministic(t *testing.T) {
	events := []Event{
		{Title: "Standup", StartTime: "09:30", Duration: 30, AllDay: false},
		{Title: "Review", StartTime: "13:00", Duration: 60, AllDay: false},
	}

	v1 := computeVersion(events)
	v2 := computeVersion(events)

	if v1 != v2 {
		t.Errorf("version should be deterministic: %q != %q", v1, v2)
	}
	if v1 == "" {
		t.Error("version should not be empty")
	}
}

func TestVersionChangesWithDifferentEvents(t *testing.T) {
	events1 := []Event{
		{Title: "Standup", StartTime: "09:30", Duration: 30},
	}
	events2 := []Event{
		{Title: "Standup", StartTime: "09:30", Duration: 30},
		{Title: "New meeting", StartTime: "14:00", Duration: 60},
	}

	v1 := computeVersion(events1)
	v2 := computeVersion(events2)

	if v1 == v2 {
		t.Error("version should differ when events differ")
	}
}

func TestExchangeCodeAndStore(t *testing.T) {
	store := &mockGoogleAuthStore{}
	exchanger := &mockOAuthExchanger{
		accessToken:  "access-token",
		refreshToken: "refresh-token",
		expiry:       time.Now().Add(time.Hour),
	}
	svc := &Service{Store: store, Exchanger: exchanger}

	err := svc.ExchangeCodeAndStore(context.Background(), "auth-code-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if store.auth == nil {
		t.Fatal("expected auth to be stored")
	}
	if store.auth.AccessToken != "access-token" {
		t.Errorf("expected access token %q, got %q", "access-token", store.auth.AccessToken)
	}
}

// --- Mock OAuthExchanger ---

type mockOAuthExchanger struct {
	accessToken  string
	refreshToken string
	expiry       time.Time
	err          error
}

func (m *mockOAuthExchanger) ExchangeCode(_ context.Context, code string) (string, string, time.Time, error) {
	if m.err != nil {
		return "", "", time.Time{}, m.err
	}
	return m.accessToken, m.refreshToken, m.expiry, nil
}

func TestDisconnect(t *testing.T) {
	store := &mockGoogleAuthStore{
		auth: &db.GoogleAuth{AccessToken: "tok"},
	}
	svc := &Service{Store: store}

	err := svc.Disconnect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.auth != nil {
		t.Error("expected auth to be deleted")
	}
}

func TestIsConnected(t *testing.T) {
	tests := []struct {
		name      string
		auth      *db.GoogleAuth
		connected bool
	}{
		{"connected", &db.GoogleAuth{AccessToken: "tok"}, true},
		{"not connected", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &Service{Store: &mockGoogleAuthStore{auth: tt.auth}}
			connected, err := svc.IsConnected(context.Background())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if connected != tt.connected {
				t.Errorf("expected connected=%v, got %v", tt.connected, connected)
			}
		})
	}
}

func TestDataMinimization(t *testing.T) {
	// Given 3 timed events and 1 all-day event, only title/startTime/duration/allDay extracted
	store := &mockGoogleAuthStore{
		auth: &db.GoogleAuth{
			AccessToken:  "tok",
			RefreshToken: "refresh",
			TokenExpiry:  pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true},
		},
	}

	api := &mockCalendarAPI{
		events: []RawEvent{
			{Title: "Event 1", Start: time.Date(2026, 3, 13, 9, 0, 0, 0, time.UTC), End: time.Date(2026, 3, 13, 10, 0, 0, 0, time.UTC)},
			{Title: "Event 2", Start: time.Date(2026, 3, 13, 11, 0, 0, 0, time.UTC), End: time.Date(2026, 3, 13, 12, 0, 0, 0, time.UTC)},
			{Title: "Event 3", Start: time.Date(2026, 3, 13, 14, 0, 0, 0, time.UTC), End: time.Date(2026, 3, 13, 15, 30, 0, 0, time.UTC)},
			{Title: "All Day", AllDay: true},
		},
	}

	svc := &Service{Store: store, Cache: newMockCache(), API: api}
	result, _ := svc.GetEvents(context.Background(), time.Date(2026, 3, 13, 0, 0, 0, 0, time.UTC))

	if len(result.Events) != 4 {
		t.Fatalf("expected 4 events, got %d", len(result.Events))
	}

	// Verify Event struct only has title, startTime, duration, allDay
	e := result.Events[2]
	if e.Title != "Event 3" || e.StartTime != "14:00" || e.Duration != 90 || e.AllDay {
		t.Errorf("unexpected event data: %+v", e)
	}

	// Verify all-day
	ad := result.Events[3]
	if !ad.AllDay || ad.Title != "All Day" {
		t.Errorf("unexpected all-day event: %+v", ad)
	}
}

func TestConvertEventsPreservesAttendeeCountAndEventType(t *testing.T) {
	raw := []RawEvent{
		{
			Title:         "Focus Time",
			Start:         time.Date(2026, 3, 13, 9, 0, 0, 0, time.UTC),
			End:           time.Date(2026, 3, 13, 11, 0, 0, 0, time.UTC),
			AttendeeCount: 0,
			EventType:     "focusTime",
		},
		{
			Title:         "Team Standup",
			Start:         time.Date(2026, 3, 13, 11, 0, 0, 0, time.UTC),
			End:           time.Date(2026, 3, 13, 11, 30, 0, 0, time.UTC),
			AttendeeCount: 5,
			EventType:     "default",
		},
		{
			Title:  "All Day",
			AllDay: true,
		},
	}

	events := convertEvents(raw)
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}

	// Focus time event
	if events[0].AttendeeCount != 0 {
		t.Errorf("expected attendee_count 0, got %d", events[0].AttendeeCount)
	}
	if events[0].EventType != "focusTime" {
		t.Errorf("expected event_type 'focusTime', got %q", events[0].EventType)
	}

	// Standup
	if events[1].AttendeeCount != 5 {
		t.Errorf("expected attendee_count 5, got %d", events[1].AttendeeCount)
	}
	if events[1].EventType != "default" {
		t.Errorf("expected event_type 'default', got %q", events[1].EventType)
	}

	// All-day events should preserve fields too
	if events[2].AttendeeCount != 0 {
		t.Errorf("expected attendee_count 0 for all-day, got %d", events[2].AttendeeCount)
	}
}

