package cloud

import (
	"context"
	"fmt"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

type HetznerConfig struct {
	Token string
}

func HetznerInstances(ctx context.Context, cfg HetznerConfig) ([]CloudInstance, error) {
	if cfg.Token == "" {
		return nil, fmt.Errorf("hetzner: token is required")
	}

	client := hcloud.NewClient(hcloud.WithToken(cfg.Token))

	servers, err := client.Server.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("hetzner: failed to list servers: %w", err)
	}

	instances := make([]CloudInstance, 0, len(servers))
	for _, s := range servers {
		inst := CloudInstance{
			Name: s.Name,
			Tags: s.Labels,
		}

		if !s.PublicNet.IPv4.IP.IsUnspecified() {
			inst.PublicIPv4 = s.PublicNet.IPv4.IP
		}
		if !s.PublicNet.IPv6.IP.IsUnspecified() {
			inst.PublicIPv6 = s.PublicNet.IPv6.IP
		}

		instances = append(instances, inst)
	}

	return instances, nil
}
