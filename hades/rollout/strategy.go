package rollout

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/SoftKiwiGames/hades/hades/ssh"
)

type Strategy struct {
	Parallelism int
	Limit       int
}

// ParseStrategy parses a parallelism string and returns a Strategy
// Supports:
// - Empty string: all hosts in parallel
// - "1": serial execution
// - "5": 5 hosts at a time
// - "40%": 40% of hosts at a time
func ParseStrategy(parallelism string, hostCount int) (*Strategy, error) {
	strategy := &Strategy{}

	// If empty, default to all hosts in parallel
	if parallelism == "" {
		strategy.Parallelism = hostCount
		return strategy, nil
	}

	// Check for percentage
	if strings.HasSuffix(parallelism, "%") {
		percentStr := strings.TrimSuffix(parallelism, "%")
		percent, err := strconv.ParseFloat(percentStr, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid percentage format: %s", parallelism)
		}

		if percent <= 0 || percent > 100 {
			return nil, fmt.Errorf("percentage must be between 0 and 100, got: %.2f", percent)
		}

		// Calculate number of hosts (at least 1)
		count := int(float64(hostCount) * (percent / 100.0))
		if count < 1 {
			count = 1
		}
		strategy.Parallelism = count
		return strategy, nil
	}

	// Parse as integer
	count, err := strconv.Atoi(parallelism)
	if err != nil {
		return nil, fmt.Errorf("invalid parallelism format: %s (expected number or percentage)", parallelism)
	}

	if count < 1 {
		return nil, fmt.Errorf("parallelism must be at least 1, got: %d", count)
	}

	strategy.Parallelism = count
	return strategy, nil
}

// CreateBatches splits hosts into batches based on the strategy
func (s *Strategy) CreateBatches(hosts []ssh.Host) [][]ssh.Host {
	if len(hosts) == 0 {
		return nil
	}

	// Apply limit if specified
	selectedHosts := hosts
	if s.Limit > 0 && s.Limit < len(hosts) {
		selectedHosts = hosts[:s.Limit]
	}

	// If parallelism >= host count, single batch
	if s.Parallelism >= len(selectedHosts) {
		return [][]ssh.Host{selectedHosts}
	}

	// Split into batches
	var batches [][]ssh.Host
	for i := 0; i < len(selectedHosts); i += s.Parallelism {
		end := i + s.Parallelism
		if end > len(selectedHosts) {
			end = len(selectedHosts)
		}
		batches = append(batches, selectedHosts[i:end])
	}

	return batches
}
