package hades

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/SoftKiwiGames/hades/hades/executor"
	"github.com/SoftKiwiGames/hades/hades/inventory"
	"github.com/SoftKiwiGames/hades/hades/loader"
	"github.com/SoftKiwiGames/hades/hades/ssh"
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
	initCmd := h.buildInitCommand()
	rootCmd.AddCommand(runCmd, initCmd)

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
	)

	cmd := &cobra.Command{
		Use:           "run [plan]",
		Short:         "Execute a plan",
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			planName := args[0]
			return h.runPlan(planName, configDir, targets, envVars, dryRun)
		},
	}

	cmd.Flags().StringVarP(&configDir, "config-dir", "c", ".", "Directory to search for YAML config files (default: current directory)")
	cmd.Flags().StringSliceVarP(&targets, "target", "t", nil, "Target groups to execute on")
	cmd.Flags().StringSliceVarP(&envVars, "env", "e", nil, "Environment variables (KEY=VALUE)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be executed without running")

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
