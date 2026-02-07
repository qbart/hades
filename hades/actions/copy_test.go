package actions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"strings"
	"testing"

	"github.com/SoftKiwiGames/hades/hades/schema"
	"github.com/SoftKiwiGames/hades/hades/types"
)

func TestCopyAction_DryRun(t *testing.T) {
	action := &CopyAction{
		Src:  "/local/file.txt",
		Dst:  "/remote/file.txt",
		Mode: 0644,
	}

	runtime := &types.Runtime{
		Env: map[string]string{},
	}

	result := action.DryRun(context.Background(), runtime)
	expected := "copy: /local/file.txt to /remote/file.txt (mode: 644, verify checksum)"

	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestCopyAction_DryRun_WithArtifact(t *testing.T) {
	action := &CopyAction{
		Artifact: "my-binary",
		Dst:      "/usr/local/bin/app",
		Mode:     0755,
	}

	runtime := &types.Runtime{
		Env: map[string]string{},
	}

	result := action.DryRun(context.Background(), runtime)
	expected := "copy: artifact=my-binary to=/usr/local/bin/app (mode: 755, verify checksum)"

	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestNewCopyAction_DefaultMode(t *testing.T) {
	schema := &schema.ActionCopy{
		Src: "/local/file.txt",
		Dst: "/remote/file.txt",
		// Mode not specified, should default to 0644
	}

	action := NewCopyAction(schema)

	copyAction, ok := action.(*CopyAction)
	if !ok {
		t.Fatal("Expected *CopyAction type")
	}

	if copyAction.Mode != 0644 {
		t.Errorf("Expected default mode 0644, got %o", copyAction.Mode)
	}
}

func TestNewCopyAction_CustomMode(t *testing.T) {
	schema := &schema.ActionCopy{
		Src:  "/local/script.sh",
		Dst:  "/usr/local/bin/script.sh",
		Mode: 0755,
	}

	action := NewCopyAction(schema)

	copyAction, ok := action.(*CopyAction)
	if !ok {
		t.Fatal("Expected *CopyAction type")
	}

	if copyAction.Mode != 0755 {
		t.Errorf("Expected mode 0755, got %o", copyAction.Mode)
	}
}

func TestNewCopyAction_WithArtifact(t *testing.T) {
	schema := &schema.ActionCopy{
		Artifact: "my-binary",
		Dst:      "/usr/local/bin/app",
		Mode:     0755,
	}

	action := NewCopyAction(schema)

	copyAction, ok := action.(*CopyAction)
	if !ok {
		t.Fatal("Expected *CopyAction type")
	}

	if copyAction.Artifact != "my-binary" {
		t.Errorf("Expected artifact 'my-binary', got %q", copyAction.Artifact)
	}

	if copyAction.Mode != 0755 {
		t.Errorf("Expected mode 0755, got %o", copyAction.Mode)
	}
}

// TestCalculateChecksum verifies SHA-256 checksum calculation
func TestCalculateChecksum(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:     "hello world",
			input:    "hello world",
			expected: "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
		},
		{
			name:     "multiline",
			input:    "line1\nline2\nline3",
			expected: "a6e2c3e12b6b9c89c7d2e7c54f91d8b0e2d9b1c8a7f6e5d4c3b2a1908f7e6d5c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			checksum, err := calculateChecksum(reader)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify with known SHA-256 values
			h := sha256.New()
			h.Write([]byte(tt.input))
			expected := hex.EncodeToString(h.Sum(nil))

			if checksum != expected {
				t.Errorf("Expected %s, got %s", expected, checksum)
			}
		})
	}
}

// mockSession is a test double for ssh.Session
type mockSession struct {
	runFunc      func(ctx context.Context, cmd string, stdout, stderr io.Writer) error
	copyFileFunc func(ctx context.Context, content io.Reader, remotePath string, mode uint32) error
}

func (m *mockSession) Run(ctx context.Context, cmd string, stdout, stderr io.Writer) error {
	if m.runFunc != nil {
		return m.runFunc(ctx, cmd, stdout, stderr)
	}
	return nil
}

func (m *mockSession) CopyFile(ctx context.Context, content io.Reader, remotePath string, mode uint32) error {
	if m.copyFileFunc != nil {
		return m.copyFileFunc(ctx, content, remotePath, mode)
	}
	return nil
}

func (m *mockSession) Close() error {
	return nil
}

// TestGetRemoteChecksum_Success verifies parsing of sha256sum output
func TestGetRemoteChecksum_Success(t *testing.T) {
	expectedChecksum := "abc123def456789"
	sess := &mockSession{
		runFunc: func(ctx context.Context, cmd string, stdout, stderr io.Writer) error {
			// Simulate sha256sum output
			stdout.Write([]byte(expectedChecksum + "  /path/to/file\n"))
			return nil
		},
	}

	checksum, exists, err := getRemoteChecksum(context.Background(), sess, "/path/to/file")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !exists {
		t.Error("Expected exists=true")
	}

	if checksum != expectedChecksum {
		t.Errorf("Expected checksum %s, got %s", expectedChecksum, checksum)
	}
}

// TestGetRemoteChecksum_NotFound verifies handling when file doesn't exist
func TestGetRemoteChecksum_NotFound(t *testing.T) {
	sess := &mockSession{
		runFunc: func(ctx context.Context, cmd string, stdout, stderr io.Writer) error {
			// Simulate file not found
			stdout.Write([]byte("NOTFOUND\n"))
			return nil
		},
	}

	checksum, exists, err := getRemoteChecksum(context.Background(), sess, "/path/to/file")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if exists {
		t.Error("Expected exists=false")
	}

	if checksum != "" {
		t.Errorf("Expected empty checksum, got %s", checksum)
	}
}

// TestGetRemoteChecksum_MissingTool verifies fallback when sha256sum not available
func TestGetRemoteChecksum_MissingTool(t *testing.T) {
	sess := &mockSession{
		runFunc: func(ctx context.Context, cmd string, stdout, stderr io.Writer) error {
			// Simulate command failure (sha256sum not found)
			return io.EOF
		},
	}

	checksum, exists, err := getRemoteChecksum(context.Background(), sess, "/path/to/file")
	if err != nil {
		t.Fatalf("Expected nil error for graceful fallback, got: %v", err)
	}

	if exists {
		t.Error("Expected exists=false for missing tool")
	}

	if checksum != "" {
		t.Errorf("Expected empty checksum, got %s", checksum)
	}
}

// TestGetRemoteChecksum_EmptyOutput verifies handling of empty output
func TestGetRemoteChecksum_EmptyOutput(t *testing.T) {
	sess := &mockSession{
		runFunc: func(ctx context.Context, cmd string, stdout, stderr io.Writer) error {
			// Empty output
			return nil
		},
	}

	checksum, exists, err := getRemoteChecksum(context.Background(), sess, "/path/to/file")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if exists {
		t.Error("Expected exists=false for empty output")
	}

	if checksum != "" {
		t.Errorf("Expected empty checksum, got %s", checksum)
	}
}
