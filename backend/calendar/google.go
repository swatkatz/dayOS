package calendar

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// GoogleCalendarAPI implements CalendarAPI using the Google Calendar REST API.
type GoogleCalendarAPI struct {
	HTTPClient *http.Client
}

func NewGoogleCalendarAPI() *GoogleCalendarAPI {
	return &GoogleCalendarAPI{HTTPClient: http.DefaultClient}
}

// calendarEventAttendee represents a single attendee entry from Google Calendar.
type calendarEventAttendee struct {
	Self bool `json:"self"`
}

// calendarEvent represents the relevant fields from a Google Calendar event.
type calendarEvent struct {
	Summary   string                  `json:"summary"`
	Start     calendarEventTime       `json:"start"`
	End       calendarEventTime       `json:"end"`
	Attendees []calendarEventAttendee `json:"attendees"`
	EventType string                  `json:"eventType"`
}

type calendarEventTime struct {
	DateTime string `json:"dateTime"` // RFC3339 for timed events
	Date     string `json:"date"`     // YYYY-MM-DD for all-day events
}

type calendarListResponse struct {
	Items []calendarEvent `json:"items"`
}

func (g *GoogleCalendarAPI) FetchEvents(ctx context.Context, accessToken string, calendarID string, date time.Time) ([]RawEvent, error) {
	// Set time bounds for the day
	loc := date.Location()
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, loc)
	dayEnd := dayStart.Add(24 * time.Hour)

	params := url.Values{}
	params.Set("timeMin", dayStart.Format(time.RFC3339))
	params.Set("timeMax", dayEnd.Format(time.RFC3339))
	params.Set("singleEvents", "true")
	params.Set("orderBy", "startTime")
	params.Set("fields", "items(summary,start,end,attendees/self,eventType)")

	apiURL := fmt.Sprintf("https://www.googleapis.com/calendar/v3/calendars/%s/events?%s",
		url.PathEscape(calendarID), params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := g.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling Google Calendar API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("google calendar API returned %d: %s", resp.StatusCode, string(body))
	}

	var listResp calendarListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	var events []RawEvent
	for _, item := range listResp.Items {
		e := RawEvent{Title: item.Summary, EventType: item.EventType}

		// Count non-self attendees (excludes the calendar owner's own RSVP)
		for _, a := range item.Attendees {
			if !a.Self {
				e.AttendeeCount++
			}
		}

		if item.Start.Date != "" {
			// All-day event
			e.AllDay = true
		} else if item.Start.DateTime != "" {
			start, err := time.Parse(time.RFC3339, item.Start.DateTime)
			if err != nil {
				continue
			}
			end, err := time.Parse(time.RFC3339, item.End.DateTime)
			if err != nil {
				continue
			}
			e.Start = start
			e.End = end
		}
		events = append(events, e)
	}
	return events, nil
}

// GoogleTokenRefresher implements TokenRefresher using Google's OAuth2 token endpoint.
type GoogleTokenRefresher struct {
	ClientID     string
	ClientSecret string
	HTTPClient   *http.Client
}

func NewGoogleTokenRefresher(clientID, clientSecret string) *GoogleTokenRefresher {
	return &GoogleTokenRefresher{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		HTTPClient:   http.DefaultClient,
	}
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

func (g *GoogleTokenRefresher) RefreshAccessToken(ctx context.Context, refreshToken string) (string, time.Time, error) {
	data := url.Values{}
	data.Set("client_id", g.ClientID)
	data.Set("client_secret", g.ClientSecret)
	data.Set("refresh_token", refreshToken)
	data.Set("grant_type", "refresh_token")

	req, err := http.NewRequestWithContext(ctx, "POST", "https://oauth2.googleapis.com/token", strings.NewReader(data.Encode()))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := g.HTTPClient.Do(req)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("calling token endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", time.Time{}, fmt.Errorf("token refresh failed (%d): %s", resp.StatusCode, string(body))
	}

	var tokenResp tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", time.Time{}, fmt.Errorf("decoding token response: %w", err)
	}

	expiry := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	return tokenResp.AccessToken, expiry, nil
}

// GoogleOAuthExchanger handles exchanging authorization codes for tokens.
type GoogleOAuthExchanger struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	HTTPClient   *http.Client
}

func NewGoogleOAuthExchanger(clientID, clientSecret, redirectURI string) *GoogleOAuthExchanger {
	return &GoogleOAuthExchanger{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURI:  redirectURI,
		HTTPClient:   http.DefaultClient,
	}
}

type exchangeResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

func (g *GoogleOAuthExchanger) ExchangeCode(ctx context.Context, code string) (accessToken, refreshToken string, expiry time.Time, err error) {
	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", g.ClientID)
	data.Set("client_secret", g.ClientSecret)
	data.Set("redirect_uri", g.RedirectURI)
	data.Set("grant_type", "authorization_code")

	req, err := http.NewRequestWithContext(ctx, "POST", "https://oauth2.googleapis.com/token", strings.NewReader(data.Encode()))
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := g.HTTPClient.Do(req)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("calling token endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", "", time.Time{}, fmt.Errorf("code exchange failed (%d): %s", resp.StatusCode, string(body))
	}

	var exResp exchangeResponse
	if err := json.NewDecoder(resp.Body).Decode(&exResp); err != nil {
		return "", "", time.Time{}, fmt.Errorf("decoding response: %w", err)
	}

	return exResp.AccessToken, exResp.RefreshToken, time.Now().Add(time.Duration(exResp.ExpiresIn) * time.Second), nil
}
