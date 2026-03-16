package planner

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"dayos/identity"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/jackc/pgx/v5/pgtype"
)

// WithUserCap decorates a PlannerService with per-user API key fallback and daily usage caps.
// It satisfies the same PlannerService interface used by resolvers.
type WithUserCap struct {
	inner      *Service
	queries    userCapQueries
	systemKey  string
	defaultCap int32 // fallback cap from DAILY_AI_CAP env var
}

// userCapQueries is the subset of db.Queries needed by the cap decorator.
type userCapQueries interface {
	GetAICallsToday(ctx context.Context, id pgtype.UUID) (int32, error)
	IncrementAICallsToday(ctx context.Context, id pgtype.UUID) (int32, error)
}

// NewWithUserCap creates a cap-enforced planner decorator.
func NewWithUserCap(inner *Service, queries userCapQueries, systemKey string) *WithUserCap {
	defaultCap := int32(50)
	if env := os.Getenv("DAILY_AI_CAP"); env != "" {
		if v, err := strconv.Atoi(env); err == nil && v > 0 {
			defaultCap = int32(v)
		}
	}
	return &WithUserCap{
		inner:      inner,
		queries:    queries,
		systemKey:  systemKey,
		defaultCap: defaultCap,
	}
}

func (w *WithUserCap) checkCap(ctx context.Context, user identity.User) error {
	calls, err := w.queries.GetAICallsToday(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("checking AI usage: %w", err)
	}
	// Use per-user cap if set, otherwise fall back to default
	cap := w.defaultCap
	if user.DailyAICap > 0 {
		cap = user.DailyAICap
	}
	if calls >= cap {
		return fmt.Errorf("Daily AI usage limit reached. Try again tomorrow.")
	}
	return nil
}

func (w *WithUserCap) clientForUser(user identity.User) *Service {
	if user.AnthropicAPIKey != nil && *user.AnthropicAPIKey != "" {
		client := NewAnthropicClientWithKey(*user.AnthropicAPIKey)
		return &Service{Client: client, Model: w.inner.Model}
	}
	return w.inner
}

// PlanChat implements PlannerService with cap enforcement and per-user key fallback.
func (w *WithUserCap) PlanChat(ctx context.Context, input PlanChatInput) (*PlanChatOutput, error) {
	user := identity.MustFromContext(ctx)

	if err := w.checkCap(ctx, user); err != nil {
		return nil, err
	}

	svc := w.clientForUser(user)
	result, err := svc.PlanChat(ctx, input)
	if err != nil {
		return nil, err
	}

	if _, incErr := w.queries.IncrementAICallsToday(ctx, user.ID); incErr != nil {
		log.Printf("usercap: incrementing AI calls for user %s: %v", user.ClerkID, incErr)
	}
	return result, nil
}

// TaskChat implements PlannerService with cap enforcement and per-user key fallback.
func (w *WithUserCap) TaskChat(ctx context.Context, history []Message, userMessage string, userName string) (*TaskChatOutput, error) {
	user := identity.MustFromContext(ctx)

	if err := w.checkCap(ctx, user); err != nil {
		return nil, err
	}

	svc := w.clientForUser(user)
	result, err := svc.TaskChat(ctx, history, userMessage, userName)
	if err != nil {
		return nil, err
	}

	if _, incErr := w.queries.IncrementAICallsToday(ctx, user.ID); incErr != nil {
		log.Printf("usercap: incrementing AI calls for user %s: %v", user.ClerkID, incErr)
	}
	return result, nil
}

// NewAnthropicClientWithKey creates an Anthropic client using the provided API key.
func NewAnthropicClientWithKey(apiKey string) *AnthropicClient {
	return &AnthropicClient{
		client: anthropic.NewClient(option.WithAPIKey(apiKey)),
	}
}
