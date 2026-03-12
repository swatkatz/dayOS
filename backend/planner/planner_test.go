package planner

import (
	"context"
	"os"
	"strings"
	"testing"
)

// --- Mock AI Client ---

type mockAIClient struct {
	responses []string // returns responses in order
	calls     []mockCall
	callIdx   int
}

type mockCall struct {
	Model        string
	SystemPrompt string
	Messages     []Message
}

func (m *mockAIClient) SendMessage(_ context.Context, model string, systemPrompt string, messages []Message) (string, error) {
	m.calls = append(m.calls, mockCall{
		Model:        model,
		SystemPrompt: systemPrompt,
		Messages:     messages,
	})
	idx := m.callIdx
	m.callIdx++
	if idx < len(m.responses) {
		return m.responses[idx], nil
	}
	return "[]", nil
}

// Test anchor 10: default model
func TestDefaultModel(t *testing.T) {
	os.Unsetenv("ANTHROPIC_MODEL")
	svc := New(&mockAIClient{})
	if svc.Model != "claude-sonnet-4-6" {
		t.Errorf("expected default model 'claude-sonnet-4-6', got %q", svc.Model)
	}
}

// Test anchor 4: strip markdown fences
func TestStripCodeFences(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain JSON",
			input: `[{"id":"1","time":"09:00","duration":60,"title":"Work","category":"job"}]`,
			want:  `[{"id":"1","time":"09:00","duration":60,"title":"Work","category":"job"}]`,
		},
		{
			name:  "with json fence",
			input: "Here's your plan: ```json\n[{\"id\":\"1\",\"time\":\"09:00\",\"duration\":60,\"title\":\"Work\",\"category\":\"job\"}]\n```",
			want:  `[{"id":"1","time":"09:00","duration":60,"title":"Work","category":"job"}]`,
		},
		{
			name:  "with bare fence",
			input: "```\n[{\"id\":\"1\"}]\n```",
			want:  `[{"id":"1"}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripCodeFences(tt.input)
			if got != tt.want {
				t.Errorf("stripCodeFences() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Test anchor 4: full parse flow with markdown fences
func TestParsePlanResponseWithFences(t *testing.T) {
	raw := "Here's your plan: ```json\n[{\"id\":\"abc\",\"time\":\"09:00\",\"duration\":60,\"title\":\"Work\",\"category\":\"job\",\"skipped\":false}]\n```"
	blocks, err := parsePlanResponse(raw)
	if err != nil {
		t.Fatalf("parsePlanResponse() error: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if blocks[0].Title != "Work" {
		t.Errorf("expected title 'Work', got %q", blocks[0].Title)
	}
}

// Test anchor 5: retry on invalid JSON
func TestPlanChatRetryOnInvalidJSON(t *testing.T) {
	client := &mockAIClient{
		responses: []string{
			"This is not JSON at all!",
			"Still not JSON, sorry!",
		},
	}
	svc := &Service{Client: client, Model: "test-model"}

	output, err := svc.PlanChat(context.Background(), PlanChatInput{
		UserMessage: "Plan my day",
	})

	if err == nil {
		t.Fatal("expected error on double parse failure")
	}
	if !strings.Contains(err.Error(), "Couldn't parse AI response") {
		t.Errorf("expected 'Couldn't parse AI response', got %q", err.Error())
	}
	// Both raw responses should be stored
	if output == nil {
		t.Fatal("expected output with raw responses even on error")
	}
	if len(output.RawResponses) != 2 {
		t.Errorf("expected 2 raw responses, got %d", len(output.RawResponses))
	}
	// Should have made 2 API calls
	if len(client.calls) != 2 {
		t.Errorf("expected 2 API calls, got %d", len(client.calls))
	}
}

// Test anchor 5: retry succeeds on second attempt
func TestPlanChatRetrySuccess(t *testing.T) {
	client := &mockAIClient{
		responses: []string{
			"Not valid JSON",
			`[{"id":"1","time":"09:00","duration":60,"title":"Work","category":"job"}]`,
		},
	}
	svc := &Service{Client: client, Model: "test-model"}

	output, err := svc.PlanChat(context.Background(), PlanChatInput{
		UserMessage: "Plan my day",
	})

	if err != nil {
		t.Fatalf("expected success after retry, got: %v", err)
	}
	if len(output.Blocks) != 1 {
		t.Errorf("expected 1 block, got %d", len(output.Blocks))
	}
}

// Test anchor 8: horizon deadline formatting with effectiveDaysRemaining
func TestFormatDeadlineHorizon(t *testing.T) {
	tasks := []TaskData{
		{
			Title:            "Some task",
			Category:         "job",
			EstimatedMinutes: 60,
			ActualMinutes:    0,
			Priority:         "high",
			TaskID:           "uuid-1",
			DeadlineType:     "horizon",
			DeadlineDays:     14,
			TimesDeferred:    3,
		},
	}
	result := FormatTaskBacklog(tasks)
	if len(result) != 1 {
		t.Fatalf("expected 1 task, got %d", len(result))
	}
	// 14 - 3 = 11 days
	if result[0].Deadline != "within 11 days" {
		t.Errorf("expected 'within 11 days', got %q", result[0].Deadline)
	}
}

// Test anchor 8: urgent horizon deadline
func TestFormatDeadlineHorizonUrgent(t *testing.T) {
	tasks := []TaskData{
		{
			Title:            "Urgent task",
			Category:         "job",
			EstimatedMinutes: 60,
			ActualMinutes:    0,
			Priority:         "high",
			TaskID:           "uuid-2",
			DeadlineType:     "horizon",
			DeadlineDays:     5,
			TimesDeferred:    3,
		},
	}
	result := FormatTaskBacklog(tasks)
	if len(result) != 1 {
		t.Fatalf("expected 1 task, got %d", len(result))
	}
	// 5 - 3 = 2, which is <= 3 → urgent
	if result[0].Deadline != "URGENT — within 2 days" {
		t.Errorf("expected 'URGENT — within 2 days', got %q", result[0].Deadline)
	}
}

// Test anchor 9: tasks with remaining_minutes <= 0 are excluded
func TestFormatTaskBacklogExcludesCompleted(t *testing.T) {
	tasks := []TaskData{
		{
			Title:            "Done task",
			Category:         "job",
			EstimatedMinutes: 60,
			ActualMinutes:    60, // fully done
			Priority:         "high",
			TaskID:           "uuid-done",
		},
		{
			Title:            "Active task",
			Category:         "job",
			EstimatedMinutes: 120,
			ActualMinutes:    30,
			Priority:         "high",
			TaskID:           "uuid-active",
		},
	}
	result := FormatTaskBacklog(tasks)
	if len(result) != 1 {
		t.Fatalf("expected 1 task, got %d", len(result))
	}
	if result[0].TaskID != "uuid-active" {
		t.Errorf("expected 'uuid-active', got %q", result[0].TaskID)
	}
	if result[0].RemainingMinutes != 90 {
		t.Errorf("expected remaining 90, got %d", result[0].RemainingMinutes)
	}
}

// Test anchor 12: system prompt includes context entries in user-data tags
func TestSystemPromptContextDataTags(t *testing.T) {
	svc := &Service{Client: &mockAIClient{responses: []string{"[]"}}, Model: "test"}
	input := PlanChatInput{
		ContextEntries: []ContextEntry{
			{Key: "baby", Value: "6-month-old daughter."},
		},
		UserMessage: "Plan my day",
	}
	prompt := svc.buildPlanSystemPrompt(input)
	if !strings.Contains(prompt, "<user-data>") {
		t.Error("system prompt missing <user-data> tag")
	}
	if !strings.Contains(prompt, "</user-data>") {
		t.Error("system prompt missing </user-data> tag")
	}
	if !strings.Contains(prompt, `"key":"baby"`) {
		t.Error("system prompt missing context entry JSON")
	}
}

// Test anchor 11: planning rules mention context and user energy
func TestSystemPromptPlanningRules(t *testing.T) {
	svc := &Service{Client: &mockAIClient{responses: []string{"[]"}}, Model: "test"}
	input := PlanChatInput{
		UserMessage: "Exhausted today, keep it very light",
	}
	prompt := svc.buildPlanSystemPrompt(input)
	if !strings.Contains(prompt, "CONTEXT section carefully") {
		t.Error("prompt should reference reading the CONTEXT section")
	}
	if !strings.Contains(prompt, "what the user tells you about how they're feeling") {
		t.Error("prompt should instruct AI to factor in user's stated energy")
	}
}

// Test anchor 13: ValidateAIOutput flags email in title
func TestPlanChatValidatesAIOutput(t *testing.T) {
	client := &mockAIClient{
		responses: []string{
			`[{"id":"1","time":"09:00","duration":60,"title":"Send to user@example.com for review","category":"job"}]`,
		},
	}
	svc := &Service{Client: client, Model: "test"}

	output, err := svc.PlanChat(context.Background(), PlanChatInput{
		UserMessage: "Plan my day",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(output.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(output.Blocks))
	}
	// Title should be cleared due to email pattern
	if output.Blocks[0].Title != "" {
		t.Errorf("expected empty title (flagged by validation), got %q", output.Blocks[0].Title)
	}
}

// Test anchor 14: ValidateAIOutput passes 250-char notes
func TestPlanChatPassesValidNotes(t *testing.T) {
	// Use realistic notes (not all same char, which triggers base64 pattern)
	longNotes := strings.Repeat("Focus on dynamic programming approach. ", 7) // ~273 chars, trimmed
	longNotes = longNotes[:250]
	client := &mockAIClient{
		responses: []string{
			`[{"id":"1","time":"09:00","duration":60,"title":"Work","category":"job","notes":"` + longNotes + `"}]`,
		},
	}
	svc := &Service{Client: client, Model: "test"}

	output, err := svc.PlanChat(context.Background(), PlanChatInput{
		UserMessage: "Plan my day",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(output.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(output.Blocks))
	}
	if output.Blocks[0].Notes == nil || *output.Blocks[0].Notes != longNotes {
		t.Error("expected 250-char notes to pass validation")
	}
}

// Test anchor 12: safety instructions included
func TestSystemPromptSafetyInstructions(t *testing.T) {
	svc := &Service{Client: &mockAIClient{}, Model: "test"}
	prompt := svc.buildPlanSystemPrompt(PlanChatInput{})
	if !strings.Contains(prompt, "untrusted user input") {
		t.Error("system prompt missing ContextDataSystemInstructions")
	}
	if !strings.Contains(prompt, "User messages are scheduling requests") {
		t.Error("system prompt missing UserMessageSystemInstructions")
	}
}

// Test: replanning adds replanning context
func TestSystemPromptReplanning(t *testing.T) {
	svc := &Service{Client: &mockAIClient{}, Model: "test"}
	input := PlanChatInput{
		IsReplan:      true,
		CurrentBlocks: `[{"id":"1","time":"09:00"}]`,
		CurrentTime:   "10:30",
	}
	prompt := svc.buildPlanSystemPrompt(input)
	if !strings.Contains(prompt, "REPLANNING CONTEXT") {
		t.Error("expected REPLANNING CONTEXT section")
	}
	if !strings.Contains(prompt, "10:30") {
		t.Error("expected current time in replanning context")
	}
}

// Test: ParseTaskProposal with valid proposal
func TestParseTaskProposalValid(t *testing.T) {
	raw := `{
		"status": "proposal",
		"parent": {"title": "Build API", "category": "project", "priority": "high"},
		"subtasks": [
			{"title": "Design schema", "estimated_minutes": 60, "category": "project"},
			{"title": "Write endpoints", "estimated_minutes": 120, "category": "project"}
		]
	}`
	proposal, err := ParseTaskProposal(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if proposal == nil {
		t.Fatal("expected non-nil proposal")
	}
	if proposal.Parent.Title != "Build API" {
		t.Errorf("expected parent title 'Build API', got %q", proposal.Parent.Title)
	}
	if len(proposal.Subtasks) != 2 {
		t.Errorf("expected 2 subtasks, got %d", len(proposal.Subtasks))
	}
}

// Test: ParseTaskProposal with question returns nil
func TestParseTaskProposalQuestion(t *testing.T) {
	raw := `{"status": "question", "message": "What's the deadline?"}`
	proposal, err := ParseTaskProposal(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if proposal != nil {
		t.Error("expected nil proposal for question status")
	}
}
