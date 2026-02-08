package actions

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/SoftKiwiGames/hades/hades/schema"
	"github.com/SoftKiwiGames/hades/hades/ssh"
	"github.com/SoftKiwiGames/hades/hades/types"
	"github.com/wzshiming/ctc"
)

type CopyAction struct {
	Src      string
	Dst      string
	Artifact string
	Mode     uint32
}

func NewCopyAction(action *schema.ActionCopy) Action {
	mode := action.Mode
	if mode == 0 {
		mode = 0644 // Default mode
	}
	return &CopyAction{
		Src:      action.Src,
		Dst:      action.Dst,
		Artifact: action.Artifact,
		Mode:     mode,
	}
}

func (a *CopyAction) Execute(ctx context.Context, runtime *types.Runtime) error {
	// Prepare source and calculate checksum
	var reader io.ReadCloser
	var localChecksum string
	var srcDesc string
	var fileSize int64

	if a.Artifact != "" {
		// ARTIFACTS: Read into memory buffer
		art, err := runtime.ArtifactMgr.Get(a.Artifact)
		if err != nil {
			return fmt.Errorf("failed to get artifact %s: %w", a.Artifact, err)
		}
		defer art.Close()

		// Read entire artifact into memory
		data, err := io.ReadAll(art)
		if err != nil {
			return fmt.Errorf("failed to read artifact: %w", err)
		}

		fileSize = int64(len(data))

		// Calculate checksum from buffer
		h := sha256.New()
		h.Write(data)
		localChecksum = hex.EncodeToString(h.Sum(nil))

		// Create reader from buffer
		reader = io.NopCloser(bytes.NewReader(data))
		srcDesc = fmt.Sprintf("artifact:%s", a.Artifact)

	} else if a.Src != "" {
		// LOCAL FILES: Read file, calculate checksum
		f, err := os.Open(a.Src)
		if err != nil {
			return fmt.Errorf("failed to open source file %s: %w", a.Src, err)
		}

		// Get file size
		stat, err := f.Stat()
		if err != nil {
			f.Close()
			return fmt.Errorf("failed to stat file: %w", err)
		}
		fileSize = stat.Size()

		// Calculate checksum
		localChecksum, err = calculateChecksum(f)
		f.Close()
		if err != nil {
			return fmt.Errorf("failed to calculate checksum: %w", err)
		}

		// Reopen file for copying (reader was consumed by checksum)
		f2, err := os.Open(a.Src)
		if err != nil {
			return fmt.Errorf("failed to reopen source: %w", err)
		}
		reader = f2
		srcDesc = a.Src

	} else {
		return fmt.Errorf("either src or artifact must be specified")
	}
	defer reader.Close()

	// Create SSH session
	sess, err := runtime.SSHClient.Connect(ctx, runtime.Host)
	if err != nil {
		return fmt.Errorf("failed to connect to host: %w", err)
	}
	defer sess.Close()

	// Expand environment variables in destination path
	dst := ExpandEnvVars(a.Dst, runtime.Env)

	// Get remote checksum (with fallback on error)
	remoteChecksum, exists, err := getRemoteChecksum(ctx, sess, dst)
	if err != nil {
		// Severe error - fail
		return fmt.Errorf("failed to check remote file: %w", err)
	}

	// Format file size
	sizeStr := formatFileSize(fileSize)

	// Compare checksums and decide
	if exists && localChecksum == remoteChecksum {
		// SKIP: Checksums match - file is identical
		// Log: plain text
		fmt.Fprintf(runtime.Stdout, "Skipping %s (%s, already up to date)\n", dst, sizeStr)
		// Console: with action format and skip symbol (blue)
		if runtime.ConsoleStdout != nil {
			fmt.Fprintf(runtime.ConsoleStdout, "[%s] %s○%s Action %s: skipped (%s, %s already up to date)\n",
				runtime.Host.Name, ctc.ForegroundBlue, ctc.Reset, runtime.ActionDesc, dst, sizeStr)
		}
		return nil
	}

	// Copy file (checksums differ, file doesn't exist, or tool missing)
	if err := sess.CopyFile(ctx, reader, dst, a.Mode); err != nil {
		return fmt.Errorf("failed to copy %s to %s: %w", srcDesc, dst, err)
	}

	// Log successful copy with size
	fmt.Fprintf(runtime.Stdout, "Copied %s to %s (%s)\n", srcDesc, dst, sizeStr)

	return nil
}

func (a *CopyAction) DryRun(ctx context.Context, runtime *types.Runtime) string {
	dst := ExpandEnvVars(a.Dst, runtime.Env)

	// Try to get file size for display
	var sizeInfo string
	if a.Src != "" {
		if stat, err := os.Stat(a.Src); err == nil {
			sizeInfo = fmt.Sprintf(", %s", formatFileSize(stat.Size()))
		}
	}

	if a.Artifact != "" {
		return fmt.Sprintf("copy: artifact=%s to=%s (mode: %o%s, verify checksum)", a.Artifact, dst, a.Mode, sizeInfo)
	}
	return fmt.Sprintf("copy: %s to %s (mode: %o%s, verify checksum)", a.Src, dst, a.Mode, sizeInfo)
}

// calculateChecksum reads content and returns SHA-256 hash
func calculateChecksum(reader io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, reader); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// formatFileSize formats file size in human-readable format
// < 1 KB: bytes, < 1 MB: KiB, >= 1 MB: MiB
func formatFileSize(size int64) string {
	const (
		KB = 1024
		MB = 1024 * 1024
	)

	switch {
	case size < KB:
		return fmt.Sprintf("%d bytes", size)
	case size < MB:
		return fmt.Sprintf("%.2f KiB", float64(size)/KB)
	default:
		return fmt.Sprintf("%.2f MiB", float64(size)/MB)
	}
}

// getRemoteChecksum returns (checksum, exists, error)
// If sha256sum is not available, returns ("", false, nil) to trigger fallback
func getRemoteChecksum(ctx context.Context, sess ssh.Session, remotePath string) (string, bool, error) {
	var stdout bytes.Buffer

	// Use shell command that handles missing file gracefully
	cmd := fmt.Sprintf("sha256sum %s 2>/dev/null || echo NOTFOUND", remotePath)

	err := sess.Run(ctx, cmd, &stdout, io.Discard)
	if err != nil {
		// Command failed - likely sha256sum not found or severe error
		// Return false to trigger fallback (copy without check)
		return "", false, nil
	}

	output := strings.TrimSpace(stdout.String())

	// File doesn't exist or sha256sum not available
	if output == "NOTFOUND" || output == "" {
		return "", false, nil
	}

	// Parse "abc123...  /path/to/file" format
	parts := strings.Fields(output)
	if len(parts) < 1 {
		return "", false, nil
	}

	return parts[0], true, nil
}
