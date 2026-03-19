package calendar

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"dayos/db"

	"github.com/jackc/pgx/v5/pgtype"
)

// GoogleAuthStore abstracts the DB queries needed by the calendar service.
type GoogleAuthStore interface {
	GetGoogleAuth(ctx context.Context) (db.GoogleAuth, error)
	UpsertGoogleAuth(ctx context.Context, arg db.UpsertGoogleAuthParams) (db.GoogleAuth, error)
	DeleteGoogleAuth(ctx context.Context) error
}

// CalendarAPI abstracts the Google Calendar API.
type CalendarAPI interface {
	FetchEvents(ctx context.Context, accessToken string, calendarID string, date time.Time) ([]RawEvent, error)
}

// TokenRefresher abstracts OAuth token refresh.
type TokenRefresher interface {
	RefreshAccessToken(ctx context.Context, refreshToken string) (newToken string, expiry time.Time, err error)
}

// Cache abstracts memcached or any key-value cache.
type Cache interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte, ttl time.Duration) error
}

// RawEvent is the raw event data from the Google Calendar API.
type RawEvent struct {
	Title         string
	Start         time.Time
	End           time.Time
	AllDay        bool
	AttendeeCount int    // number of non-self attendees
	EventType     string // e.g. "focusTime", "default"
}

// Event is the minimal event data exposed via GraphQL and to the planner.
type Event struct {
	Title         string `json:"title"`
	StartTime     string `json:"start_time"`     // "HH:MM"
	Duration      int    `json:"duration"`        // minutes
	AllDay        bool   `json:"all_day"`
	AttendeeCount int    `json:"attendee_count"`  // non-self attendees
	EventType     string `json:"event_type"`      // e.g. "focusTime", "default"
}

// EventsResult is the return value from GetEvents.
type EventsResult struct {
	Events    []Event
	Version   string // SHA-256 hash of serialized events
	Connected bool
}

// StatusResult is the return value from GetStatus.
type StatusResult struct {
	Connected    bool
	CalendarName *string
}

// OAuthExchanger abstracts OAuth code exchange.
type OAuthExchanger interface {
	ExchangeCode(ctx context.Context, code string) (accessToken, refreshToken string, expiry time.Time, err error)
}

// Service handles Google Calendar integration.
type Service struct {
	Store     GoogleAuthStore
	Cache     Cache
	API       CalendarAPI
	Refresher TokenRefresher
	Exchanger OAuthExchanger
}

const cacheTTL = 20 * time.Minute

func cacheKey(date time.Time) string {
	return fmt.Sprintf("cal:events:%s", date.Format("2006-01-02"))
}

type cachedEvents struct {
	Events  []Event `json:"events"`
	Version string  `json:"version"`
}

// GetEvents returns calendar events for the given date.
func (s *Service) GetEvents(ctx context.Context, date time.Time) (*EventsResult, error) {
	// Check if connected
	auth, err := s.Store.GetGoogleAuth(ctx)
	if err != nil {
		// Not connected
		return &EventsResult{
			Events:    []Event{},
			Version:   "",
			Connected: false,
		}, nil
	}

	// Check cache
	if s.Cache != nil {
		if data, err := s.Cache.Get(cacheKey(date)); err == nil {
			var cached cachedEvents
			if json.Unmarshal(data, &cached) == nil {
				return &EventsResult{
					Events:    cached.Events,
					Version:   cached.Version,
					Connected: true,
				}, nil
			}
		}
	}

	// Check if token is expired and refresh if needed
	if auth.TokenExpiry.Valid && auth.TokenExpiry.Time.Before(time.Now()) {
		if s.Refresher == nil {
			// No refresher — treat as disconnected
			_ = s.Store.DeleteGoogleAuth(ctx)
			return &EventsResult{Events: []Event{}, Connected: false}, nil
		}
		newToken, newExpiry, err := s.Refresher.RefreshAccessToken(ctx, auth.RefreshToken)
		if err != nil {
			// Refresh failed — token revoked, disconnect
			_ = s.Store.DeleteGoogleAuth(ctx)
			return &EventsResult{Events: []Event{}, Connected: false}, nil
		}
		// Update stored tokens
		updated, err := s.Store.UpsertGoogleAuth(ctx, db.UpsertGoogleAuthParams{
			AccessToken:  newToken,
			RefreshToken: auth.RefreshToken,
			TokenExpiry:  pgtype.Timestamptz{Time: newExpiry, Valid: true},
			CalendarID:   auth.CalendarID,
		})
		if err != nil {
			return nil, fmt.Errorf("updating tokens: %w", err)
		}
		auth = updated
	}

	// Fetch from Google Calendar API
	if s.API == nil {
		return &EventsResult{Events: []Event{}, Connected: true}, nil
	}

	rawEvents, err := s.API.FetchEvents(ctx, auth.AccessToken, auth.CalendarID, date)
	if err != nil {
		return nil, fmt.Errorf("fetching calendar events: %w", err)
	}

	// Convert to minimal events
	events := convertEvents(rawEvents)

	// Sort by start time
	sort.Slice(events, func(i, j int) bool {
		if events[i].AllDay != events[j].AllDay {
			return !events[i].AllDay // timed events first
		}
		return events[i].StartTime < events[j].StartTime
	})

	version := computeVersion(events)

	// Write to cache
	if s.Cache != nil {
		cached := cachedEvents{Events: events, Version: version}
		if data, err := json.Marshal(cached); err == nil {
			_ = s.Cache.Set(cacheKey(date), data, cacheTTL)
		}
	}

	return &EventsResult{
		Events:    events,
		Version:   version,
		Connected: true,
	}, nil
}

// ExchangeCodeAndStore exchanges an OAuth authorization code for tokens and stores them.
func (s *Service) ExchangeCodeAndStore(ctx context.Context, code string) error {
	if s.Exchanger == nil {
		return fmt.Errorf("OAuth exchanger not configured")
	}
	accessToken, refreshToken, expiry, err := s.Exchanger.ExchangeCode(ctx, code)
	if err != nil {
		return fmt.Errorf("exchanging code: %w", err)
	}
	return s.storeTokens(ctx, accessToken, refreshToken, expiry, "primary")
}

// storeTokens saves OAuth tokens to the database.
func (s *Service) storeTokens(ctx context.Context, accessToken, refreshToken string, expiry time.Time, calendarID string) error {
	_, err := s.Store.UpsertGoogleAuth(ctx, db.UpsertGoogleAuthParams{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenExpiry:  pgtype.Timestamptz{Time: expiry, Valid: true},
		CalendarID:   calendarID,
	})
	return err
}

// Disconnect deletes the stored OAuth tokens.
func (s *Service) Disconnect(ctx context.Context) error {
	return s.Store.DeleteGoogleAuth(ctx)
}

// IsConnected checks if Google Calendar is connected.
func (s *Service) IsConnected(ctx context.Context) (bool, error) {
	_, err := s.Store.GetGoogleAuth(ctx)
	if err != nil {
		return false, nil
	}
	return true, nil
}

// GetStatus returns connection status and calendar name.
func (s *Service) GetStatus(ctx context.Context) (*StatusResult, error) {
	auth, err := s.Store.GetGoogleAuth(ctx)
	if err != nil {
		return &StatusResult{Connected: false}, nil
	}
	name := auth.CalendarID
	return &StatusResult{Connected: true, CalendarName: &name}, nil
}

func convertEvents(raw []RawEvent) []Event {
	events := make([]Event, 0, len(raw))
	for _, r := range raw {
		e := Event{
			Title:         r.Title,
			AllDay:        r.AllDay,
			AttendeeCount: r.AttendeeCount,
			EventType:     r.EventType,
		}
		if !r.AllDay {
			e.StartTime = r.Start.Format("15:04")
			e.Duration = int(r.End.Sub(r.Start).Minutes())
		}
		events = append(events, e)
	}
	return events
}

func computeVersion(events []Event) string {
	// Sort by start time for determinism
	sorted := make([]Event, len(events))
	copy(sorted, events)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].AllDay != sorted[j].AllDay {
			return !sorted[i].AllDay
		}
		return sorted[i].StartTime < sorted[j].StartTime
	})
	b, _ := json.Marshal(sorted)
	h := sha256.Sum256(b)
	return fmt.Sprintf("%x", h)
}
