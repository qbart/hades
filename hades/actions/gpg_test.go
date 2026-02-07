package actions

import (
	"context"
	"testing"

	"github.com/SoftKiwiGames/hades/hades/schema"
	"github.com/SoftKiwiGames/hades/hades/types"
)

func TestGpgAction_DryRun(t *testing.T) {
	action := &GpgAction{
		Src:  "https://example.com/gpg.key",
		Path: "/usr/share/keyrings/example.gpg",
		Mode: 0644,
	}

	runtime := &types.Runtime{
		Env: map[string]string{},
	}

	result := action.DryRun(context.Background(), runtime)
	expected := "gpg: download https://example.com/gpg.key to /usr/share/keyrings/example.gpg (mode: 644)"

	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestGpgAction_DryRun_WithEnvVars(t *testing.T) {
	action := &GpgAction{
		Src:  "https://${REPO_HOST}/gpg.key",
		Path: "/usr/share/keyrings/${KEY_NAME}.gpg",
		Mode: 0644,
	}

	runtime := &types.Runtime{
		Env: map[string]string{
			"REPO_HOST": "dl.example.com",
			"KEY_NAME":  "example",
		},
	}

	result := action.DryRun(context.Background(), runtime)
	expected := "gpg: download https://dl.example.com/gpg.key to /usr/share/keyrings/example.gpg (mode: 644)"

	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestNewGpgAction(t *testing.T) {
	schema := &schema.ActionGpg{
		Src:  "https://example.com/gpg.key",
		Path: "/usr/share/keyrings/example.gpg",
		Mode: 0600,
	}

	action := NewGpgAction(schema)

	gpgAction, ok := action.(*GpgAction)
	if !ok {
		t.Fatal("Expected *GpgAction type")
	}

	if gpgAction.Src != schema.Src {
		t.Errorf("Expected Src %q, got %q", schema.Src, gpgAction.Src)
	}

	if gpgAction.Path != schema.Path {
		t.Errorf("Expected Path %q, got %q", schema.Path, gpgAction.Path)
	}

	if gpgAction.Mode != 0600 {
		t.Errorf("Expected Mode 0600, got %o", gpgAction.Mode)
	}
}

func TestNewGpgAction_DefaultMode(t *testing.T) {
	schema := &schema.ActionGpg{
		Src:  "https://example.com/gpg.key",
		Path: "/usr/share/keyrings/example.gpg",
		// Mode not specified, should default to 0644
	}

	action := NewGpgAction(schema)

	gpgAction, ok := action.(*GpgAction)
	if !ok {
		t.Fatal("Expected *GpgAction type")
	}

	if gpgAction.Mode != 0644 {
		t.Errorf("Expected default mode 0644, got %o", gpgAction.Mode)
	}
}

func TestGpgAction_DryRun_WithDearmor(t *testing.T) {
	action := &GpgAction{
		Src:     "https://example.com/gpg.asc",
		Path:    "/usr/share/keyrings/example.gpg",
		Mode:    0644,
		Dearmor: true,
	}

	runtime := &types.Runtime{
		Env: map[string]string{},
	}

	result := action.DryRun(context.Background(), runtime)
	expected := "gpg: download https://example.com/gpg.asc, dearmor to /usr/share/keyrings/example.gpg (mode: 644)"

	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestNewGpgAction_WithDearmor(t *testing.T) {
	schema := &schema.ActionGpg{
		Src:     "https://example.com/gpg.asc",
		Path:    "/usr/share/keyrings/example.gpg",
		Mode:    0600,
		Dearmor: true,
	}

	action := NewGpgAction(schema)

	gpgAction, ok := action.(*GpgAction)
	if !ok {
		t.Fatal("Expected *GpgAction type")
	}

	if gpgAction.Src != schema.Src {
		t.Errorf("Expected Src %q, got %q", schema.Src, gpgAction.Src)
	}

	if gpgAction.Path != schema.Path {
		t.Errorf("Expected Path %q, got %q", schema.Path, gpgAction.Path)
	}

	if gpgAction.Mode != 0600 {
		t.Errorf("Expected Mode 0600, got %o", gpgAction.Mode)
	}

	if !gpgAction.Dearmor {
		t.Error("Expected Dearmor to be true")
	}
}
