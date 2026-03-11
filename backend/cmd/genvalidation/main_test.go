package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerator(t *testing.T) {
	// Create a temp dir to simulate the project structure
	tmp := t.TempDir()

	// Copy validation-rules.json one level up (simulating ../validation-rules.json)
	rulesJSON, err := os.ReadFile("../../../validation-rules.json")
	if err != nil {
		t.Fatalf("reading validation-rules.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "validation-rules.json"), rulesJSON, 0644); err != nil {
		t.Fatal(err)
	}

	// Create output dirs
	backendDir := filepath.Join(tmp, "backend")
	if err := os.MkdirAll(filepath.Join(backendDir, "graph"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(backendDir, "validate"), 0755); err != nil {
		t.Fatal(err)
	}

	// Build and run the generator
	binary := filepath.Join(tmp, "genvalidation")
	build := exec.Command("go", "build", "-o", binary, ".")
	build.Dir = "."
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building generator: %v\n%s", err, out)
	}

	cmd := exec.Command(binary)
	cmd.Dir = backendDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("running generator: %v\n%s", err, out)
	}

	// Verify GraphQL file
	gql, err := os.ReadFile(filepath.Join(backendDir, "graph", "validation.graphqls"))
	if err != nil {
		t.Fatalf("reading generated GraphQL: %v", err)
	}
	gqlStr := string(gql)
	if !strings.Contains(gqlStr, "AUTO-GENERATED") {
		t.Error("GraphQL file missing AUTO-GENERATED header")
	}
	if !strings.Contains(gqlStr, "enum TextFieldRule") {
		t.Error("GraphQL file missing TextFieldRule enum")
	}
	if !strings.Contains(gqlStr, "SINGLE_LINE") {
		t.Error("GraphQL file missing SINGLE_LINE")
	}
	if !strings.Contains(gqlStr, "enum PromptRole") {
		t.Error("GraphQL file missing PromptRole enum")
	}
	if !strings.Contains(gqlStr, "directive @validate") {
		t.Error("GraphQL file missing @validate directive")
	}
	if !strings.Contains(gqlStr, "directive @prompt") {
		t.Error("GraphQL file missing @prompt directive")
	}

	// Verify Go file
	goFile, err := os.ReadFile(filepath.Join(backendDir, "validate", "generated.go"))
	if err != nil {
		t.Fatalf("reading generated Go: %v", err)
	}
	goStr := string(goFile)
	if !strings.Contains(goStr, "AUTO-GENERATED") {
		t.Error("Go file missing AUTO-GENERATED header")
	}
	if !strings.Contains(goStr, "SingleLine TextFieldRule = iota") {
		t.Error("Go file missing SingleLine enum")
	}
	if !strings.Contains(goStr, "SanitizerStripNewlines") {
		t.Error("Go file missing SanitizerStripNewlines")
	}
	if !strings.Contains(goStr, "ExfiltrationPatterns") {
		t.Error("Go file missing ExfiltrationPatterns")
	}
	if !strings.Contains(goStr, "AIOutputLimits") {
		t.Error("Go file missing AIOutputLimits")
	}
}
