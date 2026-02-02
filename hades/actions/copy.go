package actions

import (
	"context"
	"fmt"
	"os"

	"github.com/SoftKiwiGames/hades/hades/schema"
	"github.com/SoftKiwiGames/hades/hades/types"
)

type CopyAction struct {
	Src      string
	Dst      string
	Artifact string
}

func NewCopyAction(action *schema.ActionCopy) Action {
	return &CopyAction{
		Src:      action.Src,
		Dst:      action.Dst,
		Artifact: action.Artifact,
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
	var file *os.File
	var srcDesc string

	if a.Artifact != "" {
		// TODO: In Phase 3, implement artifact retrieval
		return fmt.Errorf("artifact copy not yet implemented (will be in Phase 3)")
	} else if a.Src != "" {
		// Read from local file
		f, err := os.Open(a.Src)
		if err != nil {
			return fmt.Errorf("failed to open source file %s: %w", a.Src, err)
		}
		defer f.Close()
		file = f
		srcDesc = a.Src
	} else {
		return fmt.Errorf("either src or artifact must be specified")
	}

	// Copy file to remote host atomically
	// Default mode 0644 for now
	if err := sess.CopyFile(ctx, file, a.Dst, 0644); err != nil {
		return fmt.Errorf("failed to copy %s to %s: %w", srcDesc, a.Dst, err)
	}

	return nil
}

func (a *CopyAction) DryRun(ctx context.Context, runtime *types.Runtime) string {
	if a.Artifact != "" {
		return fmt.Sprintf("copy: artifact=%s to=%s", a.Artifact, a.Dst)
	}
	return fmt.Sprintf("copy: %s to %s", a.Src, a.Dst)
}
