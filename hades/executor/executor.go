package executor

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/SoftKiwiGames/hades/hades/actions"
	"github.com/SoftKiwiGames/hades/hades/inventory"
	"github.com/SoftKiwiGames/hades/hades/schema"
	"github.com/SoftKiwiGames/hades/hades/ssh"
	"github.com/SoftKiwiGames/hades/hades/types"
)

type Executor interface {
	ExecutePlan(ctx context.Context, file *schema.File, plan *schema.Plan, planName string, inv inventory.Inventory, targets []string, env map[string]string) (*Result, error)
	DryRun(ctx context.Context, file *schema.File, plan *schema.Plan, planName string, inv inventory.Inventory, targets []string, env map[string]string) error
}

type Result struct {
	RunID      string
	StartTime  time.Time
	EndTime    time.Time
	Failed     bool
	FailedStep string
	FailedHost string
	Error      error
}

type executor struct {
	sshClient ssh.Client
	stdout    io.Writer
	stderr    io.Writer
}

func New(sshClient ssh.Client, stdout, stderr io.Writer) Executor {
	return &executor{
		sshClient: sshClient,
		stdout:    stdout,
		stderr:    stderr,
	}
}

func (e *executor) ExecutePlan(ctx context.Context, file *schema.File, plan *schema.Plan, planName string, inv inventory.Inventory, targets []string, env map[string]string) (*Result, error) {
	result := &Result{
		StartTime: time.Now(),
	}

	fmt.Fprintf(e.stdout, "Starting plan: %s\n", planName)
	fmt.Fprintf(e.stdout, "Run ID: %s\n\n", result.RunID)

	// Execute each step sequentially
	for i, step := range plan.Steps {
		fmt.Fprintf(e.stdout, "Step %d/%d: %s\n", i+1, len(plan.Steps), step.Name)
		fmt.Fprintf(e.stdout, "  Job: %s\n", step.Job)

		// Resolve hosts from targets
		var hosts []ssh.Host
		for _, targetName := range step.Targets {
			targetHosts, err := inv.ResolveTarget(targetName)
			if err != nil {
				result.Failed = true
				result.FailedStep = step.Name
				result.Error = fmt.Errorf("failed to resolve target %q: %w", targetName, err)
				return result, result.Error
			}
			hosts = append(hosts, targetHosts...)
		}

		// Apply limit if specified (canary)
		if step.Limit > 0 && step.Limit < len(hosts) {
			fmt.Fprintf(e.stdout, "  Limiting to %d hosts (canary)\n", step.Limit)
			hosts = hosts[:step.Limit]
		}

		fmt.Fprintf(e.stdout, "  Hosts: %d\n\n", len(hosts))

		// Load job
		job, err := e.loadJob(file, step.Job)
		if err != nil {
			result.Failed = true
			result.FailedStep = step.Name
			result.Error = err
			return result, result.Error
		}

		// Merge env: CLI env + step env
		mergedEnv := make(map[string]string)
		for k, v := range env {
			mergedEnv[k] = v
		}
		for k, v := range step.Env {
			mergedEnv[k] = v
		}

		// Execute job on each host (no parallelism yet - Phase 5)
		for _, host := range hosts {
			fmt.Fprintf(e.stdout, "[%s] Executing job %s\n", host.Name, step.Job)

			if err := e.executeJob(ctx, job, planName, step.Targets[0], host, mergedEnv); err != nil {
				result.Failed = true
				result.FailedStep = step.Name
				result.FailedHost = host.Name
				result.Error = fmt.Errorf("job failed on host %s: %w", host.Name, err)
				result.EndTime = time.Now()
				return result, result.Error
			}

			fmt.Fprintf(e.stdout, "[%s] ✓ Job completed\n\n", host.Name)
		}

		fmt.Fprintf(e.stdout, "✓ Step completed: %s\n\n", step.Name)
	}

	result.EndTime = time.Now()
	fmt.Fprintf(e.stdout, "✓ Plan completed successfully\n")
	fmt.Fprintf(e.stdout, "Duration: %s\n", result.EndTime.Sub(result.StartTime))

	return result, nil
}

func (e *executor) executeJob(ctx context.Context, job *schema.Job, plan string, target string, host ssh.Host, env map[string]string) error {
	// Create runtime context
	runtime := types.NewRuntime(e.sshClient, plan, target, host, env)

	// Execute each action sequentially
	for i, actionSchema := range job.Actions {
		action, err := e.createAction(&actionSchema)
		if err != nil {
			return fmt.Errorf("action %d: %w", i, err)
		}

		if err := action.Execute(ctx, runtime); err != nil {
			return fmt.Errorf("action %d failed: %w", i, err)
		}
	}

	return nil
}

func (e *executor) createAction(actionSchema *schema.Action) (actions.Action, error) {
	if actionSchema.Run != nil {
		return actions.NewRunAction(actionSchema.Run, e.stdout, e.stderr), nil
	}
	if actionSchema.Copy != nil {
		return actions.NewCopyAction(actionSchema.Copy), nil
	}
	if actionSchema.Template != nil {
		return nil, fmt.Errorf("template action not yet implemented (Phase 3)")
	}
	if actionSchema.Mkdir != nil {
		return nil, fmt.Errorf("mkdir action not yet implemented (Phase 3)")
	}
	if actionSchema.Push != nil {
		return nil, fmt.Errorf("push action not yet implemented (Phase 4)")
	}
	if actionSchema.Pull != nil {
		return nil, fmt.Errorf("pull action not yet implemented (Phase 4)")
	}
	if actionSchema.Wait != nil {
		return nil, fmt.Errorf("wait action not yet implemented (Phase 3)")
	}

	return nil, fmt.Errorf("no action type specified")
}

func (e *executor) loadJob(file *schema.File, name string) (*schema.Job, error) {
	job, ok := file.Jobs[name]
	if !ok {
		return nil, fmt.Errorf("job %q not found", name)
	}
	return &job, nil
}

func (e *executor) DryRun(ctx context.Context, file *schema.File, plan *schema.Plan, planName string, inv inventory.Inventory, targets []string, env map[string]string) error {
	fmt.Fprintf(e.stdout, "Dry-run: %s\n", planName)
	fmt.Fprintf(e.stdout, "This will execute the following:\n\n")

	// Iterate steps
	for i, step := range plan.Steps {
		fmt.Fprintf(e.stdout, "Step %d: %s\n", i+1, step.Name)
		fmt.Fprintf(e.stdout, "  Job: %s\n", step.Job)

		// Resolve hosts
		var hosts []ssh.Host
		for _, targetName := range step.Targets {
			targetHosts, err := inv.ResolveTarget(targetName)
			if err != nil {
				return fmt.Errorf("failed to resolve target %q: %w", targetName, err)
			}
			hosts = append(hosts, targetHosts...)
		}

		if step.Limit > 0 && step.Limit < len(hosts) {
			hosts = hosts[:step.Limit]
		}

		// Load job
		job, err := e.loadJob(file, step.Job)
		if err != nil {
			return err
		}

		// Merge env
		mergedEnv := make(map[string]string)
		for k, v := range env {
			mergedEnv[k] = v
		}
		for k, v := range step.Env {
			mergedEnv[k] = v
		}

		// Show actions for each host
		for _, host := range hosts {
			runtime := types.NewRuntime(e.sshClient, planName, step.Targets[0], host, mergedEnv)

			fmt.Fprintf(e.stdout, "\n  [%s]\n", host.Name)
			for _, actionSchema := range job.Actions {
				action, err := e.createAction(&actionSchema)
				if err != nil {
					return err
				}
				fmt.Fprintf(e.stdout, "    - %s\n", action.DryRun(ctx, runtime))
			}
		}

		fmt.Fprintf(e.stdout, "\n")
	}

	return nil
}
