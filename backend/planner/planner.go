package planner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"dayos/validate"

	"github.com/anthropics/anthropic-sdk-go"
)

// AIClient abstracts the Anthropic API for testability.
type AIClient interface {
	SendMessage(ctx context.Context, model string, systemPrompt string, messages []Message) (string, error)
}

// Message represents a chat message with role and content.
type Message struct {
	Role    string // "user" or "assistant"
	Content string
}

// Service handles all AI interactions for plan generation and task scoping.
type Service struct {
	Client AIClient
	Model  string
}

// New creates a new planner service with the given AI client.
func New(client AIClient) *Service {
	model := os.Getenv("ANTHROPIC_MODEL")
	if model == "" {
		model = "claude-sonnet-4-6"
	}
	return &Service{Client: client, Model: model}
}

// --- Data types for prompt building ---

type ContextEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type RoutineInfo struct {
	RoutineID     string `json:"routine_id"`
	Title         string `json:"title"`
	Category      string `json:"category"`
	DurationMin   int    `json:"duration_min"`
	PreferredTime string `json:"preferred_time,omitempty"`
	ExactTime     string `json:"exact_time,omitempty"`
}

type TaskInfo struct {
	Title            string `json:"title"`
	Category         string `json:"category"`
	RemainingMinutes int    `json:"remaining_minutes"`
	EstimatedMinutes int    `json:"estimated_minutes"`
	Priority         string `json:"priority"`
	Deadline         string `json:"deadline"`
	TaskID           string `json:"task_id"`
}

type CarryOverTask struct {
	Title             string `json:"title"`
	Category          string `json:"category"`
	TimesDeferred     int    `json:"times_deferred"`
	EffectiveDeadline string `json:"effective_deadline"`
	TaskID            string `json:"task_id"`
}

// CalendarEventInfo is calendar event data for the planner prompt.
type CalendarEventInfo struct {
	Title     string `json:"title"`
	StartTime string `json:"start_time"`
	Duration  int    `json:"duration_min"`
	AllDay    bool   `json:"all_day"`
}

// --- Plan chat ---

// PlanChatInput contains all data needed to build a plan prompt.
type PlanChatInput struct {
	ContextEntries []ContextEntry
	Routines       []RoutineInfo
	Tasks          []TaskInfo
	CarryOverTasks []CarryOverTask
	CalendarEvents []CalendarEventInfo
	History        []Message
	UserMessage    string
	CurrentBlocks  string // JSON of remaining (not done, not skipped) blocks for replanning
	CompletedBlocks string // JSON of done blocks for AI context during replanning
	CurrentTime    string // HH:MM for replanning
	IsReplan       bool
}

// Block represents a time block in a day plan, as returned by the AI.
type Block struct {
	ID        string  `json:"id"`
	Time      string  `json:"time"`
	Duration  int     `json:"duration"`
	Title     string  `json:"title"`
	Category  string  `json:"category"`
	TaskID    *string `json:"task_id"`
	RoutineID *string `json:"routine_id"`
	Notes     *string `json:"notes"`
	Skipped   bool    `json:"skipped"`
}

// PlanChatOutput contains the parsed blocks and raw AI responses.
type PlanChatOutput struct {
	Blocks       []Block
	RawResponses []string // all raw responses (1 on success, 2 on retry)
}

const retryPrompt = `Your previous response was not valid JSON. You MUST respond with ONLY a valid JSON array/object as specified in the system prompt. No markdown, no explanation, no code fences. Just the raw JSON.`

// PlanChat sends a plan message to the AI and returns parsed blocks.
func (s *Service) PlanChat(ctx context.Context, input PlanChatInput) (*PlanChatOutput, error) {
	systemPrompt := s.buildPlanSystemPrompt(input)
	fmt.Println("=== SYSTEM PROMPT ===")
	fmt.Println(systemPrompt)
	fmt.Println("=== END SYSTEM PROMPT ===")

	messages := make([]Message, 0, len(input.History)+1)
	messages = append(messages, input.History...)
	messages = append(messages, Message{Role: "user", Content: input.UserMessage})

	rawResp, err := s.Client.SendMessage(ctx, s.Model, systemPrompt, messages)
	if err != nil {
		return nil, fmt.Errorf("calling AI: %w", err)
	}

	blocks, parseErr := parsePlanResponse(rawResp)
	if parseErr != nil {
		// Retry once with stricter prompt
		retryMessages := make([]Message, len(messages))
		copy(retryMessages, messages)
		retryMessages = append(retryMessages, Message{Role: "assistant", Content: rawResp})
		retryMessages = append(retryMessages, Message{Role: "user", Content: retryPrompt})

		retryResp, retryErr := s.Client.SendMessage(ctx, s.Model, systemPrompt, retryMessages)
		if retryErr != nil {
			return &PlanChatOutput{RawResponses: []string{rawResp}}, fmt.Errorf("Couldn't parse AI response, please try again")
		}

		blocks, parseErr = parsePlanResponse(retryResp)
		if parseErr != nil {
			return &PlanChatOutput{RawResponses: []string{rawResp, retryResp}}, fmt.Errorf("Couldn't parse AI response, please try again")
		}
		rawResp = retryResp
	}

	// Validate AI output on each block
	for i := range blocks {
		if err := validate.ValidateAIOutput("title", blocks[i].Title); err != nil {
			blocks[i].Title = ""
		}
		if blocks[i].Notes != nil {
			if err := validate.ValidateAIOutput("notes", *blocks[i].Notes); err != nil {
				blocks[i].Notes = nil
			}
		}
	}

	return &PlanChatOutput{Blocks: blocks, RawResponses: []string{rawResp}}, nil
}

// --- Task scoping chat ---

// TaskChatOutput contains the raw AI response for task scoping.
type TaskChatOutput struct {
	RawResponse string
}

// TaskChat sends a message in a task scoping conversation.
func (s *Service) TaskChat(ctx context.Context, history []Message, userMessage string) (*TaskChatOutput, error) {
	messages := make([]Message, 0, len(history)+1)
	messages = append(messages, history...)
	messages = append(messages, Message{Role: "user", Content: userMessage})

	rawResp, err := s.Client.SendMessage(ctx, s.Model, taskScopingSystemPrompt, messages)
	if err != nil {
		return nil, fmt.Errorf("calling AI: %w", err)
	}

	// Validate the response is parseable JSON
	cleaned := stripCodeFences(strings.TrimSpace(rawResp))
	var parsed map[string]any
	if err := json.Unmarshal([]byte(cleaned), &parsed); err != nil {
		// Retry once
		retryMessages := make([]Message, len(messages))
		copy(retryMessages, messages)
		retryMessages = append(retryMessages, Message{Role: "assistant", Content: rawResp})
		retryMessages = append(retryMessages, Message{Role: "user", Content: retryPrompt})

		retryResp, retryErr := s.Client.SendMessage(ctx, s.Model, taskScopingSystemPrompt, retryMessages)
		if retryErr != nil {
			return &TaskChatOutput{RawResponse: rawResp}, nil // store raw even on failure
		}
		cleaned = stripCodeFences(strings.TrimSpace(retryResp))
		if err := json.Unmarshal([]byte(cleaned), &parsed); err != nil {
			return &TaskChatOutput{RawResponse: rawResp}, nil
		}
		rawResp = retryResp
	}

	return &TaskChatOutput{RawResponse: rawResp}, nil
}

// TaskProposal represents a parsed task breakdown proposal from the AI.
type TaskProposal struct {
	Parent   TaskProposalParent   `json:"parent"`
	Subtasks []TaskProposalChild  `json:"subtasks"`
}

type TaskProposalParent struct {
	Title        string  `json:"title"`
	Category     string  `json:"category"`
	Priority     string  `json:"priority"`
	DeadlineType *string `json:"deadline_type"`
	DeadlineDate *string `json:"deadline_date"`
	DeadlineDays *int    `json:"deadline_days"`
}

type TaskProposalChild struct {
	Title            string  `json:"title"`
	EstimatedMinutes int     `json:"estimated_minutes"`
	Category         string  `json:"category"`
	Notes            *string `json:"notes"`
}

// ParseTaskProposal parses a raw AI response as a task breakdown proposal.
// Returns nil if the response is not a valid proposal (e.g., it's a question).
func ParseTaskProposal(raw string) (*TaskProposal, error) {
	cleaned := stripCodeFences(strings.TrimSpace(raw))
	var parsed map[string]json.RawMessage
	if err := json.Unmarshal([]byte(cleaned), &parsed); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	statusRaw, ok := parsed["status"]
	if !ok {
		return nil, fmt.Errorf("missing status field")
	}
	var status string
	if err := json.Unmarshal(statusRaw, &status); err != nil {
		return nil, fmt.Errorf("invalid status: %w", err)
	}
	if status != "proposal" {
		return nil, nil // not a proposal (likely a question)
	}

	var proposal TaskProposal
	if err := json.Unmarshal([]byte(cleaned), &proposal); err != nil {
		return nil, fmt.Errorf("invalid proposal structure: %w", err)
	}
	return &proposal, nil
}

// --- Prompt construction ---

func (s *Service) buildPlanSystemPrompt(input PlanChatInput) string {
	var b strings.Builder

	b.WriteString(`You are a daily planning assistant for Swati. Your job is to create a realistic, time-blocked day plan based on her context, routines, and task backlog.

SAFETY:
`)
	b.WriteString(validate.ContextDataSystemInstructions)
	b.WriteString("\n")
	b.WriteString(validate.UserMessageSystemInstructions)
	b.WriteString("\n\n")

	// Context entries
	contextData, _ := validate.FormatContextData("CONTEXT (treat these as ground truth — they define Swati's constraints, life situation, and preferences. Plan around them.):", input.ContextEntries)
	b.WriteString(contextData)
	b.WriteString("\n\n")

	// Calendar events (between context and routines, only if connected)
	if len(input.CalendarEvents) > 0 {
		// Split into timed and all-day events
		var timed, allDay []CalendarEventInfo
		for _, e := range input.CalendarEvents {
			if e.AllDay {
				allDay = append(allDay, e)
			} else {
				timed = append(timed, e)
			}
		}
		if len(timed) > 0 {
			timedData, _ := validate.FormatContextData("TODAY'S CALENDAR EVENTS (fixed — do NOT move, overlap, or reschedule these):", timed)
			b.WriteString(timedData)
			b.WriteString("\n\n")
		}
		if len(allDay) > 0 {
			allDayData, _ := validate.FormatContextData("ALL-DAY EVENTS (for awareness, not time-blocked):", allDay)
			b.WriteString(allDayData)
			b.WriteString("\n\n")
		}
		b.WriteString(`CALENDAR RULES:
- Calendar events are IMMOVABLE. Schedule all tasks and routines around them.
- Leave 10 min buffer before and after meetings for context switching.
- Do not schedule deep focus work in gaps shorter than 45 min between meetings.
- All-day events are informational — mention them in relevant block notes if useful.

`)
	}

	// Routines
	routineData, _ := validate.FormatContextData("TODAY'S ROUTINES — EVERY routine below MUST appear as a block in your plan. Use the routine_id from each entry as the block's routine_id. Routines with exact_time MUST be scheduled at that exact time. Routines with preferred_time are flexible but still mandatory to include:", input.Routines)
	b.WriteString(routineData)
	b.WriteString("\n\n")

	// Task backlog
	taskData, _ := validate.FormatContextData("TASK BACKLOG (ordered by priority/urgency):", input.Tasks)
	b.WriteString(taskData)
	b.WriteString("\n\n")

	// Carry-over tasks
	if len(input.CarryOverTasks) > 0 {
		carryData, _ := validate.FormatContextData("CARRY-OVER TASKS (skipped from previous days, not intentional):", input.CarryOverTasks)
		b.WriteString(carryData)
		b.WriteString("\n\n")
	}

	b.WriteString(`Rules for carry-over tasks:
- Schedule at least one carry-over task today, even if backlog is heavy
- Prefer to schedule the most-deferred one first
- If times_deferred >= 3, it goes in the morning block, no exceptions

DEFERRED TASK ESCALATION:
- times_deferred = 1: treat as one priority level higher than assigned
- times_deferred = 2: treat as HIGH priority regardless of assigned priority
- times_deferred >= 3: flag in the plan title with "⚑ OVERDUE" and schedule in the first available slot of the day

PLANNING RULES:
- Read the CONTEXT section carefully. It tells you when deep work is possible, when family time is, energy limits, and scheduling constraints. Follow it.
- Routines with an "exact_time" field MUST be scheduled at that exact time — treat it as a hard constraint, not a preference. Routines with only a "preferred_time" (morning/midday/afternoon/evening/any) should be treated as a preference, not a constraint. If the preferred slot is packed with higher-priority work, move the routine to another available slot. Routines marked "any" should fill gaps in the schedule.
- Hardest cognitive tasks go in the earliest available deep work slot.
- Always leave 15-min buffers between intense 90+ min sessions.
- Be SPECIFIC in block titles (e.g., "Meta LC: Coin Change II — bottom-up DP" not "Interview prep").
- Don't schedule more than is realistic given the energy constraints in CONTEXT and what the user tells you about how they're feeling today.
- A task with remaining_minutes > block duration CAN be scheduled — schedule what fits today, the rest goes on subsequent days.
- Never schedule anything in a past time slot.

CATEGORIES (use exactly one per block):
- job: paid work, meetings, deliverables
- interview: interview prep, leetcode, system design
- project: personal/side projects
- meal: cooking, meal prep, grocery shopping, eating
- baby: childcare, baby-related tasks
- exercise: workouts, walks, physical activity
- admin: errands, chores, appointments, anything that doesn't fit above

RESPONSE FORMAT:
Respond ONLY with a JSON array. No explanation. No markdown. Just the array.
Each element: { "id": "uuid-v4", "time": "HH:MM", "duration": 60, "title": "...", "category": "interview", "task_id": "uuid-or-null", "routine_id": "uuid-or-null", "notes": "...", "skipped": false }`)

	// Replanning context
	if input.IsReplan {
		b.WriteString(fmt.Sprintf(`

REPLANNING CONTEXT:
These blocks are already COMPLETED — do not include them in your response:
%s

These are the remaining unfinished blocks to reschedule:
%s

Current time: %s

REPLANNING RULES:
- Return ONLY new/rescheduled blocks from current time onward.
- Do NOT include completed or skipped blocks in your response.
- All blocks must have "time" >= current time.
- The user is asking to adjust the remaining schedule.`, input.CompletedBlocks, input.CurrentBlocks, input.CurrentTime))
	}

	return b.String()
}

const taskScopingSystemPrompt = `You are helping Swati break down a goal into concrete, schedulable tasks.

Your job:
1. Ask clarifying questions to understand the scope (what needs to be done, what's the deliverable, what are the dependencies between steps)
2. Ask about the deadline
3. Propose a breakdown of subtasks, each with:
   - A specific, actionable title
   - An estimated duration in minutes
   - The category (job|interview|project|meal|baby|exercise|admin)
   - Any notes

When proposing the breakdown, respond with a JSON object:
{
  "status": "proposal",
  "parent": { "title": "...", "category": "...", "priority": "...",
              "deadline_type": "hard|horizon|null", "deadline_date": "YYYY-MM-DD|null",
              "deadline_days": N|null },
  "subtasks": [
    { "title": "...", "estimated_minutes": N, "category": "...", "notes": "..." },
    ...
  ]
}

When asking questions, respond with:
{ "status": "question", "message": "..." }

Keep it conversational. Don't ask more than 2-3 questions before proposing.
Adjust if Swati gives feedback on the proposal.`

// --- JSON parsing ---

var codeBlockRegex = regexp.MustCompile("(?s)```(?:json)?\\s*\n?(.*?)\\s*```")

// stripCodeFences removes markdown code fences from a string.
func stripCodeFences(s string) string {
	matches := codeBlockRegex.FindStringSubmatch(s)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return s
}

// parsePlanResponse parses a raw AI response into blocks.
func parsePlanResponse(raw string) ([]Block, error) {
	cleaned := stripCodeFences(strings.TrimSpace(raw))
	var blocks []Block
	if err := json.Unmarshal([]byte(cleaned), &blocks); err != nil {
		return nil, fmt.Errorf("parsing plan blocks: %w", err)
	}

	// Validate required fields
	var valid []Block
	for _, b := range blocks {
		if b.ID == "" || b.Time == "" || b.Title == "" || b.Category == "" {
			continue
		}
		valid = append(valid, b)
	}
	return valid, nil
}

// FormatTaskBacklog converts raw task data into TaskInfo entries for the prompt.
// It handles parent/subtask filtering, remaining minutes computation, and deadline formatting.
func FormatTaskBacklog(tasks []TaskData) []TaskInfo {
	var result []TaskInfo
	for _, t := range tasks {
		remaining := t.EstimatedMinutes - t.ActualMinutes
		if remaining <= 0 {
			continue
		}
		// Skip parent tasks (they have subtasks and no estimated_minutes typically)
		// The spec says: don't include parent itself, include its incomplete subtasks individually
		// Parents are identified by having subtasks (parent_id IS NULL and no estimated_minutes)
		// But ListSchedulableTasks already filters: parent_id IS NOT NULL OR (parent_id IS NULL AND estimated_minutes IS NOT NULL)
		// So parents without estimated_minutes are already excluded.

		info := TaskInfo{
			Title:            t.Title,
			Category:         t.Category,
			RemainingMinutes: remaining,
			EstimatedMinutes: t.EstimatedMinutes,
			Priority:         t.Priority,
			TaskID:           t.TaskID,
		}

		// Format deadline
		info.Deadline = formatDeadline(t)

		result = append(result, info)
	}
	return result
}

// TaskData is the raw task data passed to FormatTaskBacklog.
type TaskData struct {
	Title            string
	Category         string
	EstimatedMinutes int
	ActualMinutes    int
	Priority         string
	TaskID           string
	DeadlineType     string // "hard", "horizon", or ""
	DeadlineDate     string // YYYY-MM-DD for hard deadlines
	DeadlineDays     int    // for horizon deadlines
	TimesDeferred    int
}

func formatDeadline(t TaskData) string {
	switch t.DeadlineType {
	case "hard":
		if t.DeadlineDate != "" {
			return fmt.Sprintf("due %s", t.DeadlineDate)
		}
	case "horizon":
		effective := t.DeadlineDays - t.TimesDeferred
		if effective <= 0 {
			effective = 1
		}
		if effective <= 3 {
			return fmt.Sprintf("URGENT — within %d days", effective)
		}
		return fmt.Sprintf("within %d days", effective)
	}
	return ""
}

// --- Anthropic SDK client implementation ---

// AnthropicClient implements AIClient using the Anthropic Go SDK.
type AnthropicClient struct {
	client anthropic.Client
}

// NewAnthropicClient creates a new client that reads ANTHROPIC_API_KEY from the environment.
func NewAnthropicClient() *AnthropicClient {
	return &AnthropicClient{
		client: anthropic.NewClient(),
	}
}

func (c *AnthropicClient) SendMessage(ctx context.Context, model string, systemPrompt string, messages []Message) (string, error) {
	apiMessages := make([]anthropic.MessageParam, 0, len(messages))
	for _, m := range messages {
		switch m.Role {
		case "user":
			apiMessages = append(apiMessages, anthropic.NewUserMessage(anthropic.NewTextBlock(m.Content)))
		case "assistant":
			apiMessages = append(apiMessages, anthropic.NewAssistantMessage(anthropic.NewTextBlock(m.Content)))
		}
	}

	resp, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: 4096,
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt},
		},
		Messages: apiMessages,
	})
	if err != nil {
		return "", fmt.Errorf("anthropic API error: %w", err)
	}

	// Extract text from response
	var text strings.Builder
	for _, block := range resp.Content {
		if block.Type == "text" {
			text.WriteString(block.Text)
		}
	}
	return text.String(), nil
}
