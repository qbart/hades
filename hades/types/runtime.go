package types

import (
	"fmt"

	"github.com/SoftKiwiGames/hades/hades/ssh"
	"github.com/google/uuid"
)

type Runtime struct {
	SSHClient ssh.Client
	Env       map[string]string
	RunID     string
	Plan      string
	Target    string
	Host      ssh.Host
}

func NewRuntime(sshClient ssh.Client, plan string, target string, host ssh.Host, userEnv map[string]string) *Runtime {
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
		SSHClient: sshClient,
		Env:       env,
		RunID:     runID,
		Plan:      plan,
		Target:    target,
		Host:      host,
	}
}

func (r *Runtime) EnvSlice() []string {
	var envSlice []string
	for k, v := range r.Env {
		envSlice = append(envSlice, fmt.Sprintf("%s=%s", k, v))
	}
	return envSlice
}
