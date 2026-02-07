package hades

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/SoftKiwiGames/hades/hades/executor"
	"github.com/SoftKiwiGames/hades/hades/inventory"
	"github.com/SoftKiwiGames/hades/hades/loader"
	"github.com/SoftKiwiGames/hades/hades/ssh"
	"github.com/hekmon/liveterm/v2"
	"github.com/spf13/cobra"
	"github.com/wzshiming/ctc"
)

type Hades struct {
	stdout *os.File
	stderr *os.File
	loader *loader.Loader
}

func New(stdout *os.File, stderr *os.File) *Hades {
	return &Hades{
		stdout: stdout,
		stderr: stderr,
		loader: loader.New(),
	}
}

func (h *Hades) Run() {
	rootCmd := &cobra.Command{
		Use:   "hades",
		Short: "Hades - Change execution tool for servers",
		Long:  "Hades gives provisioned machines a soul through explicit, predictable change execution.",
		Version: "1.0.0",
	}

	runCmd := h.buildRunCommand()
	rootCmd.AddCommand(runCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(h.stderr, "%sError:%s %v\n", ctc.ForegroundRed, ctc.Reset, err)
		os.Exit(1)
	}
}

func (h *Hades) buildRunCommand() *cobra.Command {
	var (
		configDir string
		targets   []string
		envVars   []string
		dryRun    bool
		follow    bool
	)

	cmd := &cobra.Command{
		Use:           "run [plan]",
		Short:         "Execute a plan",
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !follow {
				showDemoTable(h.stdout)
				return nil
			}
			planName := args[0]
			return h.runPlan(planName, configDir, targets, envVars, dryRun)
		},
	}

	cmd.Flags().StringVarP(&configDir, "config-dir", "c", ".", "Directory to search for YAML config files (default: current directory)")
	cmd.Flags().StringSliceVarP(&targets, "target", "t", nil, "Target groups to execute on")
	cmd.Flags().StringSliceVarP(&envVars, "env", "e", nil, "Environment variables (KEY=VALUE)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be executed without running")
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Stream output in real-time (follow mode)")

	return cmd
}

func (h *Hades) runPlan(planName, configDir string, targets, envVars []string, dryRun bool) error {
	// Load and merge all YAML files from the config directory
	file, err := h.loader.LoadDirectory(configDir)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate the file structure
	if err := h.loader.Validate(file); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Load the plan
	plan, err := h.loader.LoadPlan(file, planName)
	if err != nil {
		return fmt.Errorf("failed to load plan: %w", err)
	}

	// Parse environment variables from CLI
	env, err := h.parseEnvVars(envVars)
	if err != nil {
		return fmt.Errorf("failed to parse environment variables: %w", err)
	}

	// Expand environment variables (${VAR})
	expandedEnv, err := h.loader.ExpandEnv(env)
	if err != nil {
		return fmt.Errorf("failed to expand environment variables: %w", err)
	}

	// Validate environment variables against plan
	if err := loader.ValidatePlanEnv(file, planName, expandedEnv); err != nil {
		return fmt.Errorf("environment validation failed: %w", err)
	}

	// Load inventory from the same config directory
	inv, err := inventory.LoadDirectory(configDir)
	if err != nil {
		return fmt.Errorf("failed to load inventory: %w", err)
	}

	// Create SSH client
	sshClient := ssh.NewClient()
	defer sshClient.Close()

	// Create executor
	exec := executor.New(sshClient, h.stdout, h.stderr)

	// Execute plan or dry-run
	ctx := context.Background()
	if dryRun {
		return exec.DryRun(ctx, file, plan, planName, inv, targets, expandedEnv)
	}

	result, err := exec.ExecutePlan(ctx, file, plan, planName, inv, targets, expandedEnv)
	if err != nil {
		return fmt.Errorf("execution failed: %w", err)
	}

	if result.Failed {
		return fmt.Errorf("plan failed")
	}

	return nil
}

func (h *Hades) parseEnvVars(envVars []string) (map[string]string, error) {
	env := make(map[string]string)
	for _, ev := range envVars {
		parts := strings.SplitN(ev, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid environment variable format: %s (expected KEY=VALUE)", ev)
		}
		env[parts[0]] = parts[1]
	}
	return env, nil
}

type executionLog struct {
	timestamp string
	host      string
	message   string
}

type executionState struct {
	planName    string
	runID       string
	currentStep int
	totalSteps  int
	stepName    string
	batchInfo   string
	logs        []executionLog
	completed   bool
	failed      bool
	duration    time.Duration
}

func showDemoTable(out *os.File) {
	// Recover from liveterm panics (happens when not in a TTY)
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintln(out, "\nDemo mode requires an interactive terminal.")
			fmt.Fprintln(out, "Run this command in a real terminal, or use -f flag for follow mode.")
		}
	}()

	state := &executionState{
		planName:   "check",
		runID:      "hades-" + time.Now().Format("20060102-150405"),
		totalSteps: 2,
		logs:       make([]executionLog, 0),
	}
	var mu sync.Mutex

	// Configure liveterm
	liveterm.RefreshInterval = 100 * time.Millisecond
	liveterm.Output = out

	// Set update function that renders execution output
	liveterm.SetMultiLinesUpdateFx(func() []string {
		mu.Lock()
		defer mu.Unlock()

		lines := []string{}

		// Header
		lines = append(lines, strings.Repeat("=", 40))
		lines = append(lines, fmt.Sprintf("Plan: %s", state.planName))
		lines = append(lines, strings.Repeat("=", 40))
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("Run ID: %s", state.runID))
		lines = append(lines, fmt.Sprintf("Started: %s", time.Now().Format(time.RFC3339)))
		lines = append(lines, "")

		// Current step progress
		if state.currentStep > 0 {
			lines = append(lines, fmt.Sprintf("Step %d/%d: %s", state.currentStep, state.totalSteps, state.stepName))
			if state.batchInfo != "" {
				lines = append(lines, state.batchInfo)
			}
			lines = append(lines, "")
		}

		// Show last 15 log lines
		startIdx := 0
		if len(state.logs) > 15 {
			startIdx = len(state.logs) - 15
		}
		for i := startIdx; i < len(state.logs); i++ {
			log := state.logs[i]
			lines = append(lines, fmt.Sprintf("[%s] [%s] %s", log.timestamp, log.host, log.message))
		}

		// Completion message
		if state.completed {
			lines = append(lines, "")
			if state.failed {
				lines = append(lines, "✗ Plan failed")
			} else {
				lines = append(lines, "✓ Plan completed successfully")
			}
			lines = append(lines, fmt.Sprintf("Duration: %s", state.duration.Round(time.Millisecond)))
		}

		return lines
	})

	// Start live updates
	liveterm.Start()
	startTime := time.Now()

	addLog := func(host, message string) {
		mu.Lock()
		defer mu.Unlock()
		state.logs = append(state.logs, executionLog{
			timestamp: time.Now().Format("15:04:05"),
			host:      host,
			message:   message,
		})
	}

	// Simulate execution
	go func() {
		// Step 1: setup
		mu.Lock()
		state.currentStep = 1
		state.stepName = "setup"
		state.batchInfo = "Batch 1/1 (2 hosts)"
		mu.Unlock()
		time.Sleep(time.Millisecond * 200)

		addLog("app-1", "Running: apt-get update")
		time.Sleep(time.Millisecond * 300)
		addLog("app-2", "Running: apt-get update")
		time.Sleep(time.Millisecond * 400)
		addLog("app-1", "Reading package lists...")
		time.Sleep(time.Millisecond * 300)
		addLog("app-2", "Reading package lists...")
		time.Sleep(time.Millisecond * 350)
		addLog("app-1", "✓ setup completed (exit 0)")
		time.Sleep(time.Millisecond * 200)
		addLog("app-2", "✓ setup completed (exit 0)")
		time.Sleep(time.Millisecond * 500)

		// Step 2: deploy
		mu.Lock()
		state.currentStep = 2
		state.stepName = "deploy"
		state.batchInfo = "Batch 1/1 (2 hosts)"
		mu.Unlock()
		time.Sleep(time.Millisecond * 200)

		addLog("app-1", "Running: docker pull myapp:latest")
		time.Sleep(time.Millisecond * 300)
		addLog("app-2", "Running: docker pull myapp:latest")
		time.Sleep(time.Millisecond * 400)
		addLog("app-1", "latest: Pulling from library/myapp")
		time.Sleep(time.Millisecond * 500)
		addLog("app-2", "latest: Pulling from library/myapp")
		time.Sleep(time.Millisecond * 400)
		addLog("app-1", "Digest: sha256:abc123...")
		time.Sleep(time.Millisecond * 350)
		addLog("app-1", "Status: Downloaded newer image")
		time.Sleep(time.Millisecond * 300)
		addLog("app-2", "Error: manifest unknown")
		time.Sleep(time.Millisecond * 200)
		addLog("app-1", "Running: docker-compose up -d")
		time.Sleep(time.Millisecond * 600)
		addLog("app-1", "Creating network myapp_default")
		time.Sleep(time.Millisecond * 300)
		addLog("app-1", "Creating container myapp_web_1")
		time.Sleep(time.Millisecond * 400)
		addLog("app-1", "✓ deploy completed (exit 0)")
		time.Sleep(time.Millisecond * 300)
		addLog("app-2", "✗ deploy failed (exit 1)")
		time.Sleep(time.Millisecond * 500)

		// Mark as completed
		mu.Lock()
		state.completed = true
		state.failed = true
		state.duration = time.Since(startTime)
		mu.Unlock()

		time.Sleep(time.Second * 1)
	}()

	// Wait for completion
	time.Sleep(time.Second * 8)
	liveterm.Stop(false)
}
