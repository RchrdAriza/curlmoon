package tui

import (
	"strings"
	"testing"
)

func TestHighlightJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "simple object",
			input:    `{"key":"value"}`,
			contains: "value",
		},
		{
			name:     "number values",
			input:    `{"count":42}`,
			contains: "42",
		},
		{
			name:     "boolean values",
			input:    `{"active":true,"done":false}`,
			contains: "true",
		},
		{
			name:     "null value",
			input:    `{"data":null}`,
			contains: "null",
		},
		{
			name:     "nested object",
			input:    `{"outer":{"inner":[1,2]}}`,
			contains: "inner",
		},
		{
			name:     "array",
			input:    `[1,2,3]`,
			contains: "2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := highlightJSON(tt.input)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("expected result to contain %q", tt.contains)
			}
		})
	}
}

func TestHighlightJSON_NonJSON(t *testing.T) {
	input := "Just a plain string"
	result := highlightJSON(input)
	if result != input {
		t.Errorf("expected non-JSON to pass through unchanged, got %q", result)
	}
}

func TestHighlightJSON_Empty(t *testing.T) {
	result := highlightJSON("")
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestHighlightJSON_PreservesStructure(t *testing.T) {
	input := `{"a":1,"b":2}`
	result := highlightJSON(input)
	if !strings.Contains(result, "{") || !strings.Contains(result, "}") {
		t.Error("expected braces to be preserved")
	}
	if !strings.Contains(result, ",") {
		t.Error("expected commas to be preserved")
	}
	if !strings.Contains(result, ":") {
		t.Error("expected colons to be preserved")
	}
}

func TestHighlightJSON_EdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty object", "{}"},
		{"empty array", "[]"},
		{"string with escape", `{"s":"hello \"world\""}`},
		{"number edge", `{"n":-0.5e10}`},
		{"mixed types", `{"s":"str","n":1,"b":true,"x":null}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := highlightJSON(tt.input)
			if result == "" {
				t.Error("expected non-empty result")
			}
		})
	}
}
