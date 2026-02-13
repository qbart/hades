package hades

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/SoftKiwiGames/hades/hades/cloud"
	"github.com/spf13/cobra"
	"github.com/wzshiming/ctc"
)

func (h *Hades) buildCloudCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cloud [provider]",
		Short: "Interact with cloud providers",
	}

	cmd.AddCommand(h.buildCloudHetznerCommand())

	return cmd
}

func (h *Hades) buildCloudHetznerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hetzner",
		Short: "Hetzner Cloud",
	}

	cmd.AddCommand(h.buildCloudHetznerHostsCommand())

	return cmd
}

func (h *Hades) buildCloudHetznerHostsCommand() *cobra.Command {
	var token string

	cmd := &cobra.Command{
		Use:           "hosts",
		Short:         "List cloud instances",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if token == "" {
				token = os.Getenv("HCLOUD_TOKEN")
			}
			if token == "" {
				return fmt.Errorf("token is required (use --token or HCLOUD_TOKEN)")
			}

			instances, err := cloud.HetznerInstances(
				context.Background(),
				cloud.HetznerConfig{Token: token},
			)
			if err != nil {
				return err
			}

			h.printInstances(instances)
			return nil
		},
	}

	cmd.Flags().StringVar(&token, "token", "", "Hetzner Cloud API token (default: HCLOUD_TOKEN)")

	return cmd
}

func (h *Hades) printInstances(instances []cloud.CloudInstance) {
	if len(instances) == 0 {
		fmt.Fprintln(h.stdout, "No instances found.")
		return
	}

	// Build rows and compute column widths
	type row struct {
		name, ipv4, ipv6, tags string
	}

	rows := make([]row, len(instances))
	nameW, ipv4W, ipv6W := len("NAME"), len("IPV4"), len("IPV6")

	for i, inst := range instances {
		r := row{name: inst.Name}

		r.ipv4 = "-"
		if inst.PublicIPv4 != nil {
			r.ipv4 = inst.PublicIPv4.String()
		}
		r.ipv6 = "-"
		if inst.PublicIPv6 != nil {
			r.ipv6 = inst.PublicIPv6.String()
		}
		r.tags = formatTags(inst.Tags)

		if len(r.name) > nameW {
			nameW = len(r.name)
		}
		if len(r.ipv4) > ipv4W {
			ipv4W = len(r.ipv4)
		}
		if len(r.ipv6) > ipv6W {
			ipv6W = len(r.ipv6)
		}

		rows[i] = r
	}

	pad := 3

	// header (bold)
	fmt.Fprintf(h.stdout, "\033[1m%-*s%-*s%-*s%s\033[0m\n",
		nameW+pad, "NAME",
		ipv4W+pad, "IPV4",
		ipv6W+pad, "IPV6",
		"TAGS",
	)

	for _, r := range rows {
		paddedIPv4 := fmt.Sprintf("%-*s", ipv4W+pad, r.ipv4)
		fmt.Fprintf(h.stdout, "%-*s%s%s%s%-*s%s\n",
			nameW+pad, r.name,
			ctc.ForegroundMagenta, paddedIPv4, ctc.Reset,
			ipv6W+pad, r.ipv6,
			r.tags,
		)
	}
}

func formatTags(tags map[string]string) string {
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, tags[k]))
	}
	return strings.Join(parts, ", ")
}
