# Getting Started with Hades

This guide will walk you through your first Hades deployment in 10 minutes.

## Prerequisites

- SSH access to at least one server
- SSH key authentication configured
- Go 1.21+ (for building from source)

## Step 1: Install Hades

```bash
# Clone repository
git clone https://github.com/SoftKiwiGames/hades
cd hades

# Build
make build

# Verify installation
./build/hades --version
# Output: hades version 1.0.0
```

## Step 2: Create Your First Hadesfile

Create `hadesfile.yaml`:

```yaml
jobs:
  hello:
    actions:
      - run: echo "Hello from Hades!"
      - run: hostname
      - run: uptime

plans:
  greet:
    steps:
      - name: say-hello
        job: hello
        targets: [my-servers]
```

This defines:
- **Job `hello`**: Three simple commands
- **Plan `greet`**: Executes `hello` job on `my-servers` target

## Step 3: Create Inventory

Create `inventory.yaml`:

```yaml
hosts:
  - name: server-1
    addr: 192.168.1.10
    user: deploy
    key: /home/user/.ssh/id_rsa

targets:
  my-servers:
    - server-1
```

Update with your actual:
- `addr`: Server IP or hostname
- `user`: SSH user
- `key`: Path to SSH private key

## Step 4: Test with Dry-Run

```bash
./build/hades run greet \
  -f hadesfile.yaml \
  -i inventory.yaml \
  --dry-run
```

Output:
```
===================
DRY-RUN: greet
===================

This will execute the following actions:
Step 1: say-hello
  Job: hello

  [server-1]
    - run: echo "Hello from Hades!"
    - run: hostname
    - run: uptime
```

Dry-run shows **exactly** what will execute. No surprises!

## Step 5: Execute the Plan

Remove `--dry-run` to actually execute:

```bash
./build/hades run greet \
  -f hadesfile.yaml \
  -i inventory.yaml
```

Output:
```
============
Plan: greet
============

Run ID: hades-20240101-120000
Started: 2024-01-01T12:00:00Z

Step 1/1: say-hello
  Job: hello
  Hosts: 1

[server-1] Executing job...
Hello from Hades!
server-1
 12:00:00 up 30 days, 5:23, 1 user, load average: 0.00, 0.00, 0.00
[server-1] ✓ Job completed

✓ Step completed: say-hello

✓ Plan completed successfully
Duration: 2.3s
```

## Step 6: Add Environment Variables

Let's make it more dynamic. Update `hadesfile.yaml`:

```yaml
jobs:
  deploy:
    env:
      VERSION:              # Required
      MODE:
        default: production # Optional with default
    actions:
      - run: echo "Deploying version ${VERSION} in ${MODE} mode"
      - run: mkdir -p /opt/myapp/${VERSION}

plans:
  deploy-app:
    steps:
      - name: deploy
        job: deploy
        targets: [my-servers]
        env:
          VERSION: v1.0.0
```

Run with different versions:

```bash
# Use plan-defined version
./build/hades run deploy-app -f hadesfile.yaml -i inventory.yaml

# Override via CLI
./build/hades run deploy-app \
  -f hadesfile.yaml \
  -i inventory.yaml \
  -e VERSION=v2.0.0 \
  -e MODE=staging
```

## Step 7: Copy Files to Server

Add file copy action:

```yaml
jobs:
  setup:
    actions:
      - mkdir:
          path: /opt/myapp
          mode: 0755
      - copy:
          src: ./config/app.conf
          dst: /opt/myapp/app.conf
      - run: cat /opt/myapp/app.conf

plans:
  setup-app:
    steps:
      - name: setup
        job: setup
        targets: [my-servers]
```

Create `config/app.conf`:
```
debug=false
port=8080
```

Run:
```bash
./build/hades run setup-app -f hadesfile.yaml -i inventory.yaml
```

## Step 8: Add Parallelism

Deploy to multiple servers with controlled rollout:

```yaml
# inventory.yaml - add more servers
hosts:
  - name: server-1
    addr: 192.168.1.10
    user: deploy
    key: ~/.ssh/id_rsa

  - name: server-2
    addr: 192.168.1.11
    user: deploy
    key: ~/.ssh/id_rsa

  - name: server-3
    addr: 192.168.1.12
    user: deploy
    key: ~/.ssh/id_rsa

targets:
  my-servers:
    - server-1
    - server-2
    - server-3
```

Update plan:

```yaml
plans:
  safe-deploy:
    steps:
      # Canary: Deploy to one server first
      - name: canary
        job: deploy
        targets: [my-servers]
        limit: 1
        env:
          VERSION: v1.0.0

      # Rollout: Deploy to rest, 2 at a time
      - name: rollout
        job: deploy
        targets: [my-servers]
        parallelism: "2"
        env:
          VERSION: v1.0.0
```

Run:
```bash
./build/hades run safe-deploy -f hadesfile.yaml -i inventory.yaml
```

Output shows batched execution:
```
Step 1/2: canary
  Hosts: 1
  Limited to: 1 hosts

[server-1] Executing job...
[server-1] ✓ Job completed

Step 2/2: rollout
  Hosts: 3
  Batches: 2 (parallelism: 2)

Batch 1/2 (2 hosts)
[server-1] Executing job...
[server-2] Executing job...
[server-1] ✓ Job completed
[server-2] ✓ Job completed

Batch 2/2 (1 hosts)
[server-3] Executing job...
[server-3] ✓ Job completed
```

## Step 9: Add Interactive Gate

Add approval between canary and rollout:

```yaml
jobs:
  confirm:
    actions:
      - wait:
          message: "Canary successful. Continue with rollout?"
          timeout: "5m"

plans:
  gated-deploy:
    steps:
      - name: canary
        job: deploy
        targets: [my-servers]
        limit: 1
        env:
          VERSION: v1.0.0

      - name: approval
        job: confirm
        targets: [my-servers]
        limit: 1

      - name: rollout
        job: deploy
        targets: [my-servers]
        parallelism: "2"
        env:
          VERSION: v1.0.0
```

Run:
```bash
./build/hades run gated-deploy -f hadesfile.yaml -i inventory.yaml
```

Hades will pause and ask:
```
⏸️  Canary successful. Continue with rollout? [y/N]:
```

Type `y` to continue, `n` to abort.

## Step 10: Use Templates

Create `templates/nginx.conf.tmpl`:
```nginx
server {
    listen 80;
    server_name {{ .Host }};

    location / {
        proxy_pass http://localhost:{{ .Env.APP_PORT }};
    }
}
```

Add template action:

```yaml
jobs:
  configure:
    env:
      APP_PORT:
        default: "8080"
    actions:
      - template:
          src: templates/nginx.conf.tmpl
          dst: /etc/nginx/sites-available/myapp
      - run: ln -sf /etc/nginx/sites-available/myapp /etc/nginx/sites-enabled/
      - run: nginx -t
      - run: systemctl reload nginx
```

Template context includes:
- `.Env` - All environment variables
- `.Host` - Current host name
- `.Target` - Current target group

## Next Steps

### Learn More

- **[Environment Variables](ENV_GUIDE.md)** - Required vs optional, validation
- **[Parallelism](PARALLELISM_GUIDE.md)** - Rollout strategies, batching
- **[Registry Guide](REGISTRY_GUIDE.md)** - Artifact management
- **[Examples](.)** - More complex scenarios

### Common Patterns

**Blue-Green Deployment**:
```yaml
steps:
  - name: deploy-green
    job: deploy
    targets: [green-pool]

  - name: switch-traffic
    job: update-lb
    targets: [lb-servers]

  - name: drain-blue
    job: drain
    targets: [blue-pool]
```

**Multi-Region**:
```yaml
steps:
  - name: us-east
    job: deploy
    targets: [us-east-servers]
    env:
      REGION: us-east-1

  - name: eu-west
    job: deploy
    targets: [eu-west-servers]
    env:
      REGION: eu-west-1
```

**Rolling Restart**:
```yaml
steps:
  - name: restart
    job: restart-service
    targets: [servers]
    parallelism: "1"  # One at a time
```

## Troubleshooting

### SSH Connection Failed

**Error**: `failed to connect to host`

**Fix**: Check SSH key permissions:
```bash
chmod 600 ~/.ssh/id_rsa
```

Verify SSH access manually:
```bash
ssh -i ~/.ssh/id_rsa user@host
```

### Missing Required Variable

**Error**: `required environment variable "VERSION" not provided`

**Fix**: Provide via step env or CLI:
```bash
hades run deploy -e VERSION=v1.0.0
```

### Unknown Variable

**Error**: `unknown environment variable "VERSOIN"`

**Fix**: Check for typos in variable names. All variables must be defined in job's `env` contract.

### Dry-Run Shows Wrong Values

**Issue**: `${VERSION}` not expanded in dry-run

**Cause**: Variable not provided or incorrectly formatted.

**Fix**: Ensure variables use `${VAR}` syntax and are provided:
```bash
hades run deploy -e VERSION=v1.0.0 --dry-run
```

## Tips

1. **Always dry-run first**: Catches issues before execution
2. **Start with one host**: Test on single server before fleet
3. **Use canaries**: Deploy to 1 host, verify, then rollout
4. **Version everything**: Pass VERSION via CLI from CI/CD
5. **Keep jobs simple**: One job = one purpose
6. **Use targets**: Group servers logically (web, db, cache)

## Getting Help

- **Documentation**: Check `examples/` directory
- **Issues**: [GitHub Issues](https://github.com/SoftKiwiGames/hades/issues)
- **Discussions**: [GitHub Discussions](https://github.com/SoftKiwiGames/hades/discussions)

## What's Next?

Now that you understand the basics, try:
- Deploying a real application
- Setting up CI/CD integration
- Creating multi-stage rollouts
- Using artifact registries
- Building reusable job libraries

Happy deploying with Hades! 🚀
