package inventory

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SoftKiwiGames/hades/hades/ssh"
	"github.com/SoftKiwiGames/hades/hades/utils"
	"gopkg.in/yaml.v3"
)

type fileInventory struct {
	hosts   []ssh.Host
	targets map[string][]string
}

type inventoryFile struct {
	Hosts   map[string]hostDef  `yaml:"hosts"`
	Targets map[string][]string `yaml:"targets"`
}

type hostDef struct {
	Addr         string `yaml:"addr"`
	User         string `yaml:"user"`
	IdentityFile string `yaml:"identity_file"`
}

func LoadFile(path string) (Inventory, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read inventory file: %w", err)
	}

	var file inventoryFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("failed to parse inventory YAML: %w", err)
	}

	// Convert host definitions to ssh.Host
	var hosts []ssh.Host
	for name, h := range file.Hosts {
		keyPath, err := utils.ExpandPath(h.IdentityFile)
		if err != nil {
			return nil, fmt.Errorf("failed to expand identity_file for host %q: %w", name, err)
		}
		hosts = append(hosts, ssh.Host{
			Name:    name,
			Address: h.Addr,
			User:    h.User,
			KeyPath: keyPath,
		})
	}

	return &fileInventory{
		hosts:   hosts,
		targets: file.Targets,
	}, nil
}

// LoadDirectory recursively walks a directory, finds all .yml and .yaml files,
// and merges them into a single inventory
func LoadDirectory(rootPath string) (Inventory, error) {
	allHosts := make(map[string]ssh.Host)
	allTargets := make(map[string][]string)

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
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}

		var file inventoryFile
		if err := yaml.Unmarshal(data, &file); err != nil {
			// Skip files that don't parse as inventory
			return nil
		}

		// Merge hosts
		for name, h := range file.Hosts {
			if _, exists := allHosts[name]; exists {
				return fmt.Errorf("duplicate host %q found in %s", name, path)
			}
			keyPath, err := utils.ExpandPath(h.IdentityFile)
			if err != nil {
				return fmt.Errorf("failed to expand identity_file for host %q in %s: %w", name, path, err)
			}
			allHosts[name] = ssh.Host{
				Name:    name,
				Address: h.Addr,
				User:    h.User,
				KeyPath: keyPath,
			}
		}

		// Merge targets
		for name, hostList := range file.Targets {
			if _, exists := allTargets[name]; exists {
				return fmt.Errorf("duplicate target %q found in %s", name, path)
			}
			allTargets[name] = hostList
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Convert map to slice for hosts
	hosts := make([]ssh.Host, 0, len(allHosts))
	for _, h := range allHosts {
		hosts = append(hosts, h)
	}

	return &fileInventory{
		hosts:   hosts,
		targets: allTargets,
	}, nil
}

func (f *fileInventory) ResolveTarget(name string) ([]ssh.Host, error) {
	hostNames, ok := f.targets[name]
	if !ok {
		return nil, fmt.Errorf("target %q not found in inventory", name)
	}

	// Build map of hosts by name for quick lookup
	hostMap := make(map[string]ssh.Host)
	for _, h := range f.hosts {
		hostMap[h.Name] = h
	}

	// Resolve host names to Host objects
	var hosts []ssh.Host
	for _, name := range hostNames {
		host, ok := hostMap[name]
		if !ok {
			return nil, fmt.Errorf("host %q referenced in target but not defined", name)
		}
		hosts = append(hosts, host)
	}

	return hosts, nil
}

func (f *fileInventory) AllHosts() []ssh.Host {
	return f.hosts
}
