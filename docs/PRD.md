Product Requirements Document (PRD)
Project: Hades
One-liner

Provisioning creates machines.
Hades gives them a soul.

1. Purpose & Scope

Hades is a change-execution tool for servers you fully own.

It is not:

a provisioning tool

a desired-state reconciler

a long-running controller

It is:

an explicit execution engine

a deployment & rollout orchestrator

a bootstrap / configuration / lifecycle tool

a human-first ops system

Hades must always:

execute exactly what it prints

have predictable, reviewable behavior

avoid hidden state and magic

2. Non-Goals (Explicit)

Hades must not:

manage cloud resources (VMs, networks, disks)

reconcile desired state

auto-retry silently

template shell commands

store secrets in YAML

maintain long-lived agents on hosts

3. Core Concepts
3.1 Hosts & Targets

Host = concrete machine reachable via SSH

Target = named group of hosts

Inventory is external (file, CLI input, future API)

Hosts are assumed to exist.

3.2 Jobs

A job is a reusable unit of work.

Mental model:

A job is a function.
Environment variables are its arguments.

Job characteristics:

ordered list of actions

explicit env contract

optional local execution

may produce ephemeral artifacts

Jobs never:

know about registries

know about environments (prod/beta)

branch internally

3.3 Plans

A plan is an ordered execution of jobs.

Plans:

define control flow

define rollout strategy

define concurrency

define canary / promote behavior

Plans do not:

define commands

embed logic

contain shell scripting

3.4 Runs

A run is a single execution of a plan.

Run properties:

ephemeral

isolated

has a unique run ID

no persisted state

failure aborts by default

4. Execution Model
4.1 Actions (Exact Semantics)
run

Executes a shell command string

No templating

Env expansion only

Executed exactly as printed

- run: systemctl restart myservice

copy

Copies file or artifact to host

Checksum-based

Atomic (tmp + mv)

- copy:
    src: files/a.conf
    dst: /etc/a.conf


or

- copy:
    artifact: binary
    to: /usr/local/bin/app

template

Go template rendering only for files

Supports loops and conditionals

Never used for commands

- template:
    src: templates/app.conf.tmpl
    dst: /etc/app.conf


Template context:

.Env (validated envs + HADES_*)

.Host

.Target

mkdir
- mkdir:
    path: /opt/app
    mode: 0755

push

Local → registry

Explicit publishing step

- push:
    registry: prod
    artifact: binary
    name: app
    tag: ${TAG}

pull

Registry → remote host

- pull:
    registry: prod
    name: app
    tag: ${TAG}
    to: /opt/app/app

wait

Manual or timed gate

- wait:
    message: "Approve promotion?"

5. Artifacts

Artifacts are always ephemeral

Exist only during a run

Produced by jobs

Consumed by later jobs

Never persisted unless explicitly pushed

Artifacts are identified by name, not path.

6. Registries

Registries are external artifact stores.

Supported types (v1):

filesystem

s3

http (future)

Registry responsibilities:

immutable storage

versioning via name + tag or digest

rollback support

Registries are never implicit.

7. Environment Variables
7.1 Job Env Contract

All env vars are required by default

Optional only if default is specified

Unknown envs cause hard errors

env:
  BINARY:
  MODE:
    default: prod

7.2 Built-in Envs (Read-Only)

Injected automatically:

HADES_PLAN

HADES_RUN_ID

HADES_TARGET

HADES_HOST_NAME

HADES_HOST_ADDR

Users cannot define or override HADES_*.

7.3 OS Env Expansion

${VAR} expansion allowed only in plan env values

Expanded once, before execution

Missing OS env → hard error

8. Rollouts & Concurrency
8.1 Parallelism

Defined at step level:

parallelism: 2
parallelism: "40%"
parallelism: 1


Limits number of hosts executing simultaneously

Deterministic ordering

Abort on failure by default

8.2 Canary + Promote
steps:
  - name: canary
    targets: app
    job: deploy
    limit: 1

  - name: promote
    targets: app
    job: deploy
    parallelism: 2


No hidden rollout state.

9. Failure Semantics

Any action failure fails the job

Job failure fails the step

Step failure aborts the run

No implicit retries

Retries must be explicit (future)

10. UX & Trust Guarantees

Hades must guarantee:

dry-run shows exact commands

logs stream per host

no hidden behavior

copy-pasteable output

predictable failure modes

If the user cannot reason about it, it’s a bug.

11. Positioning (for context)

Terraform: creates machines

Ansible/Salt: reconcile state

Hades: execute change

Hades is optimized for:

deployments

upgrades

cert rotation

rollouts

incident response

12. Guiding Principle (Do Not Violate)

Hades executes intent, it does not infer it.

This principle overrides all convenience features.

13. Success Criteria (v1)

deterministic execution

zero hidden state

reproducible runs

explicit artifact flow

clear separation of concerns

minimal DSL surface
