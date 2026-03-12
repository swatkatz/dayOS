package graph

import (
	"context"
	"strings"
	"testing"

	"dayos/graph/model"

	"github.com/99designs/gqlgen/graphql"

)

func TestValidateDirective(t *testing.T) {
	handler := ValidateDirective()

	tests := []struct {
		name     string
		rule     model.TextFieldRule
		value    any
		want     any
		wantErr  string
	}{
		{
			name:  "SingleLine strips newlines and nulls",
			rule:  model.TextFieldRuleSingleLine,
			value: "Hello\nWorld\r\x00",
			want:  "HelloWorld",
		},
		{
			name:    "SingleLine rejects over 255 chars",
			rule:    model.TextFieldRuleSingleLine,
			value:   strings.Repeat("a", 256),
			wantErr: "must be 255 characters or fewer",
		},
		{
			name:  "SingleLine accepts 255 chars",
			rule:  model.TextFieldRuleSingleLine,
			value: strings.Repeat("a", 255),
			want:  strings.Repeat("a", 255),
		},
		{
			name:  "SingleLineShort strips nulls",
			rule:  model.TextFieldRuleSingleLineShort,
			value: "Short\x00key",
			want:  "Shortkey",
		},
		{
			name:    "SingleLineShort rejects 101 chars",
			rule:    model.TextFieldRuleSingleLineShort,
			value:   strings.Repeat("a", 101),
			wantErr: "must be 100 characters or fewer",
		},
		{
			name:  "PlainText preserves newlines",
			rule:  model.TextFieldRulePlainText,
			value: "line1\nline2\rline3",
			want:  "line1\nline2\rline3",
		},
		{
			name:  "ChatMessage allows up to 5000",
			rule:  model.TextFieldRuleChatMessage,
			value: strings.Repeat("a", 5000),
			want:  strings.Repeat("a", 5000),
		},
		{
			name:    "ChatMessage rejects over 5000",
			rule:    model.TextFieldRuleChatMessage,
			value:   strings.Repeat("a", 5001),
			wantErr: "must be 5000 characters or fewer",
		},
		{
			name:  "nil value passes through",
			rule:  model.TextFieldRuleSingleLine,
			value: nil,
			want:  nil,
		},
		{
			name:  "empty string passes through",
			rule:  model.TextFieldRuleSingleLine,
			value: "",
			want:  "",
		},
		{
			name:  "non-string value passes through",
			rule:  model.TextFieldRuleSingleLine,
			value: 42,
			want:  42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := graphql.WithPathContext(context.Background(), &graphql.PathContext{
				Field: strPtr("testfield"),
			})

			next := func(ctx context.Context) (any, error) {
				return tt.value, nil
			}

			got, err := handler(ctx, nil, next, tt.rule)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}


func strPtr(s string) *string {
	return &s
}
