package hades

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wzshiming/ctc"
)

type templateFile struct {
	filename string
	content  string
}

func (h *Hades) buildInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:           "init",
		Short:         "Initialize a new Hades project with example files",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return h.runInit()
		},
	}
}

func (h *Hades) runInit() error {
	files := []templateFile{
		{filename: "hades/example/hosts.hades.yaml", content: hostsTemplate},
		{filename: "hades/example/plans.hades.yaml", content: plansTemplate},
		{filename: "hades/example/jobs.hades.yaml", content: jobsTemplate},
		{filename: "hades/example/tpl/sample", content: serverTemplate},
		{filename: "hades/example/tpl/apt-caddy", content: aptCaddyTemplate},
		{filename: "hades/example/tpl/Caddyfile", content: caddyfileTemplate},
	}

	// Compute max filename length for aligned output
	maxLen := 0
	for _, f := range files {
		if len(f.filename) > maxLen {
			maxLen = len(f.filename)
		}
	}

	for _, f := range files {
		padding := strings.Repeat(" ", maxLen-len(f.filename))

		if _, err := os.Stat(f.filename); err == nil {
			fmt.Fprintf(h.stdout, "  %s%s   ..%sskipped%s\n", f.filename, padding, ctc.ForegroundYellow, ctc.Reset)
			continue
		}

		if dir := filepath.Dir(f.filename); dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				fmt.Fprintf(h.stdout, "  %s%s   ..%sfailed%s (%s)\n", f.filename, padding, ctc.ForegroundRed, ctc.Reset, err)
				continue
			}
		}

		if err := os.WriteFile(f.filename, []byte(f.content), 0644); err != nil {
			fmt.Fprintf(h.stdout, "  %s%s   ..%sfailed%s (%s)\n", f.filename, padding, ctc.ForegroundRed, ctc.Reset, err)
			continue
		}

		fmt.Fprintf(h.stdout, "  %s%s   ..%screated%s\n", f.filename, padding, ctc.ForegroundGreen, ctc.Reset)
	}

	return nil
}
var serverTemplate = `
# sample http server

while true; do
  { printf 'HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n\r\nHades\n'; } | nc -l 8080 -q 1
done
`

var aptCaddyTemplate = `
# Source: Caddy
# Site: https://github.com/caddyserver/caddy
# Repository: Caddy / stable
# Description: Fast, multi-platform web server with automatic HTTPS

deb [signed-by=/usr/share/keyrings/caddy-stable-archive-keyring.gpg] https://dl.cloudsmith.io/public/caddy/stable/deb/debian any-version main

deb-src [signed-by=/usr/share/keyrings/caddy-stable-archive-keyring.gpg] https://dl.cloudsmith.io/public/caddy/stable/deb/debian any-version main
`

var caddyfileTemplate = `
# This file was generated during hades run: ${HADES_RUN_ID}
# Do not edit manually.

${DOMAIN} {
	reverse_proxy localhost:8080
}
`

var hostsTemplate = `hosts:
  worker-1:
    addr: 127.0.0.1
    port: 22
    user: root
    identity_file: ~/.ssh/id_ed25519

  worker-2:
    addr: 127.0.0.1
    port: 22
    user: root
    identity_file: ~/.ssh/id_ed25519

targets:
  workers: [worker-1, worker-2]
`


var jobsTemplate = `jobs:
  config:
    actions:
      - name: Setup dirs
        run: |
          set -e
          mkdir -p /root/tpl
          mkdir -p /app/releases
          mkdir -p /app/config

  install-caddy:
    guard:
      if: "! which caddy"
    actions:
      - name: Install deps
        run: |
          apt install -y \
            vim \
            wget \
            curl \
            unzip \
            debian-keyring debian-archive-keyring \
            apt-transport-https

      - name: GPG
        gpg:
          src: https://dl.cloudsmith.io/public/caddy/stable/gpg.key
          path: /usr/share/keyrings/caddy-stable-archive-keyring.gpg
          dearmor: true

      - name: Configure apt
        copy:
          src: tpl/apt-caddy.list
          dst: /etc/apt/sources.list.d/caddy.list
          mode: 0644

      - name: Update permissions
        run: |
          set -e
          chmod o+r /usr/share/keyrings/caddy-stable-archive-keyring.gpg
          chmod o+r /etc/apt/sources.list.d/caddy.list

      - name: Install
        run: |
          set -e
          apt update
          apt install caddy

      - name: Start and enable service
        run: systemctl enable --now caddy

  update-caddy:
    env:
      DOMAIN:
    actions:
      - name: Update Caddyfile
        template:
          src: tpl/Caddyfile
          dst: /etc/caddy/Caddyfile

      - name: Reload config
        run: systemctl reload caddy

  build:
    local: true
    env:
      TAG:
    artifacts:
      bin:
        path: build/app
    actions:
    - name: Build
      run: |
        set -e
        # GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o build/app
        #
        mkdir -p build/
        cp tpl/sample build/app

  deploy:
    env:
      CONFIG:
      TAG:
    actions:
    - name: Prepare dirs
      run: "mkdir -p /app/config/${CONFIG}"

    - name: Prepare release
      run: |
        mkdir -p /app/releases/${TAG}
        ln -sfn /app/config/${CONFIG}/.env /app/releases/${TAG}/.env

    - name: Copy artifact
      copy:
        artifact: bin
        dst: /app/releases/${TAG}/app
        mode: 0755

    - name: Release
      run: |
        ln -sfn /app/releases/${TAG} /app/current

    # - name: Restart
    # run: systemctl restart app
`

var plansTemplate = `plans:
  bootstrap:
    steps:
    - job: config
      targets: [workers]

    - job: install-caddy
      targets: [workers]

  deploy:
    env:
      TAG: v0.0.1
    steps:
    - job: build
      targets: [workers]

    - job: update-caddy
      targets: [workers]
      env:
        DOMAIN: beta.example.tld

    - job: deploy
      targets: [workers]
      env:
        CONFIG: v1
`

