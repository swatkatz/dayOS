package graph_test

import (
	"context"
	"testing"

	"dayos/graph"
	"dayos/graph/model"

	"github.com/google/uuid"
)

func newContextResolver(store *mockContextStore) *graph.Resolver {
	return &graph.Resolver{ContextStore: store}
}

// Test Anchor 1: Given seed context exists (9 entries), when contextEntries()
// is called, then all 9 entries are returned ordered by category then key.
func TestContextEntries_ListAll(t *testing.T) {
	store := newMockContextStore()
	seedContextEntries(store)
	r := newContextResolver(store)

	entries, err := r.Query().ContextEntries(context.Background(), nil)
	if err != nil {
		t.Fatalf("ContextEntries: %v", err)
	}
	if len(entries) != 9 {
		t.Fatalf("expected 9 entries, got %d", len(entries))
	}

	// Verify ordering: constraints < equipment < life < preferences
	prevCat := ""
	for _, e := range entries {
		cat := string(e.Category)
		if cat < prevCat {
			t.Errorf("entries not ordered by category: %s came after %s", cat, prevCat)
		}
		prevCat = cat
	}
}

// Test Anchor 1b: Filter by category
func TestContextEntries_FilterByCategory(t *testing.T) {
	store := newMockContextStore()
	seedContextEntries(store)
	r := newContextResolver(store)

	cat := model.ContextCategoryConstraints
	entries, err := r.Query().ContextEntries(context.Background(), &cat)
	if err != nil {
		t.Fatalf("ContextEntries: %v", err)
	}
	if len(entries) != 5 {
		t.Fatalf("expected 5 constraint entries, got %d", len(entries))
	}
	for _, e := range entries {
		if e.Category != model.ContextCategoryConstraints {
			t.Errorf("expected CONSTRAINTS, got %s", e.Category)
		}
	}
}

// Test Anchor 2: Given a context entry exists with category=constraints, key=work_window,
// when upsertContext is called with the same category+key but new value, then the existing
// entry is updated (not duplicated) and the new value is returned.
func TestUpsertContext_UpdateExisting(t *testing.T) {
	store := newMockContextStore()
	seedContextEntries(store)
	r := newContextResolver(store)
	ctx := context.Background()

	result, err := r.Mutation().UpsertContext(ctx, model.UpsertContextInput{
		Category: model.ContextCategoryConstraints,
		Key:      "work_window",
		Value:    "New schedule",
	})
	if err != nil {
		t.Fatalf("UpsertContext: %v", err)
	}
	if result.Value != "New schedule" {
		t.Errorf("value = %q, want %q", result.Value, "New schedule")
	}

	// Verify no duplicate was created
	entries, _ := r.Query().ContextEntries(ctx, nil)
	if len(entries) != 9 {
		t.Errorf("expected 9 entries after upsert, got %d", len(entries))
	}
}

// Test Anchor 3: Given no entry with key=commute exists, when upsertContext is called,
// then a new entry is created with isActive=true.
func TestUpsertContext_CreateNew(t *testing.T) {
	store := newMockContextStore()
	r := newContextResolver(store)

	result, err := r.Mutation().UpsertContext(context.Background(), model.UpsertContextInput{
		Category: model.ContextCategoryConstraints,
		Key:      "commute",
		Value:    "15 min walk",
	})
	if err != nil {
		t.Fatalf("UpsertContext: %v", err)
	}
	if result.ID == uuid.Nil {
		t.Error("expected non-nil UUID")
	}
	if result.Key != "commute" {
		t.Errorf("key = %q, want %q", result.Key, "commute")
	}
	if result.Value != "15 min walk" {
		t.Errorf("value = %q, want %q", result.Value, "15 min walk")
	}
	if !result.IsActive {
		t.Error("expected isActive = true for new entry")
	}
}

// Test Anchor 4: Given an active context entry, when toggleContext(id, false) is called,
// then isActive is false.
func TestToggleContext(t *testing.T) {
	store := newMockContextStore()
	seedContextEntries(store)
	r := newContextResolver(store)
	ctx := context.Background()

	// Get the first entry
	entries, _ := r.Query().ContextEntries(ctx, nil)
	entry := entries[0]

	// Toggle inactive
	toggled, err := r.Mutation().ToggleContext(ctx, entry.ID, false)
	if err != nil {
		t.Fatalf("ToggleContext: %v", err)
	}
	if toggled.IsActive {
		t.Error("expected isActive = false after toggle")
	}

	// Toggle back active
	toggled, err = r.Mutation().ToggleContext(ctx, entry.ID, true)
	if err != nil {
		t.Fatalf("ToggleContext: %v", err)
	}
	if !toggled.IsActive {
		t.Error("expected isActive = true after re-toggle")
	}
}

// Test Anchor 5: Given a context entry exists, when deleteContext(id) is called,
// then the entry is permanently removed.
func TestDeleteContext(t *testing.T) {
	store := newMockContextStore()
	seedContextEntries(store)
	r := newContextResolver(store)
	ctx := context.Background()

	entries, _ := r.Query().ContextEntries(ctx, nil)
	target := entries[0]

	ok, err := r.Mutation().DeleteContext(ctx, target.ID)
	if err != nil {
		t.Fatalf("DeleteContext: %v", err)
	}
	if !ok {
		t.Error("expected true from DeleteContext")
	}

	// Verify entry is gone
	remaining, _ := r.Query().ContextEntries(ctx, nil)
	if len(remaining) != 8 {
		t.Errorf("expected 8 entries after delete, got %d", len(remaining))
	}
}

// Test Anchor 5b: Delete non-existent ID returns error.
func TestDeleteContext_NotFound(t *testing.T) {
	store := newMockContextStore()
	r := newContextResolver(store)

	_, err := r.Mutation().DeleteContext(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error deleting non-existent entry, got nil")
	}
}
