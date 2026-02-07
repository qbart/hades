package loader

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SoftKiwiGames/hades/hades/schema"
	"github.com/SoftKiwiGames/hades/hades/utils"
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

// LoadDirectory recursively walks a directory, finds all .yml and .yaml files,
// and merges them into a single schema.File
func (l *Loader) LoadDirectory(rootPath string) (*schema.File, error) {
	merged := &schema.File{
		Jobs:       make(map[string]schema.Job),
		Plans:      make(map[string]schema.Plan),
		Registries: make(schema.Registries),
	}

	err := filepath.WalkDir(rootPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-YAML files
		if d.IsDir() {
			return nil
		}

		if !utils.FileHasValidExtension(path) {
			return nil
		}

		// Load the file
		file, err := l.LoadFile(path)
		if err != nil {
			// Skip files that don't parse - they may not be Hades config files
			return nil
		}

		// Skip files that don't contain any Hades configuration
		if len(file.Jobs) == 0 && len(file.Plans) == 0 && len(file.Registries) == 0 {
			return nil
		}

		// Merge jobs
		for name, job := range file.Jobs {
			if _, exists := merged.Jobs[name]; exists {
				return fmt.Errorf("duplicate job %q found in %s", name, path)
			}
			merged.Jobs[name] = job
		}

		// Merge plans
		for name, plan := range file.Plans {
			if _, exists := merged.Plans[name]; exists {
				return fmt.Errorf("duplicate plan %q found in %s", name, path)
			}
			merged.Plans[name] = plan
		}

		// Merge registries
		for name, reg := range file.Registries {
			if _, exists := merged.Registries[name]; exists {
				return fmt.Errorf("duplicate registry %q found in %s", name, path)
			}
			merged.Registries[name] = reg
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return merged, nil
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
			if action.Gpg != nil {
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
