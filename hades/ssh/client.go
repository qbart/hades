package ssh

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/crypto/ssh"
)

type Client interface {
	Connect(ctx context.Context, host Host) (Session, error)
	Close() error
}

type Host struct {
	Name    string
	Address string
	User    string
	KeyPath string
}

type client struct {
	connections map[string]*ssh.Client
}

func NewClient() Client {
	return &client{
		connections: make(map[string]*ssh.Client),
	}
}

func (c *client) Connect(ctx context.Context, host Host) (Session, error) {
	// Check if we already have a connection to this host
	key := fmt.Sprintf("%s@%s", host.User, host.Address)
	if conn, ok := c.connections[key]; ok {
		return newSession(conn, host)
	}

	// Read private key
	keyData, err := os.ReadFile(host.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read SSH key %s: %w", host.KeyPath, err)
	}

	signer, err := ssh.ParsePrivateKey(keyData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SSH key: %w", err)
	}

	// Configure SSH client
	config := &ssh.ClientConfig{
		User: host.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: Add host key verification
	}

	// Connect to the host
	addr := host.Address
	if addr[len(addr)-3:] != ":22" && addr[len(addr)-4:] != ":22" {
		// Check if address already has port
		hasPort := false
		for i := len(addr) - 1; i >= 0 && i > len(addr)-6; i-- {
			if addr[i] == ':' {
				hasPort = true
				break
			}
		}
		if !hasPort {
			addr = addr + ":22"
		}
	}

	conn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", addr, err)
	}

	// Store connection for reuse
	c.connections[key] = conn

	return newSession(conn, host)
}

func (c *client) Close() error {
	var firstErr error
	for key, conn := range c.connections {
		if err := conn.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		delete(c.connections, key)
	}
	return firstErr
}
