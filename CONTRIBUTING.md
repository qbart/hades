# Contributing to Hades

Thank you for your interest in contributing to Hades! This document provides guidelines for contributing.

## Code of Conduct

- Be respectful and inclusive
- Focus on constructive feedback
- Help others learn and grow
- Assume good intent

## How to Contribute

### Reporting Bugs

1. Check if the bug is already reported in [Issues](https://github.com/SoftKiwiGames/hades/issues)
2. If not, create a new issue with:
   - Clear title describing the problem
   - Steps to reproduce
   - Expected vs actual behavior
   - Hades version (`hades --version`)
   - Operating system and Go version
   - Example hadesfile if relevant

### Suggesting Features

1. Check [Issues](https://github.com/SoftKiwiGames/hades/issues) for similar requests
2. Open a new issue with:
   - Clear description of the feature
   - Use case / motivation
   - Proposed API or interface
   - Consider if it aligns with Hades philosophy (explicit, predictable, no magic)

### Pull Requests

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass: `make test`
6. Update documentation
7. Commit with clear messages
8. Push and create pull request

## Development Setup

### Prerequisites

- Go 1.21+
- Make
- SSH access to test servers (optional, for integration tests)

### Setup

```bash
# Clone repository
git clone https://github.com/SoftKiwiGames/hades
cd hades

# Install dependencies
go mod download

# Build
make build

# Run tests
make test
```

### Project Structure

```
hades/
├── main.go                 # Entry point
├── hades/
│   ├── cli.go             # CLI commands
│   ├── schema/            # YAML schema definitions
│   ├── loader/            # YAML loading & validation
│   ├── executor/          # Execution orchestration
│   ├── actions/           # Action implementations
│   ├── ssh/               # SSH client
│   ├── artifacts/         # Artifact manager
│   ├── registry/          # Registry backends
│   ├── inventory/         # Host/target resolution
│   ├── rollout/           # Rollout strategies
│   ├── types/             # Shared types
│   └── ui/                # Output formatting
├── examples/              # Example files & guides
└── docs/                  # Documentation
```

## Coding Guidelines

### Go Style

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting
- Run `go vet` for static analysis
- Keep functions small and focused
- Use meaningful variable names

### Error Handling

```go
// Good: Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to connect to host %s: %w", host, err)
}

// Bad: Generic error messages
if err != nil {
    return err
}
```

### Testing

- Write tests for all new functionality
- Use table-driven tests when applicable
- Test both success and failure cases
- Mock external dependencies (SSH, registries)

Example:
```go
func TestParseStrategy(t *testing.T) {
    tests := []struct {
        name      string
        input     string
        hostCount int
        want      int
        wantErr   bool
    }{
        {"serial", "1", 10, 1, false},
        {"percentage", "40%", 10, 4, false},
        {"invalid", "abc", 10, 0, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ParseStrategy(tt.input, tt.hostCount)
            if (err != nil) != tt.wantErr {
                t.Errorf("want err=%v, got err=%v", tt.wantErr, err)
            }
            if !tt.wantErr && got.Parallelism != tt.want {
                t.Errorf("want %d, got %d", tt.want, got.Parallelism)
            }
        })
    }
}
```

### Documentation

- Add godoc comments for exported functions
- Update README.md for user-facing changes
- Add examples for new features
- Update guides in `examples/` directory

## Hades Philosophy

When contributing, keep these principles in mind:

### 1. Explicit over Implicit

```yaml
# Good: Explicit
- run: systemctl restart myapp

# Bad: Implicit magic
- ensure_running: myapp
```

### 2. Predictable Behavior

- Same input = same output
- No hidden state
- No reconciliation loops
- Deterministic execution order

### 3. No Magic

- Every action is exactly what it says
- No auto-detection or guessing
- User explicitly defines behavior
- Dry-run shows exact commands

### 4. Fail Fast

- Validate early (before SSH)
- Abort on first failure
- Clear error messages
- No silent retries

## Adding New Action Types

If proposing a new action type:

1. Check if existing actions can accomplish the goal
2. Ensure it fits Hades' explicit model
3. Define clear semantics (what exactly does it do?)
4. Implement interface:

```go
type NewAction struct {
    Field1 string
    Field2 int
}

func (a *NewAction) Execute(ctx context.Context, runtime *types.Runtime) error {
    // Implementation
}

func (a *NewAction) DryRun(ctx context.Context, runtime *types.Runtime) string {
    return fmt.Sprintf("new-action: field1=%s field2=%d", a.Field1, a.Field2)
}
```

5. Add to schema
6. Add to executor action dispatch
7. Write tests
8. Document with examples

## Testing Guidelines

### Unit Tests

```bash
# Run all tests
make test

# Run specific package
go test ./hades/rollout -v

# Run with coverage
go test ./... -cover
```

### Integration Tests

For testing with real SSH:

```bash
# Setup test server (Docker)
docker run -d --name hades-test \
  -p 2222:22 \
  -e SSH_ENABLE_ROOT=true \
  panubo/sshd

# Run integration tests
SSH_TEST_HOST=localhost:2222 go test ./... -tags=integration
```

### Manual Testing

```bash
# Build
make build

# Test with examples
./build/hades run test -f examples/simple-hadesfile.yaml --dry-run

# Test error handling
./build/hades run invalid-plan -f examples/simple-hadesfile.yaml
```

## Commit Messages

Use conventional commit format:

```
feat: add percentage-based parallelism
fix: correct SSH connection pooling
docs: update getting started guide
test: add rollout strategy tests
refactor: simplify env validation
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation
- `test`: Tests
- `refactor`: Code refactoring
- `perf`: Performance improvement
- `chore`: Maintenance

## Review Process

1. Automated tests run on all PRs
2. Maintainer reviews code and design
3. Feedback addressed
4. Approved and merged

## Release Process

1. Update version in `version.go`
2. Update CHANGELOG.md
3. Create git tag: `git tag v1.2.0`
4. Push tag: `git push --tags`
5. GitHub Actions builds and releases

## Questions?

- Open a [Discussion](https://github.com/SoftKiwiGames/hades/discussions)
- Ask in issues
- Check existing documentation

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
