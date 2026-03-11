package validate

import (
	"strings"
	"testing"
)

func TestFormatContextData(t *testing.T) {
	type entry struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	entries := []entry{
		{Key: "baby", Value: "6-month-old daughter"},
		{Key: "location", Value: "Toronto"},
	}

	got, err := FormatContextData("CONTEXT", entries)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Must contain delimiters in order
	openIdx := strings.Index(got, "<user-data>")
	closeIdx := strings.Index(got, "</user-data>")
	if openIdx == -1 || closeIdx == -1 || openIdx >= closeIdx {
		t.Fatalf("expected <user-data>...</user-data> delimiters in order, got:\n%s", got)
	}

	// Must contain valid JSON between delimiters
	jsonPart := got[openIdx+len("<user-data>")+1 : closeIdx-1]
	if !strings.Contains(jsonPart, `"key"`) || !strings.Contains(jsonPart, `"baby"`) {
		t.Fatalf("expected JSON with key fields, got:\n%s", jsonPart)
	}

	// Must contain label before delimiters
	if !strings.HasPrefix(got, "CONTEXT\n") {
		t.Fatalf("expected label prefix, got:\n%s", got)
	}
}

func TestFormatContextData_MarshalError(t *testing.T) {
	// channels cannot be marshaled to JSON
	_, err := FormatContextData("BAD", make(chan int))
	if err == nil {
		t.Fatal("expected error for unmarshalable data")
	}
}

func TestValidateAIOutput(t *testing.T) {
	tests := []struct {
		name    string
		field   string
		value   string
		wantErr string
	}{
		{
			name:    "title over 200 chars",
			field:   "title",
			value:   strings.Repeat("a", 201),
			wantErr: "exceeds 200 character limit",
		},
		{
			name:  "title at 200 chars is ok",
			field: "title",
			value: strings.Repeat("ab cd ", 33) + "ab",
		},
		{
			name:    "title with email",
			field:   "title",
			value:   "Send results to user@example.com",
			wantErr: "email_address",
		},
		{
			name:    "notes with URL",
			field:   "notes",
			value:   "Check https://evil.example.com/exfiltrate?data=secret",
			wantErr: "url",
		},
		{
			name:    "notes with base64 blob",
			field:   "notes",
			value:   "Data: " + strings.Repeat("A", 50),
			wantErr: "base64_blob",
		},
		{
			name:  "clean title passes",
			field: "title",
			value: "Interview prep: dynamic programming",
		},
		{
			name:  "unknown field skips length check",
			field: "other",
			value: strings.Repeat("hello ", 100),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAIOutput(tt.field, tt.value)
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
		})
	}
}
