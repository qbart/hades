package schema

type Plan struct {
	Steps []Step `yaml:"steps"`
}

type Step struct {
	Name        string            `yaml:"name"`
	Job         string            `yaml:"job"`
	Targets     []string          `yaml:"targets"`
	Env         map[string]string `yaml:"env,omitempty"`
	Parallelism string            `yaml:"parallelism,omitempty"`
	Limit       int               `yaml:"limit,omitempty"`
}
