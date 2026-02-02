package schema

type Job struct {
	Local     bool                `yaml:"local"`
	Env       map[string]Env      `yaml:"env"`
	Artifacts map[string]Artifact `yaml:"artifacts"`
	Actions   []Action            `yaml:"actions"`
}

type Artifact struct {
	Path string `yaml:"path"`
}

type Action struct {
	Run      *ActionRun      `yaml:"run,omitempty"`
	Copy     *ActionCopy     `yaml:"copy,omitempty"`
	Template *ActionTemplate `yaml:"template,omitempty"`
	Mkdir    *ActionMkdir    `yaml:"mkdir,omitempty"`
	Push     *ActionPush     `yaml:"push,omitempty"`
	Pull     *ActionPull     `yaml:"pull,omitempty"`
	Wait     *ActionWait     `yaml:"wait,omitempty"`
}

type ActionRun string

type ActionCopy struct {
	Src      string `yaml:"src,omitempty"`
	Dst      string `yaml:"dst"`
	Artifact string `yaml:"artifact,omitempty"`
}

type ActionTemplate struct {
	Src string `yaml:"src"`
	Dst string `yaml:"dst"`
}

type ActionMkdir struct {
	Path string `yaml:"path"`
	Mode uint32 `yaml:"mode"`
}

type ActionPush struct {
	Registry string `yaml:"registry"`
	Artifact string `yaml:"artifact"`
	Name     string `yaml:"name"`
	Tag      string `yaml:"tag"`
}

type ActionPull struct {
	Registry string `yaml:"registry"`
	Name     string `yaml:"name"`
	Tag      string `yaml:"tag"`
	To       string `yaml:"to"`
}

type ActionWait struct {
	Message string `yaml:"message,omitempty"`
	Timeout string `yaml:"timeout,omitempty"`
}
