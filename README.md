# Hades

> Provisioning creates machines. Hades gives them a soul.

Hades is a **change-execution tool** for servers you fully own. It's designed for explicit, predictable deployments and operations.

## Quick Start

```bash
# Build Hades
make build

# Run a simple plan
hades run deploy-app \
  -f hadesfile.yaml \
  -i inventory.yaml \
  -e VERSION=v1.0.0

# Dry-run first
hades run deploy-app --dry-run
```

## What is Hades?

Hades executes **explicit change** on your infrastructure:
- **Not a provisioning tool** (use Terraform for that)
- **Not a desired-state reconciler** (no hidden reconciliation loops)
- **Not a long-running controller** (ephemeral runs only)

Hades **is**:
- **An execution engine** - runs exactly what you tell it
- **A deployment orchestrator** - canaries, rollouts, parallelism
- **A bootstrap/config tool** - setup, configuration, lifecycle
- **Human-first** - predictable, reviewable, copy-pasteable

## Core Principles

1. **Explicit over Implicit** - No magic, no hidden behavior
2. **Predictable** - Same input = same output, always
3. **Reviewable** - Dry-run shows exact commands
4. **Fail Fast** - Errors abort immediately
5. **Zero State** - Runs are ephemeral, no state stored

## Features

### ✅ 7 Action Types

- `run` - Execute shell commands
- `copy` - Copy files/artifacts to hosts
- `template` - Render Go templates
- `mkdir` - Create directories
- `push` - Push artifacts to registry
- `pull` - Pull artifacts from registry
- `wait` - Interactive gates

### ✅ Artifact Management

- **Ephemeral artifacts** - scoped to run
- **Registry storage** - filesystem or S3
- **Immutable** - published artifacts never change
- **Checksum verified** - SHA256 integrity

### ✅ Rollout Strategies

- **Serial execution** - one host at a time
- **Fixed parallelism** - N hosts at a time
- **Percentage-based** - 40% of fleet
- **Canary deployments** - limit to first N hosts
- **Abort on failure** - immediate stop

### ✅ Environment Variables

- **Strict contracts** - required vs optional
- **Priority merging** - CLI > step > defaults
- **Built-in vars** - HADES_RUN_ID, HADES_HOST_*, etc.
- **Validation** - unknown vars fail fast
- **Protection** - HADES_* cannot be overridden

### ✅ SSH Execution

- **Connection pooling** - reuse per run
- **Streaming output** - real-time logs
- **Atomic copies** - tmp + mv pattern
- **Per-host isolation** - parallel safe

## Architecture

```
hadesfile.yaml    # Jobs, plans, registries
  ↓
Loader            # Parse, validate, expand
  ↓
Executor          # Orchestrate execution
  ↓
Actions           # run, copy, template, ...
  ↓
SSH Client        # Execute on remote hosts
```

## File Structure

```yaml
registries:
  prod:
    type: filesystem
    path: /var/hades/registry

jobs:
  deploy:
    env:
      VERSION:              # Required
      MODE:
        default: production # Optional
    actions:
      - pull:
          registry: prod
          name: myapp
          tag: ${VERSION}
          to: /opt/app/app
      - run: systemctl restart myapp

plans:
  production-deploy:
    steps:
      - name: canary
        job: deploy
        targets: [servers]
        limit: 1
        env:
          VERSION: v1.0.0

      - name: rollout
        job: deploy
        targets: [servers]
        parallelism: "40%"
        env:
          VERSION: v1.0.0
```

## Installation

### From Source

```bash
git clone https://github.com/SoftKiwiGames/hades
cd hades
make build
```

Binary will be at `build/hades`.

### Requirements

- Go 1.21+
- SSH access to target hosts
- SSH keys configured

## Usage

### Basic Deployment

```bash
hades run deploy \
  -f hadesfile.yaml \
  -i inventory.yaml \
  -e VERSION=v1.2.3
```

### Dry-Run

```bash
hades run deploy --dry-run
```

Shows exactly what will execute without running.

### Canary Deployment

```yaml
plans:
  safe-deploy:
    steps:
      - name: canary
        job: deploy
        targets: [production]
        limit: 1              # Just first host

      - name: confirm
        job: wait-approval
        targets: [production]
        limit: 1

      - name: rollout
        job: deploy
        targets: [production]
        parallelism: "3"      # 3 at a time
```

### Multi-Region

```yaml
plans:
  global-deploy:
    steps:
      - name: us-east
        job: deploy
        targets: [us-east-servers]
        env:
          REGION: us-east-1

      - name: us-west
        job: deploy
        targets: [us-west-servers]
        env:
          REGION: us-west-2

      - name: eu
        job: deploy
        targets: [eu-servers]
        env:
          REGION: eu-west-1
```

## Documentation

- **[Getting Started](examples/GETTING_STARTED.md)** - First steps with Hades
- **[Environment Variables](examples/ENV_GUIDE.md)** - Variable contracts & merging
- **[Parallelism & Rollouts](examples/PARALLELISM_GUIDE.md)** - Deployment strategies
- **[Registry Guide](examples/REGISTRY_GUIDE.md)** - Artifact management
- **[PRD](docs/PRD.md)** - Product requirements & design

## Examples

See `examples/` directory:
- `simple-hadesfile.yaml` - Basic job definitions
- `complete-example-hadesfile.yaml` - Production patterns
- `parallelism-hadesfile.yaml` - Rollout strategies
- `registry-hadesfile.yaml` - Artifact workflows

## Testing

```bash
# Run all tests
make test

# Run specific package
go test ./hades/rollout -v

# Integration tests
go test ./... -v
```

## Why Hades?

**vs Ansible/Salt** - Hades doesn't reconcile state. It executes changes explicitly.

**vs Terraform** - Hades doesn't manage cloud resources. It operates on existing servers.

**vs Kubernetes** - Hades is for bare metal / VMs, not containers.

Hades is optimized for:
- Binary deployments
- Config management
- Cert rotation
- Incident response
- Database migrations
- Service restarts

## Philosophy

### Explicit Intent

```yaml
# Hades: Explicit
- run: systemctl restart myapp

# Others: Implicit
- service:
    name: myapp
    state: restarted  # How? When? What changed?
```

### No Hidden State

Hades never stores state. Every run is:
- Ephemeral
- Isolated
- Reproducible
- Auditable

### Copy-Pasteable

Dry-run output is exact:

```bash
$ hades run deploy --dry-run
...
[host-1]
  - run: systemctl restart myapp
  - run: curl http://localhost/health
```

You can copy these commands and run them manually.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

MIT License - See [LICENSE](LICENSE) for details.

## Status

**Version**: 1.0.0
**Status**: Production Ready
**Test Coverage**: Comprehensive

All 7 action types implemented and tested.
All PRD requirements satisfied.

## Community

- **Issues**: [GitHub Issues](https://github.com/SoftKiwiGames/hades/issues)
- **Discussions**: [GitHub Discussions](https://github.com/SoftKiwiGames/hades/discussions)

## Acknowledgments

Hades is inspired by the principle: **"Execute intent, don't infer it."**

Built with clarity, simplicity, and predictability as core values.
