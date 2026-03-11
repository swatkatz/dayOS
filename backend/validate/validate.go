package validate

import "fmt"

// Validate sanitizes and validates a string field against a TextFieldRule.
// Returns the sanitized string or a validation error.
func Validate(rule TextFieldRule, field, value string) (string, error) {
	r, ok := TextFieldRules[rule]
	if !ok {
		return "", fmt.Errorf("unknown validation rule for field %q", field)
	}

	// Apply sanitizers
	if r.StripNulls {
		value = SanitizerStripNulls.Replace(value)
	}
	if r.StripNewlines {
		value = SanitizerStripNewlines.Replace(value)
	}

	// Validate length
	if len([]rune(value)) > r.MaxLen {
		return "", fmt.Errorf("%s must be %d characters or fewer", field, r.MaxLen)
	}

	return value, nil
}
