package schema

type Registries map[string]RegistryConfig

type RegistryConfig struct {
	Type     string            `yaml:"type"`
	Path     string            `yaml:"path,omitempty"`
	Bucket   string            `yaml:"bucket,omitempty"`
	Region   string            `yaml:"region,omitempty"`
	Endpoint string            `yaml:"endpoint,omitempty"`
	Params   map[string]string `yaml:"params,omitempty"`
}
