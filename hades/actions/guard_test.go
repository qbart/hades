package actions

import (
	"testing"

	"github.com/SoftKiwiGames/hades/hades/schema"
)

func TestFormatGuardCondition(t *testing.T) {
	tests := []struct {
		name     string
		guard    *schema.Guard
		env      map[string]string
		expected string
	}{
		{
			name:     "nil guard",
			guard:    nil,
			env:      map[string]string{},
			expected: "",
		},
		{
			name: "simple command",
			guard: &schema.Guard{
				If: "which caddy",
			},
			env:      map[string]string{},
			expected: "guard: which caddy",
		},
		{
			name: "command with env vars",
			guard: &schema.Guard{
				If: "test ${VERSION} = 1.0",
			},
			env: map[string]string{
				"VERSION": "1.0",
			},
			expected: "guard: test 1.0 = 1.0",
		},
		{
			name: "negated command",
			guard: &schema.Guard{
				If: "! which caddy",
			},
			env:      map[string]string{},
			expected: "guard: ! which caddy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatGuardCondition(tt.guard, tt.env)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGuardResult(t *testing.T) {
	// Test GuardResult struct creation
	result := &GuardResult{
		Pass:   true,
		Output: "test output",
	}

	if !result.Pass {
		t.Error("Expected Pass to be true")
	}

	if result.Output != "test output" {
		t.Errorf("Expected Output to be 'test output', got %q", result.Output)
	}
}
