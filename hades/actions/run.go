package actions

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/SoftKiwiGames/hades/hades/schema"
	"github.com/SoftKiwiGames/hades/hades/types"
)

type RunAction struct {
	Command string
	Output  io.Writer
	ErrOut  io.Writer
}

func NewRunAction(action *schema.ActionRun, stdout, stderr io.Writer) Action {
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}

	return &RunAction{
		Command: string(*action),
		Output:  stdout,
		ErrOut:  stderr,
	}
}

func (a *RunAction) Execute(ctx context.Context, runtime *types.Runtime) error {
	// Create SSH session
	sess, err := runtime.SSHClient.Connect(ctx, runtime.Host)
	if err != nil {
		return fmt.Errorf("failed to connect to host: %w", err)
	}
	defer sess.Close()

	// Build command with environment variables
	// For now, just run the command directly
	// TODO: In Phase 6, properly inject environment variables
	cmd := a.Command

	// Execute command
	if err := sess.Run(ctx, cmd, a.Output, a.ErrOut); err != nil {
		return fmt.Errorf("command execution failed: %w", err)
	}

	return nil
}

func (a *RunAction) DryRun(ctx context.Context, runtime *types.Runtime) string {
	return fmt.Sprintf("run: %s", a.Command)
}
