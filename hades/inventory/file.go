package inventory

import (
	"fmt"
	"os"

	"github.com/SoftKiwiGames/hades/hades/ssh"
	"gopkg.in/yaml.v3"
)

type fileInventory struct {
	hosts   []ssh.Host
	targets map[string][]string
}

type inventoryFile struct {
	Hosts   []hostDef              `yaml:"hosts"`
	Targets map[string][]string    `yaml:"targets"`
}

type hostDef struct {
	Name    string `yaml:"name"`
	Addr    string `yaml:"addr"`
	User    string `yaml:"user"`
	Key     string `yaml:"key"`
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
	hosts := make([]ssh.Host, len(file.Hosts))
	for i, h := range file.Hosts {
		hosts[i] = ssh.Host{
			Name:    h.Name,
			Address: h.Addr,
			User:    h.User,
			KeyPath: h.Key,
		}
	}

	return &fileInventory{
		hosts:   hosts,
		targets: file.Targets,
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
