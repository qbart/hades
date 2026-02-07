package actions

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/SoftKiwiGames/hades/config"
	"github.com/SoftKiwiGames/hades/hades/schema"
	"github.com/SoftKiwiGames/hades/hades/types"
)

type GpgAction struct {
	Src     string
	Path    string
	Mode    uint32
	Dearmor bool
}

func NewGpgAction(action *schema.ActionGpg) Action {
	mode := action.Mode
	if mode == 0 {
		mode = 0644 // Default mode for GPG keyrings
	}
	return &GpgAction{
		Src:     action.Src,
		Path:    action.Path,
		Mode:    mode,
		Dearmor: action.Dearmor,
	}
}

func (a *GpgAction) Execute(ctx context.Context, runtime *types.Runtime) error {
	// Expand environment variables in fields
	src, err := expandEnv(a.Src, runtime.Env)
	if err != nil {
		return fmt.Errorf("failed to expand src: %w", err)
	}

	path, err := expandEnv(a.Path, runtime.Env)
	if err != nil {
		return fmt.Errorf("failed to expand path: %w", err)
	}

	// Download GPG keyring from URL
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, src, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers similar to curl
	req.Header.Set("User-Agent", "hades/" + config.Version)
	req.Header.Set("Accept", "*/*")

	// Create HTTP client with TLS 1.0+ support (like curl -1)
	client := &http.Client{
		Timeout: 5 * time.Minute,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS10, // Allow TLS 1.0+ like curl -1
			},
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download GPG keyring from %s: %w", src, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Read a bit of the response body for better error messages
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("failed to download GPG keyring from %s: HTTP %d %s\nResponse: %s",
			src, resp.StatusCode, resp.Status, string(body))
	}

	// Create SSH session
	sess, err := runtime.SSHClient.Connect(ctx, runtime.Host)
	if err != nil {
		return fmt.Errorf("failed to connect to host: %w", err)
	}
	defer sess.Close()

	if a.Dearmor {
		// Copy to temp file, dearmor, then move to final location
		tmpPath := fmt.Sprintf("/tmp/hades-gpg-%s.asc", runtime.RunID)

		// Copy downloaded content to temp file
		if err := sess.CopyFile(ctx, resp.Body, tmpPath, 0644); err != nil {
			return fmt.Errorf("failed to copy GPG keyring to temp location: %w", err)
		}

		// Run gpg --dearmor to convert ASCII to binary
		dearmorCmd := fmt.Sprintf("gpg --dearmor -o %s < %s && chmod %o %s && rm -f %s",
			path, tmpPath, a.Mode, path, tmpPath)

		var stderr io.Writer
		if err := sess.Run(ctx, dearmorCmd, nil, stderr); err != nil {
			return fmt.Errorf("failed to dearmor GPG keyring: %w", err)
		}
	} else {
		// Copy GPG keyring directly to remote host
		if err := sess.CopyFile(ctx, resp.Body, path, a.Mode); err != nil {
			return fmt.Errorf("failed to copy GPG keyring to host: %w", err)
		}
	}

	return nil
}

func (a *GpgAction) DryRun(ctx context.Context, runtime *types.Runtime) string {
	// Expand for dry-run display
	src, _ := expandEnv(a.Src, runtime.Env)
	path, _ := expandEnv(a.Path, runtime.Env)

	if a.Dearmor {
		return fmt.Sprintf("gpg: download %s, dearmor to %s (mode: %o)", src, path, a.Mode)
	}
	return fmt.Sprintf("gpg: download %s to %s (mode: %o)", src, path, a.Mode)
}
