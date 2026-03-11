package validate

import (
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		rule    TextFieldRule
		field   string
		value   string
		want    string
		wantErr string
	}{
		{
			name:  "SingleLine strips newlines and nulls",
			rule:  SingleLine,
			field: "title",
			value: "Hello\nWorld\r\x00",
			want:  "HelloWorld",
		},
		{
			name:    "SingleLine rejects 256 chars",
			rule:    SingleLine,
			field:   "title",
			value:   strings.Repeat("a", 256),
			wantErr: "title must be 255 characters or fewer",
		},
		{
			name:  "SingleLine accepts 255 chars",
			rule:  SingleLine,
			field: "title",
			value: strings.Repeat("a", 255),
			want:  strings.Repeat("a", 255),
		},
		{
			name:  "SingleLineShort strips nulls",
			rule:  SingleLineShort,
			field: "key",
			value: "Short\x00key",
			want:  "Shortkey",
		},
		{
			name:    "SingleLineShort rejects 101 chars",
			rule:    SingleLineShort,
			field:   "key",
			value:   strings.Repeat("a", 101),
			wantErr: "key must be 100 characters or fewer",
		},
		{
			name:  "PlainText preserves newlines",
			rule:  PlainText,
			field: "notes",
			value: "line1\nline2\rline3",
			want:  "line1\nline2\rline3",
		},
		{
			name:  "PlainText strips nulls",
			rule:  PlainText,
			field: "notes",
			value: "hello\x00world",
			want:  "helloworld",
		},
		{
			name:  "ChatMessage allows long text",
			rule:  ChatMessage,
			field: "message",
			value: strings.Repeat("a", 5000),
			want:  strings.Repeat("a", 5000),
		},
		{
			name:    "ChatMessage rejects over 5000",
			rule:    ChatMessage,
			field:   "message",
			value:   strings.Repeat("a", 5001),
			wantErr: "message must be 5000 characters or fewer",
		},
		{
			name:  "length measured in runes not bytes",
			rule:  SingleLine,
			field: "title",
			// 255 multi-byte runes should pass
			value: strings.Repeat("é", 255),
			want:  strings.Repeat("é", 255),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Validate(tt.rule, tt.field, tt.value)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error %q, got %q", tt.wantErr, err.Error())
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
