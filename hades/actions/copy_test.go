package actions

import (
	"context"
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
	expected := "copy: /local/file.txt to /remote/file.txt (mode: 644)"

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
	expected := "copy: artifact=my-binary to=/usr/local/bin/app (mode: 755)"

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
