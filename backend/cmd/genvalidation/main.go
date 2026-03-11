package main

import (
	"encoding/json"
	"fmt"
	"go/format"
	"os"
	"slices"
	"strings"
)

type Rules struct {
	Version           string                       `json:"version"`
	TextFieldRules    map[string]TextFieldRuleDef   `json:"textFieldRules"`
	PromptRoles       map[string]PromptRoleDef      `json:"promptRoles"`
	Sanitizers        map[string]SanitizerDef       `json:"sanitizers"`
	RejectionPatterns map[string]RejectionPatternDef `json:"rejectionPatterns"`
}

type TextFieldRuleDef struct {
	Description string            `json:"description"`
	Sanitize    []string          `json:"sanitize"`
	Validate    []json.RawMessage `json:"validate"`
}

type PromptRoleDef struct {
	Description        string            `json:"description"`
	Embedding          *EmbeddingDef     `json:"embedding"`
	SystemInstructions []string          `json:"systemInstructions"`
	Validate           []json.RawMessage `json:"validate"`
	OnFailure          string            `json:"onFailure"`
}

type EmbeddingDef struct {
	Format    string       `json:"format"`
	Delimiter *DelimiterDef `json:"delimiter"`
	Placement string       `json:"placement"`
}

type DelimiterDef struct {
	Open  string `json:"open"`
	Close string `json:"close"`
}

type SanitizerDef struct {
	Description string   `json:"description"`
	Removes     []string `json:"removes"`
}

type RejectionPatternDef struct {
	Description string `json:"description"`
	Regex       string `json:"regex"`
}

// snakeToCamel converts UPPER_SNAKE to CamelCase
func snakeToCamel(s string) string {
	parts := strings.Split(strings.ToLower(s), "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}

func main() {
	data, err := os.ReadFile("../validation-rules.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "reading validation-rules.json: %v\n", err)
		os.Exit(1)
	}

	var rules Rules
	if err := json.Unmarshal(data, &rules); err != nil {
		fmt.Fprintf(os.Stderr, "parsing validation-rules.json: %v\n", err)
		os.Exit(1)
	}

	if err := generateGraphQL(rules); err != nil {
		fmt.Fprintf(os.Stderr, "generating GraphQL: %v\n", err)
		os.Exit(1)
	}

	if err := generateGo(rules); err != nil {
		fmt.Fprintf(os.Stderr, "generating Go: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Generated backend/graph/validation.graphqls")
	fmt.Println("Generated backend/validate/generated.go")
}

// Ordered rule names for deterministic output
var textFieldRuleOrder = []string{"SINGLE_LINE", "SINGLE_LINE_SHORT", "PLAIN_TEXT", "CHAT_MESSAGE"}
var promptRoleOrder = []string{"CONTEXT_DATA", "USER_MESSAGE", "AI_OUTPUT"}

func generateGraphQL(rules Rules) error {
	var b strings.Builder
	b.WriteString("# AUTO-GENERATED from validation-rules.json — do not edit manually\n\n")

	b.WriteString("enum TextFieldRule {\n")
	for _, name := range textFieldRuleOrder {
		def := rules.TextFieldRules[name]
		sanitizers := strings.Join(def.Sanitize, ", ")
		maxLen := extractMaxLength(def.Validate)
		b.WriteString(fmt.Sprintf("  \"\"\"%s. Sanitizers: %s, max %d.\"\"\"\n", def.Description, sanitizers, maxLen))
		b.WriteString(fmt.Sprintf("  %s\n", name))
	}
	b.WriteString("}\n\n")

	b.WriteString("enum PromptRole {\n")
	for _, name := range promptRoleOrder {
		def := rules.PromptRoles[name]
		b.WriteString(fmt.Sprintf("  \"\"\"%s\"\"\"\n", def.Description))
		b.WriteString(fmt.Sprintf("  %s\n", name))
	}
	b.WriteString("}\n\n")

	b.WriteString("directive @validate(rule: TextFieldRule!) on INPUT_FIELD_DEFINITION | ARGUMENT_DEFINITION\n")
	b.WriteString("directive @prompt(role: PromptRole!) on INPUT_FIELD_DEFINITION | FIELD_DEFINITION | ARGUMENT_DEFINITION\n")

	return os.WriteFile("graph/validation.graphqls", []byte(b.String()), 0644)
}

func generateGo(rules Rules) error {
	var b strings.Builder
	b.WriteString("// AUTO-GENERATED from validation-rules.json — do not edit manually\n")
	b.WriteString("package validate\n\n")
	b.WriteString("import \"strings\"\n\n")

	// Text field rule enum
	b.WriteString("// --- Text field rule enum ---\n\n")
	b.WriteString("type TextFieldRule int\n\n")
	b.WriteString("const (\n")
	for i, name := range textFieldRuleOrder {
		camel := snakeToCamel(name)
		if i == 0 {
			b.WriteString(fmt.Sprintf("\t%s TextFieldRule = iota\n", camel))
		} else {
			b.WriteString(fmt.Sprintf("\t%s\n", camel))
		}
	}
	b.WriteString(")\n\n")

	// Text field rule definitions
	b.WriteString("// --- Text field rule definitions ---\n\n")
	b.WriteString("type TextRule struct {\n")
	b.WriteString("\tMaxLen        int\n")
	b.WriteString("\tStripNewlines bool\n")
	b.WriteString("\tStripNulls    bool\n")
	b.WriteString("}\n\n")

	b.WriteString("var TextFieldRules = map[TextFieldRule]TextRule{\n")
	for _, name := range textFieldRuleOrder {
		def := rules.TextFieldRules[name]
		camel := snakeToCamel(name)
		maxLen := extractMaxLength(def.Validate)
		stripNewlines := slices.Contains(def.Sanitize, "strip_newlines")
		stripNulls := slices.Contains(def.Sanitize, "strip_nulls")
		b.WriteString(fmt.Sprintf("\t%s: {MaxLen: %d, StripNewlines: %v, StripNulls: %v},\n",
			camel, maxLen, stripNewlines, stripNulls))
	}
	b.WriteString("}\n\n")

	// Sanitizers
	b.WriteString("// --- Sanitizers ---\n\n")
	b.WriteString("var SanitizerStripNewlines = strings.NewReplacer(\"\\n\", \"\", \"\\r\", \"\")\n")
	b.WriteString("var SanitizerStripNulls = strings.NewReplacer(\"\\x00\", \"\")\n\n")

	// Prompt roles
	for _, name := range promptRoleOrder {
		def := rules.PromptRoles[name]
		camel := snakeToCamel(name)
		b.WriteString(fmt.Sprintf("// --- Prompt role: %s ---\n\n", name))

		if def.Embedding != nil && def.Embedding.Delimiter != nil {
			b.WriteString(fmt.Sprintf("var %sDelimiterOpen = %q\n", camel, def.Embedding.Delimiter.Open))
			b.WriteString(fmt.Sprintf("var %sDelimiterClose = %q\n\n", camel, def.Embedding.Delimiter.Close))
		}

		if len(def.SystemInstructions) > 0 {
			b.WriteString(fmt.Sprintf("var %sSystemInstructions = ", camel))
			for i, instr := range def.SystemInstructions {
				if i == 0 {
					b.WriteString(fmt.Sprintf("%q", instr))
				} else {
					b.WriteString(fmt.Sprintf(" +\n\t%q", "\n"+instr))
				}
			}
			b.WriteString("\n\n")
		}

		if name == "AI_OUTPUT" {
			// Extract max_length map and rejection patterns
			b.WriteString("var AIOutputLimits = map[string]int{\n")
			for _, v := range def.Validate {
				var m map[string]json.RawMessage
				if err := json.Unmarshal(v, &m); err != nil {
					continue
				}
				if raw, ok := m["max_length"]; ok {
					var limits map[string]int
					if err := json.Unmarshal(raw, &limits); err == nil {
						// Sort keys for deterministic output
						keys := make([]string, 0, len(limits))
						for k := range limits {
							keys = append(keys, k)
						}
						slices.Sort(keys)
						for _, k := range keys {
							b.WriteString(fmt.Sprintf("\t%q: %d,\n", k, limits[k]))
						}
					}
				}
			}
			b.WriteString("}\n\n")
		}
	}

	// Exfiltration patterns
	b.WriteString("// --- Exfiltration patterns ---\n\n")
	b.WriteString("var ExfiltrationPatterns = map[string]string{\n")
	patternNames := make([]string, 0, len(rules.RejectionPatterns))
	for name := range rules.RejectionPatterns {
		patternNames = append(patternNames, name)
	}
	slices.Sort(patternNames)
	for _, name := range patternNames {
		def := rules.RejectionPatterns[name]
		b.WriteString(fmt.Sprintf("\t%q: `%s`,\n", name, def.Regex))
	}
	b.WriteString("}\n")

	// Format the Go source
	formatted, err := format.Source([]byte(b.String()))
	if err != nil {
		return fmt.Errorf("formatting generated Go: %w", err)
	}

	return os.WriteFile("validate/generated.go", formatted, 0644)
}

func extractMaxLength(validate []json.RawMessage) int {
	for _, v := range validate {
		var m map[string]int
		if err := json.Unmarshal(v, &m); err != nil {
			continue
		}
		if maxLen, ok := m["max_length"]; ok {
			return maxLen
		}
	}
	return 0
}
