package actions

import (
	"bytes"
	"context"
	"fmt"

	"github.com/SoftKiwiGames/hades/hades/schema"
	"github.com/SoftKiwiGames/hades/hades/types"
)

// GuardResult represents the outcome of a guard evaluation
type GuardResult struct {
	Pass   bool   // true = continue, false = skip
	Output string // stdout from command (for debugging)
}

// EvaluateGuard runs a guard test and returns whether to continue
func EvaluateGuard(ctx context.Context, guard *schema.Guard, runtime *types.Runtime) (*GuardResult, error) {
	if guard == nil {
		return &GuardResult{Pass: true}, nil // No guard = always pass
	}

	// Expand environment variables in command
	cmd, err := expandEnv(guard.If, runtime.Env)
	if err != nil {
		return nil, fmt.Errorf("failed to expand guard command: %w", err)
	}

	// Create SSH session
	sess, err := runtime.SSHClient.Connect(ctx, runtime.Host)
	if err != nil {
		return nil, fmt.Errorf("failed to connect for guard check: %w", err)
	}
	defer sess.Close()

	// Run guard command
	var stdout, stderr bytes.Buffer
	err = sess.Run(ctx, cmd, &stdout, &stderr)

	result := &GuardResult{
		Pass:   err == nil, // Exit 0 = pass, non-zero = skip
		Output: stdout.String(),
	}

	return result, nil
}

// FormatGuardCondition returns human-readable guard description
func FormatGuardCondition(guard *schema.Guard, env map[string]string) string {
	if guard == nil {
		return ""
	}

	cmd, _ := expandEnv(guard.If, env)
	return fmt.Sprintf("guard: %s", cmd)
}
