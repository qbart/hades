package executor

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/SoftKiwiGames/hades/hades/actions"
	"github.com/SoftKiwiGames/hades/hades/artifacts"
	"github.com/SoftKiwiGames/hades/hades/inventory"
	"github.com/SoftKiwiGames/hades/hades/loader"
	"github.com/SoftKiwiGames/hades/hades/logger"
	"github.com/SoftKiwiGames/hades/hades/registry"
	"github.com/SoftKiwiGames/hades/hades/rollout"
	"github.com/SoftKiwiGames/hades/hades/schema"
	"github.com/SoftKiwiGames/hades/hades/ssh"
	"github.com/SoftKiwiGames/hades/hades/types"
	"github.com/SoftKiwiGames/hades/hades/ui"
	"github.com/wzshiming/ctc"
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
	ui        *ui.Output
}

func New(sshClient ssh.Client, stdout, stderr io.Writer) Executor {
	return &executor{
		sshClient: sshClient,
		stdout:    stdout,
		stderr:    stderr,
		ui:        ui.NewOutput(stdout, stderr),
	}
}

func (e *executor) ExecutePlan(ctx context.Context, file *schema.File, plan *schema.Plan, planName string, inv inventory.Inventory, targets []string, env map[string]string) (*Result, error) {
	result := &Result{
		StartTime: time.Now(),
	}

	// Create artifact manager for this run
	artifactMgr := artifacts.NewManager()
	defer artifactMgr.Clear()

	// Create registry manager
	registryMgr, err := registry.NewManager(file.Registries)
	if err != nil {
		result.Failed = true
		result.Error = fmt.Errorf("failed to initialize registries: %w", err)
		return result, result.Error
	}

	// Generate unique run ID
	result.RunID = "hades-" + time.Now().Format("20060102-150405")

	e.ui.PlanStarted(planName, result.RunID)

	// Execute each step sequentially
	for i, step := range plan.Steps {
		e.ui.StepProgress(i+1, len(plan.Steps), step.Name)
		e.ui.Info("  Job: %s", step.Job)
		targetsStr := strings.Join(step.Targets, ", ")
		e.ui.Info("  Targets: %s", targetsStr)

		// Load job once for this step
		job, err := e.loadJob(file, step.Job)
		if err != nil {
			result.Failed = true
			result.FailedStep = step.Name
			result.Error = err
			return result, result.Error
		}

		// Merge env with priority: CLI > step > job defaults
		stepEnv := make(map[string]string)
		for k, v := range step.Env {
			stepEnv[k] = v
		}
		for k, v := range env {
			stepEnv[k] = v // CLI overrides step
		}

		// Merge with job defaults
		mergedEnv := loader.MergeEnv(job, stepEnv)

		// Load artifacts for this job if any are defined
		if err := e.loadArtifacts(job, artifactMgr); err != nil {
			result.Failed = true
			result.FailedStep = step.Name
			result.Error = fmt.Errorf("failed to load artifacts: %w", err)
			return result, result.Error
		}

		// Execute each target separately
		totalHosts := 0
		for _, targetName := range step.Targets {
			// Resolve hosts for this target
			targetHosts, err := inv.ResolveTarget(targetName)
			if err != nil {
				result.Failed = true
				result.FailedStep = step.Name
				result.Error = fmt.Errorf("failed to resolve target %q: %w", targetName, err)
				return result, result.Error
			}

			// Apply limit if specified (canary)
			if step.Limit > 0 && step.Limit < len(targetHosts) {
				targetHosts = targetHosts[:step.Limit]
			}

			totalHosts += len(targetHosts)

			// Target started
			fmt.Fprintf(e.stdout, "\n%s□%s Target %q: started (%d hosts)\n", ctc.ForegroundYellow, ctc.Reset, targetName, len(targetHosts))

			// Parse rollout strategy for this target
			strategy, err := rollout.ParseStrategy(step.Parallelism, len(targetHosts))
			if err != nil {
				result.Failed = true
				result.FailedStep = step.Name
				result.Error = fmt.Errorf("invalid parallelism: %w", err)
				return result, result.Error
			}
			strategy.Limit = step.Limit

			// Create batches based on strategy
			batches := strategy.CreateBatches(targetHosts)

			// Execute batches sequentially, hosts within batch in parallel
			for batchIdx, batch := range batches {
				if len(batches) > 1 {
					fmt.Fprintf(e.stdout, "  Batch %d/%d (%d hosts)\n", batchIdx+1, len(batches), len(batch))
				}

				// Execute batch in parallel
				if err := e.executeBatch(ctx, job, step.Job, result.RunID, planName, targetName, batch, mergedEnv, artifactMgr, registryMgr); err != nil {
					// Target failed
					fmt.Fprintf(e.stderr, "%s■%s Target %q: failed\n", ctc.ForegroundRed, ctc.Reset, targetName)
					result.Failed = true
					result.FailedStep = step.Name
					result.Error = err
					result.EndTime = time.Now()
					return result, result.Error
				}

				if len(batches) > 1 {
					fmt.Fprintf(e.stdout, "  ✓ Batch %d/%d completed\n", batchIdx+1, len(batches))
				}
			}

			// Target completed
			fmt.Fprintf(e.stdout, "%s■%s Target %q: completed (%d hosts)\n", ctc.ForegroundGreen, ctc.Reset, targetName, len(targetHosts))
		}

		// Step completion summary
		fmt.Fprintf(e.stdout, "\n%s✓%s Step completed: %s\n", ctc.ForegroundGreen, ctc.Reset, step.Name)
		fmt.Fprintf(e.stdout, "  Targets: %s (%d hosts)\n\n", targetsStr, totalHosts)
	}

	result.EndTime = time.Now()
	e.ui.PlanCompleted(result.EndTime.Sub(result.StartTime))

	return result, nil
}

func (e *executor) executeBatch(ctx context.Context, job *schema.Job, jobName string, runID string, plan string, target string, hosts []ssh.Host, env map[string]string, artifactMgr artifacts.Manager, registryMgr registry.Manager) error {
	// Use channels to coordinate parallel execution
	type result struct {
		host ssh.Host
		err  error
	}

	resultChan := make(chan result, len(hosts))
	var wg sync.WaitGroup

	// Execute on each host in parallel
	for _, host := range hosts {
		wg.Add(1)
		go func(h ssh.Host) {
			defer wg.Done()

			err := e.executeJob(ctx, job, jobName, runID, plan, target, h, env, artifactMgr, registryMgr)

			if err != nil {
				fmt.Fprintf(e.stderr, "[%s] %s◆%s Job %q: failed - %v\n", h.Name, ctc.ForegroundRed, ctc.Reset, jobName, err)
			} else {
				fmt.Fprintf(e.stdout, "[%s] %s◆%s Job %q: completed\n", h.Name, ctc.ForegroundGreen, ctc.Reset, jobName)
			}

			resultChan <- result{host: h, err: err}
		}(host)
	}

	// Wait for all goroutines to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results (abort on first failure)
	for res := range resultChan {
		if res.err != nil {
			// Abort on first failure
			return fmt.Errorf("job failed on host %s: %w", res.host.Name, res.err)
		}
	}

	return nil
}

func (e *executor) executeJob(ctx context.Context, job *schema.Job, jobName string, runID string, plan string, target string, host ssh.Host, env map[string]string, artifactMgr artifacts.Manager, registryMgr registry.Manager) error {
	// Create logger for this host
	hostLogger, err := logger.New(runID, plan, host.Name, e.stdout, e.stderr)
	if err != nil {
		return fmt.Errorf("failed to initialize logger for host %s: %w", host.Name, err)
	}
	defer hostLogger.Close()

	// Create runtime context with logger writers and console writers
	runtime := types.NewRuntime(e.sshClient, artifactMgr, registryMgr, plan, target, host, env, hostLogger.Stdout(), hostLogger.Stderr(), e.stdout, e.stderr)

	// Evaluate guard condition first (before showing job starting)
	if job.Guard != nil {
		result, err := actions.EvaluateGuard(ctx, job.Guard, runtime)
		if err != nil {
			return fmt.Errorf("guard evaluation failed: %w", err)
		}

		if !result.Pass {
			// Console: Job skipped
			fmt.Fprintf(e.stdout, "[%s] %s◇%s Job %q: skipped (guard failed)\n", host.Name, ctc.ForegroundBlue, ctc.Reset, jobName)
			return nil // Skip job, but not an error
		}
	}

	// Console: Job starting (only if guard passed or no guard)
	fmt.Fprintf(e.stdout, "[%s] %s◇%s Job %q: starting\n", host.Name, ctc.ForegroundYellow, ctc.Reset, jobName)

	// Execute each action sequentially
	for i, actionSchema := range job.Actions {
		// Get action type for delimiter
		actionType := getActionType(&actionSchema)

		// Format action description for console
		actionDesc := fmt.Sprintf("[%d] %s", i, actionType)
		if actionSchema.Name != "" {
			actionDesc = fmt.Sprintf("[%d] %s (%s)", i, actionType, actionSchema.Name)
		}

		// Set action description in runtime for use by actions
		runtime.ActionDesc = actionDesc

		// Write delimiter to log (with optional name)
		if err := hostLogger.WriteJobDelimiter(jobName, actionType, actionSchema.Name, i); err != nil {
			return fmt.Errorf("failed to write log delimiter: %w", err)
		}

		// Console: Action starting
		fmt.Fprintf(e.stdout, "[%s] %s◌%s Action %s: in progress\n", host.Name, ctc.ForegroundYellow, ctc.Reset, actionDesc)

		action, err := e.createAction(&actionSchema, hostLogger)
		if err != nil {
			return fmt.Errorf("action %d: %w", i, err)
		}

		if err := action.Execute(ctx, runtime); err != nil {
			// Console: Action failed
			fmt.Fprintf(e.stderr, "[%s] %s●%s Action %s: failed - %v\n", host.Name, ctc.ForegroundRed, ctc.Reset, actionDesc, err)
			return fmt.Errorf("action %d failed: %w", i, err)
		}

		// Console: Action completed
		fmt.Fprintf(e.stdout, "[%s] %s●%s Action %s: completed\n", host.Name, ctc.ForegroundGreen, ctc.Reset, actionDesc)
	}

	return nil
}

func (e *executor) createAction(actionSchema *schema.Action, planLogger *logger.Logger) (actions.Action, error) {
	if actionSchema.Run != nil {
		return actions.NewRunAction(actionSchema.Run), nil
	}
	if actionSchema.Copy != nil {
		return actions.NewCopyAction(actionSchema.Copy), nil
	}
	if actionSchema.Template != nil {
		return actions.NewTemplateAction(actionSchema.Template), nil
	}
	if actionSchema.Mkdir != nil {
		return actions.NewMkdirAction(actionSchema.Mkdir), nil
	}
	if actionSchema.Push != nil {
		return actions.NewPushAction(actionSchema.Push), nil
	}
	if actionSchema.Pull != nil {
		return actions.NewPullAction(actionSchema.Pull), nil
	}
	if actionSchema.Wait != nil {
		return actions.NewWaitAction(actionSchema.Wait), nil
	}
	if actionSchema.Gpg != nil {
		return actions.NewGpgAction(actionSchema.Gpg), nil
	}

	return nil, fmt.Errorf("no action type specified")
}

// getActionType extracts the action type from the action schema
func getActionType(actionSchema *schema.Action) string {
	if actionSchema.Run != nil {
		return "run"
	}
	if actionSchema.Copy != nil {
		return "copy"
	}
	if actionSchema.Template != nil {
		return "template"
	}
	if actionSchema.Mkdir != nil {
		return "mkdir"
	}
	if actionSchema.Push != nil {
		return "push"
	}
	if actionSchema.Pull != nil {
		return "pull"
	}
	if actionSchema.Wait != nil {
		return "wait"
	}
	if actionSchema.Gpg != nil {
		return "gpg"
	}
	return "unknown"
}

func (e *executor) loadArtifacts(job *schema.Job, artifactMgr artifacts.Manager) error {
	// Load artifacts defined in the job
	for name, artifact := range job.Artifacts {
		file, err := os.Open(artifact.Path)
		if err != nil {
			return fmt.Errorf("failed to open artifact %s at %s: %w", name, artifact.Path, err)
		}
		defer file.Close()

		if err := artifactMgr.Store(name, file); err != nil {
			return fmt.Errorf("failed to store artifact %s: %w", name, err)
		}

		fmt.Fprintf(e.stdout, "  Loaded artifact: %s from %s\n", name, artifact.Path)
	}

	return nil
}

func (e *executor) loadJob(file *schema.File, name string) (*schema.Job, error) {
	job, ok := file.Jobs[name]
	if !ok {
		return nil, fmt.Errorf("job %q not found", name)
	}
	return &job, nil
}

func (e *executor) DryRun(ctx context.Context, file *schema.File, plan *schema.Plan, planName string, inv inventory.Inventory, targets []string, env map[string]string) error {
	// Create artifact manager for dry-run (won't actually load artifacts)
	artifactMgr := artifacts.NewManager()

	// Create registry manager for dry-run
	registryMgr, err := registry.NewManager(file.Registries)
	if err != nil {
		return fmt.Errorf("failed to initialize registries: %w", err)
	}

	e.ui.DryRunHeader(planName)

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

		// Merge env with priority: CLI > step > job defaults
		stepEnv := make(map[string]string)
		for k, v := range step.Env {
			stepEnv[k] = v
		}
		for k, v := range env {
			stepEnv[k] = v // CLI overrides step
		}

		// Merge with job defaults
		mergedEnv := loader.MergeEnv(job, stepEnv)

		// Show actions for each host
		for _, host := range hosts {
			runtime := types.NewRuntime(e.sshClient, artifactMgr, registryMgr, planName, step.Targets[0], host, mergedEnv, e.stdout, e.stderr, e.stdout, e.stderr)

			fmt.Fprintf(e.stdout, "\n  [%s]\n", host.Name)
			for _, actionSchema := range job.Actions {
				action, err := e.createAction(&actionSchema, nil)
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
