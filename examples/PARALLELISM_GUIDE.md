# Hades Parallelism & Rollout Guide

Control how Hades executes jobs across multiple hosts with parallelism and rollout strategies.

## Parallelism Modes

Hades supports three parallelism modes via the `parallelism` field on steps:

### 1. Serial Execution (One at a Time)

```yaml
steps:
  - name: careful-deploy
    job: deploy
    targets: [app-servers]
    parallelism: "1"
```

Executes on one host at a time. Use for:
- High-risk deployments
- Database migrations
- Leader elections
- Any operation requiring sequential execution

### 2. Fixed Parallelism

```yaml
steps:
  - name: batch-deploy
    job: deploy
    targets: [app-servers]
    parallelism: "5"
```

Executes on 5 hosts at a time. Hosts are batched automatically. Use for:
- Controlled rollouts
- Rate-limiting deployments
- Managing resource constraints

### 3. Percentage-Based Parallelism

```yaml
steps:
  - name: gradual-rollout
    job: deploy
    targets: [app-servers]
    parallelism: "40%"
```

Executes on 40% of hosts at a time. Scales with fleet size. Use for:
- Large fleets where absolute numbers vary
- Gradual rollouts
- Maintaining availability during deployments

### 4. Full Parallel (Default)

```yaml
steps:
  - name: fast-deploy
    job: deploy
    targets: [app-servers]
    # No parallelism specified = all hosts in parallel
```

Executes on all hosts simultaneously. Use for:
- Configuration changes
- Fast deployments
- Read-only operations

## Canary Deployments

Use `limit` to restrict execution to a subset of hosts:

```yaml
steps:
  - name: canary
    job: deploy
    targets: [app-servers]
    limit: 1              # Only first host
    parallelism: "1"

  - name: verify-canary
    job: health-check
    targets: [app-servers]
    limit: 1

  - name: rollout
    job: deploy
    targets: [app-servers]
    parallelism: "3"      # 3 hosts at a time
```

The `limit` field:
- Selects the first N hosts from the target group
- Applied **before** parallelism calculation
- Deterministic ordering (same hosts each time)

## Batching Behavior

When parallelism < host count, Hades creates batches:

**Example**: 10 hosts, parallelism "3"
- Batch 1: 3 hosts (parallel)
- Batch 2: 3 hosts (parallel)
- Batch 3: 3 hosts (parallel)
- Batch 4: 1 host

Batches execute **sequentially**. Within a batch, hosts execute **in parallel**.

## Failure Handling

**Abort on First Failure**: If any host fails, the entire batch (and plan) aborts immediately.

```yaml
steps:
  - name: deploy-batch-1
    job: deploy
    targets: [app-servers]
    parallelism: "5"
    # If host 2 fails, remaining 3 in batch continue
    # But next batch won't start
```

This ensures:
- Fast failure detection
- No cascading failures
- Predictable state (you know exactly which hosts succeeded)

## Common Patterns

### Pattern 1: Progressive Rollout

Start small, gradually increase parallelism:

```yaml
steps:
  - name: canary
    job: deploy
    targets: [production]
    limit: 1

  - name: small-batch
    job: deploy
    targets: [production]
    limit: 10
    parallelism: "5"

  - name: full-rollout
    job: deploy
    targets: [production]
    parallelism: "20%"
```

### Pattern 2: Blue-Green Deployment

Deploy to inactive pool, then switch:

```yaml
steps:
  - name: deploy-green
    job: deploy
    targets: [green-pool]
    parallelism: "5"

  - name: verify-green
    job: health-check
    targets: [green-pool]

  - name: switch-traffic
    job: update-load-balancer
    targets: [lb-servers]

  - name: drain-blue
    job: drain
    targets: [blue-pool]
    parallelism: "1"
```

### Pattern 3: Rolling Restart

Restart services with availability:

```yaml
steps:
  - name: rolling-restart
    job: restart-service
    targets: [app-servers]
    parallelism: "1"      # One at a time
    # OR parallelism: "25%"  # 25% at a time
```

### Pattern 4: Multi-Region Deployment

Deploy region by region:

```yaml
steps:
  - name: deploy-us-east
    job: deploy
    targets: [us-east-servers]
    parallelism: "50%"

  - name: verify-us-east
    job: health-check
    targets: [us-east-servers]

  - name: deploy-us-west
    job: deploy
    targets: [us-west-servers]
    parallelism: "50%"

  - name: deploy-eu
    job: deploy
    targets: [eu-servers]
    parallelism: "50%"
```

## Interactive Gates

Combine parallelism with `wait` actions for manual approval:

```yaml
steps:
  - name: canary
    job: deploy
    targets: [production]
    limit: 1

  - name: verify-canary
    job: health-check
    targets: [production]
    limit: 1

  - name: approval-gate
    job: confirm
    targets: [production]
    limit: 1

  - name: rollout
    job: deploy
    targets: [production]
    parallelism: "10"

jobs:
  confirm:
    actions:
      - wait:
          message: "Canary healthy. Proceed with rollout?"
          timeout: "5m"
```

## Performance Considerations

**Higher Parallelism**:
- Faster deployments
- More load on SSH connections
- Higher risk if changes are bad

**Lower Parallelism**:
- Slower deployments
- Less resource usage
- Safer rollouts
- Easier to monitor

**Recommendations**:
- Development: High parallelism (fast iteration)
- Staging: Medium parallelism (balance)
- Production: Low parallelism (safety first)

## Examples by Fleet Size

**Small fleet (< 10 hosts)**:
```yaml
parallelism: "1"      # Serial, watch each host
```

**Medium fleet (10-50 hosts)**:
```yaml
parallelism: "5"      # 5 at a time
# OR
parallelism: "20%"    # 20% at a time
```

**Large fleet (50-500 hosts)**:
```yaml
parallelism: "10%"    # Percentage scales with fleet
```

**Very large fleet (> 500 hosts)**:
```yaml
parallelism: "5%"     # Small percentage = many hosts
```

## Monitoring During Rollouts

Hades displays:
- Batch progress (Batch 1/5)
- Per-host status
- Failures immediately

Watch for:
- First batch completion (quick feedback)
- Consistent failures across hosts
- Performance degradation

## Troubleshooting

**Too slow**: Increase parallelism
```yaml
# Before: parallelism: "1"
# After:  parallelism: "5"
```

**Too fast, overwhelming systems**: Decrease parallelism
```yaml
# Before: parallelism: "50%"
# After:  parallelism: "10%"
```

**Need to restart from partial failure**: Hades is idempotent - re-run the same plan. Consider using `limit` to skip already-successful hosts if your job supports it.

## Best Practices

1. **Start Conservative**: Begin with low parallelism, increase as confidence grows
2. **Test in Staging**: Validate parallelism settings in non-production first
3. **Use Canaries**: Always deploy to 1 host first for high-risk changes
4. **Add Gates**: Use `wait` actions between stages for manual verification
5. **Monitor Metrics**: Watch application metrics during rollouts
6. **Keep History**: Log deployments to track what parallelism worked best
