package validate

import (
	"encoding/json"
	"fmt"
	"regexp"
)

// FormatContextData wraps structured data in delimiters for safe prompt embedding.
// The data is JSON-serialized inside <user-data> tags.
func FormatContextData(label string, data any) (string, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("marshaling %s for prompt: %w", label, err)
	}
	return fmt.Sprintf("%s\n%s\n%s\n%s",
		label,
		ContextDataDelimiterOpen,
		string(b),
		ContextDataDelimiterClose,
	), nil
}

// ValidateAIOutput checks an AI-generated string field for length limits
// and exfiltration patterns. Returns an error if the value is suspicious.
func ValidateAIOutput(field, value string) error {
	if limit, ok := AIOutputLimits[field]; ok {
		if len([]rune(value)) > limit {
			return fmt.Errorf("AI output %q exceeds %d character limit", field, limit)
		}
	}
	for name, pattern := range ExfiltrationPatterns {
		matched, err := regexp.MatchString(pattern, value)
		if err != nil {
			return fmt.Errorf("checking pattern %s: %w", name, err)
		}
		if matched {
			return fmt.Errorf("AI output %q contains suspicious pattern (%s)", field, name)
		}
	}
	return nil
}
