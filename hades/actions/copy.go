package actions

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/SoftKiwiGames/hades/hades/schema"
	"github.com/SoftKiwiGames/hades/hades/types"
)

type CopyAction struct {
	Src      string
	Dst      string
	Artifact string
	Mode     uint32
}

func NewCopyAction(action *schema.ActionCopy) Action {
	mode := action.Mode
	if mode == 0 {
		mode = 0644 // Default mode
	}
	return &CopyAction{
		Src:      action.Src,
		Dst:      action.Dst,
		Artifact: action.Artifact,
		Mode:     mode,
	}
}

func (a *CopyAction) Execute(ctx context.Context, runtime *types.Runtime) error {
	// Create SSH session
	sess, err := runtime.SSHClient.Connect(ctx, runtime.Host)
	if err != nil {
		return fmt.Errorf("failed to connect to host: %w", err)
	}
	defer sess.Close()

	// Determine source
	var reader io.ReadCloser
	var srcDesc string

	if a.Artifact != "" {
		// Get artifact from manager
		art, err := runtime.ArtifactMgr.Get(a.Artifact)
		if err != nil {
			return fmt.Errorf("failed to get artifact %s: %w", a.Artifact, err)
		}
		defer art.Close()
		reader = art
		srcDesc = fmt.Sprintf("artifact:%s", a.Artifact)
	} else if a.Src != "" {
		// Read from local file
		f, err := os.Open(a.Src)
		if err != nil {
			return fmt.Errorf("failed to open source file %s: %w", a.Src, err)
		}
		defer f.Close()
		reader = f
		srcDesc = a.Src
	} else {
		return fmt.Errorf("either src or artifact must be specified")
	}

	// Copy file to remote host atomically
	if err := sess.CopyFile(ctx, reader, a.Dst, a.Mode); err != nil {
		return fmt.Errorf("failed to copy %s to %s: %w", srcDesc, a.Dst, err)
	}

	return nil
}

func (a *CopyAction) DryRun(ctx context.Context, runtime *types.Runtime) string {
	if a.Artifact != "" {
		return fmt.Sprintf("copy: artifact=%s to=%s (mode: %o)", a.Artifact, a.Dst, a.Mode)
	}
	return fmt.Sprintf("copy: %s to %s (mode: %o)", a.Src, a.Dst, a.Mode)
}
