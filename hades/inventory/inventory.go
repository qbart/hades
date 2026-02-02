package inventory

import "github.com/SoftKiwiGames/hades/hades/ssh"

type Inventory interface {
	ResolveTarget(name string) ([]ssh.Host, error)
	AllHosts() []ssh.Host
}
