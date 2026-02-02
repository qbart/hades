package schema

type File struct {
	Jobs       map[string]Job  `yaml:"jobs"`
	Plans      map[string]Plan `yaml:"plans"`
	Registries Registries      `yaml:"registries,omitempty"`
}
