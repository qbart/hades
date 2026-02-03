package rollout

import (
	"testing"

	"github.com/SoftKiwiGames/hades/hades/ssh"
)

func TestParseStrategy(t *testing.T) {
	tests := []struct {
		name        string
		parallelism string
		hostCount   int
		want        int
		wantErr     bool
	}{
		{
			name:        "empty defaults to all hosts",
			parallelism: "",
			hostCount:   10,
			want:        10,
			wantErr:     false,
		},
		{
			name:        "serial execution",
			parallelism: "1",
			hostCount:   10,
			want:        1,
			wantErr:     false,
		},
		{
			name:        "fixed number",
			parallelism: "5",
			hostCount:   10,
			want:        5,
			wantErr:     false,
		},
		{
			name:        "percentage 40%",
			parallelism: "40%",
			hostCount:   10,
			want:        4,
			wantErr:     false,
		},
		{
			name:        "percentage 50%",
			parallelism: "50%",
			hostCount:   10,
			want:        5,
			wantErr:     false,
		},
		{
			name:        "percentage rounds down but min 1",
			parallelism: "10%",
			hostCount:   5,
			want:        1,
			wantErr:     false,
		},
		{
			name:        "invalid percentage",
			parallelism: "abc%",
			hostCount:   10,
			want:        0,
			wantErr:     true,
		},
		{
			name:        "invalid number",
			parallelism: "abc",
			hostCount:   10,
			want:        0,
			wantErr:     true,
		},
		{
			name:        "zero parallelism",
			parallelism: "0",
			hostCount:   10,
			want:        0,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy, err := ParseStrategy(tt.parallelism, tt.hostCount)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseStrategy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && strategy.Parallelism != tt.want {
				t.Errorf("ParseStrategy() = %v, want %v", strategy.Parallelism, tt.want)
			}
		})
	}
}

func TestCreateBatches(t *testing.T) {
	// Create test hosts
	hosts := make([]ssh.Host, 10)
	for i := 0; i < 10; i++ {
		hosts[i] = ssh.Host{Name: string(rune('a' + i))}
	}

	tests := []struct {
		name        string
		parallelism int
		limit       int
		hostCount   int
		wantBatches int
		wantSizes   []int
	}{
		{
			name:        "single batch - all hosts",
			parallelism: 10,
			limit:       0,
			hostCount:   10,
			wantBatches: 1,
			wantSizes:   []int{10},
		},
		{
			name:        "serial - one at a time",
			parallelism: 1,
			limit:       0,
			hostCount:   10,
			wantBatches: 10,
			wantSizes:   []int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		},
		{
			name:        "two hosts at a time",
			parallelism: 2,
			limit:       0,
			hostCount:   10,
			wantBatches: 5,
			wantSizes:   []int{2, 2, 2, 2, 2},
		},
		{
			name:        "three hosts at a time",
			parallelism: 3,
			limit:       0,
			hostCount:   10,
			wantBatches: 4,
			wantSizes:   []int{3, 3, 3, 1},
		},
		{
			name:        "with limit - canary",
			parallelism: 1,
			limit:       1,
			hostCount:   10,
			wantBatches: 1,
			wantSizes:   []int{1},
		},
		{
			name:        "with limit and parallelism",
			parallelism: 2,
			limit:       5,
			hostCount:   10,
			wantBatches: 3,
			wantSizes:   []int{2, 2, 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := &Strategy{
				Parallelism: tt.parallelism,
				Limit:       tt.limit,
			}

			batches := strategy.CreateBatches(hosts[:tt.hostCount])

			if len(batches) != tt.wantBatches {
				t.Errorf("CreateBatches() batch count = %v, want %v", len(batches), tt.wantBatches)
				return
			}

			for i, batch := range batches {
				if len(batch) != tt.wantSizes[i] {
					t.Errorf("Batch %d size = %v, want %v", i, len(batch), tt.wantSizes[i])
				}
			}
		})
	}
}
