package loader

import (
	"fmt"
	"os"

	"github.com/SoftKiwiGames/hades/hades/schema"
	"gopkg.in/yaml.v3"
)

type Loader struct{}

func New() *Loader {
	return &Loader{}
}

// LoadFile parses a YAML file and returns the schema
func (l *Loader) LoadFile(path string) (*schema.File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	var file schema.File
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &file, nil
}

// LoadJob retrieves a job by name from the file
func (l *Loader) LoadJob(file *schema.File, name string) (*schema.Job, error) {
	job, ok := file.Jobs[name]
	if !ok {
		return nil, fmt.Errorf("job %q not found", name)
	}
	return &job, nil
}

// LoadPlan retrieves a plan by name from the file
func (l *Loader) LoadPlan(file *schema.File, name string) (*schema.Plan, error) {
	plan, ok := file.Plans[name]
	if !ok {
		return nil, fmt.Errorf("plan %q not found", name)
	}
	return &plan, nil
}

// Validate checks the file for structural correctness
func (l *Loader) Validate(file *schema.File) error {
	// Check that all steps reference existing jobs
	for planName, plan := range file.Plans {
		for i, step := range plan.Steps {
			if _, ok := file.Jobs[step.Job]; !ok {
				return fmt.Errorf("plan %q step %d references non-existent job %q", planName, i, step.Job)
			}
		}
	}

	// Check that no action has more than one field set
	for jobName, job := range file.Jobs {
		for i, action := range job.Actions {
			count := 0
			if action.Run != nil {
				count++
			}
			if action.Copy != nil {
				count++
			}
			if action.Template != nil {
				count++
			}
			if action.Mkdir != nil {
				count++
			}
			if action.Push != nil {
				count++
			}
			if action.Pull != nil {
				count++
			}
			if action.Wait != nil {
				count++
			}
			if count == 0 {
				return fmt.Errorf("job %q action %d has no action type set", jobName, i)
			}
			if count > 1 {
				return fmt.Errorf("job %q action %d has multiple action types set", jobName, i)
			}
		}
	}

	return nil
}
