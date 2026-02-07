package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExpandPath_Tilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "tilde with path",
			input:    "~/.ssh/id_rsa",
			expected: filepath.Join(home, ".ssh/id_rsa"),
		},
		{
			name:     "tilde alone",
			input:    "~",
			expected: home,
		},
		{
			name:     "tilde with subdirectory",
			input:    "~/Documents/keys/key.pem",
			expected: filepath.Join(home, "Documents/keys/key.pem"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExpandPath(tt.input)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestExpandPath_AbsolutePath(t *testing.T) {
	input := "/absolute/path/to/file"
	result, err := ExpandPath(input)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != input {
		t.Errorf("Expected %q, got %q", input, result)
	}
}

func TestExpandPath_RelativePath(t *testing.T) {
	input := "relative/path/file.txt"
	result, err := ExpandPath(input)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	// Should be converted to absolute path
	if !filepath.IsAbs(result) {
		t.Errorf("Expected absolute path, got %q", result)
	}
	// Should end with the relative path
	if !strings.HasSuffix(result, input) {
		t.Errorf("Expected result to end with %q, got %q", input, result)
	}
}

func TestExpandPath_Empty(t *testing.T) {
	result, err := ExpandPath("")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("Expected empty string, got %q", result)
	}
}
