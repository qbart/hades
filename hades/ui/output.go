package ui

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/wzshiming/ctc"
)

type Output struct {
	stdout io.Writer
	stderr io.Writer
}

func NewOutput(stdout, stderr io.Writer) *Output {
	return &Output{
		stdout: stdout,
		stderr: stderr,
	}
}

// Header prints a formatted section header
func (o *Output) Header(text string) {
	fmt.Fprintf(o.stdout, "\n%s\n", strings.Repeat("=", len(text)))
	fmt.Fprintf(o.stdout, "%s\n", text)
	fmt.Fprintf(o.stdout, "%s\n\n", strings.Repeat("=", len(text)))
}

// Section prints a section title
func (o *Output) Section(text string) {
	fmt.Fprintf(o.stdout, "\n%s\n%s\n", text, strings.Repeat("-", len(text)))
}

// Info prints an informational message
func (o *Output) Info(format string, args ...any) {
	fmt.Fprintf(o.stdout, format+"\n", args...)
}

// Success prints a success message with checkmark
func (o *Output) Success(format string, args ...any) {
	fmt.Fprintf(o.stdout, "✓ "+format+"\n", args...)
}

// Error prints an error message
func (o *Output) Error(format string, args ...any) {
	fmt.Fprintf(o.stderr, o.DotRed()+" "+format+"\n", args...)
}

// Warning prints a warning message
func (o *Output) Warning(format string, args ...any) {
	fmt.Fprintf(o.stdout, "⚠ "+format+"\n", args...)
}

// HostLog prints a host-specific log message
func (o *Output) HostLog(host, format string, args ...any) {
	timestamp := time.Now().Format("15:04:05")
	message := fmt.Sprintf(format, args...)
	fmt.Fprintf(o.stdout, "[%s] [%s] %s\n", timestamp, host, message)
}

// StepProgress prints step progress
func (o *Output) StepProgress(current, total int, name string) {
	fmt.Fprintf(o.stdout, "\nStep %d/%d: %s\n", current, total, name)
}

// BatchProgress prints batch progress
func (o *Output) BatchProgress(current, total, hostCount int) {
	fmt.Fprintf(o.stdout, "\nBatch %d/%d (%d hosts)\n", current, total, hostCount)
}

// DryRunHeader prints dry-run mode header
func (o *Output) DryRunHeader(plan string) {
	o.Header(fmt.Sprintf("DRY-RUN: %s", plan))
	o.Info("This will execute the following actions:")
}

// PlanStarted prints plan start information
func (o *Output) PlanStarted(plan, runID string) {
	o.Header(fmt.Sprintf("Plan: %s", plan))
	o.Info("Run ID: %s", runID)
	o.Info("Started: %s", time.Now().Format(time.RFC3339))
}

// PlanCompleted prints plan completion summary
func (o *Output) PlanCompleted(duration time.Duration) {
	o.Success("Plan completed successfully")
	o.Info("Duration: %s", duration)
}

// PlanFailed prints plan failure information
func (o *Output) PlanFailed(step, host string, err error) {
	o.Error("Plan failed")
	if step != "" {
		o.Info("Failed step: %s", step)
	}
	if host != "" {
		o.Info("Failed host: %s", host)
	}
	o.Info("Error: %v", err)
}

func (o *Output) DotRed() string {
	return fmt.Sprint(ctc.ForegroundRed, "•", ctc.Reset)
}
