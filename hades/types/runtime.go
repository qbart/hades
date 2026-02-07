package types

import (
	"fmt"
	"io"

	"github.com/SoftKiwiGames/hades/hades/artifacts"
	"github.com/SoftKiwiGames/hades/hades/registry"
	"github.com/SoftKiwiGames/hades/hades/ssh"
	"github.com/google/uuid"
)

type Runtime struct {
	SSHClient      ssh.Client
	ArtifactMgr    artifacts.Manager
	RegistryMgr    registry.Manager
	Env            map[string]string
	RunID          string
	Plan           string
	Target         string
	Host           ssh.Host
	Stdout         io.Writer // Logs only
	Stderr         io.Writer // Logs only
	ConsoleStdout  io.Writer // Console only
	ConsoleStderr  io.Writer // Console only
	ActionDesc     string    // For formatted console messages
}

func NewRuntime(sshClient ssh.Client, artifactMgr artifacts.Manager, registryMgr registry.Manager, plan string, target string, host ssh.Host, userEnv map[string]string, stdout, stderr io.Writer, consoleStdout, consoleStderr io.Writer) *Runtime {
	runID := uuid.New().String()

	// Build environment with HADES_* built-ins
	env := make(map[string]string)

	// Copy user-provided env
	for k, v := range userEnv {
		env[k] = v
	}

	// Add HADES_* built-ins (these cannot be overridden by users)
	env["HADES_RUN_ID"] = runID
	env["HADES_PLAN"] = plan
	env["HADES_TARGET"] = target
	env["HADES_HOST_NAME"] = host.Name
	env["HADES_HOST_ADDR"] = host.Address

	return &Runtime{
		SSHClient:     sshClient,
		ArtifactMgr:   artifactMgr,
		RegistryMgr:   registryMgr,
		Env:           env,
		RunID:         runID,
		Plan:          plan,
		Target:        target,
		Host:          host,
		Stdout:        stdout,
		Stderr:        stderr,
		ConsoleStdout: consoleStdout,
		ConsoleStderr: consoleStderr,
	}
}

func (r *Runtime) EnvSlice() []string {
	var envSlice []string
	for k, v := range r.Env {
		envSlice = append(envSlice, fmt.Sprintf("%s=%s", k, v))
	}
	return envSlice
}
