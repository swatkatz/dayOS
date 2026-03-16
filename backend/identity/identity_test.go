package identity

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
)

func testUUID(b byte) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte{b}, Valid: true}
}

func TestFromContext_WithUser(t *testing.T) {
	u := User{
		ID:      testUUID(1),
		ClerkID: "clerk_123",
		Email:   "test@example.com",
	}
	ctx := WithUser(context.Background(), u)

	got, ok := FromContext(ctx)
	if !ok {
		t.Fatal("expected ok=true, got false")
	}
	if got.ClerkID != u.ClerkID {
		t.Errorf("ClerkID = %q, want %q", got.ClerkID, u.ClerkID)
	}
	if got.Email != u.Email {
		t.Errorf("Email = %q, want %q", got.Email, u.Email)
	}
	if got.ID != u.ID {
		t.Errorf("ID mismatch")
	}
}

func TestFromContext_NoUser(t *testing.T) {
	_, ok := FromContext(context.Background())
	if ok {
		t.Fatal("expected ok=false for bare context")
	}
}

func TestMustFromContext_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic, got none")
		}
	}()
	MustFromContext(context.Background())
}

func TestMustFromContext_ReturnsUser(t *testing.T) {
	u := User{
		ID:              testUUID(2),
		ClerkID:         "clerk_456",
		Email:           "user@example.com",
		DisplayName:     "Test User",
		AnthropicAPIKey: strPtr("sk-test"),
	}
	ctx := WithUser(context.Background(), u)

	got := MustFromContext(ctx)
	if got.DisplayName != "Test User" {
		t.Errorf("DisplayName = %q, want %q", got.DisplayName, "Test User")
	}
	if got.AnthropicAPIKey == nil || *got.AnthropicAPIKey != "sk-test" {
		t.Error("AnthropicAPIKey not preserved")
	}
}

func strPtr(s string) *string { return &s }
