# Contributing to Code Reviewer AI Agent

Thank you for your interest in contributing to **Code Reviewer AI Agent**! We welcome bug reports, feature requests, pull requests, and documentation improvements.

---

## Development Prerequisites

- **Go**: Version 1.25 or higher.
- **Git**: Modern version.
- **Make**: Standard build automation tool.
- **Docker & Docker Compose**: Optional for container build verification.

---

## Local Development Workflow

### 1. Clone & Build

```bash
git clone https://github.com/divmora/code-reviewer-ai-agent.git
cd code-reviewer-ai-agent

# Compile binary into bin/code-reviewer
make build
```

### 2. Running Unit Tests

All unit tests run locally with the race detector enabled:

```bash
# Run all unit tests with race detection
make test

# Run a specific package test suite
go test -v -race ./pkg/ast/...
go test -v -race ./pkg/reviewer/...
go test -v -race ./pkg/provider/...
```

### 3. Local Workspace Testing

You can run the agent locally against your current repository diff:

```bash
# Review uncommitted changes in current directory
./bin/code-reviewer --workspace . --diff

# Review uncommitted changes formatted as JSON
./bin/code-reviewer --workspace . --diff --format json
```

### 4. Code Formatting & Linting

```bash
# Format code
go fmt ./...

# Run static analysis
go vet ./...
```

---

## Conventional Commits

We adhere to the [Conventional Commits](https://www.conventionalcommits.org/) specification for automated release management via Release Please:

```
<type>(<scope>): <short summary>

[optional body]

[optional footer(s)]
```

### Common Types:
- `feat`: New user-facing reviewer capability, provider, or AST parser.
- `fix`: Bug fix in diff parsing, suggestion formatting, or provider client.
- `docs`: Documentation portal and README updates.
- `chore`: Dependency bumps, build adjustments, or internal refactoring.
- `test`: Adding or refactoring test suites.
- `ci`: Changes to GitHub Actions workflows, Dockerfile, or GoReleaser.

---

## Pull Request Guidelines

1. **Branch Naming**: Use descriptive branch names like `feat/bitbucket-server` or `fix/ast-slice-recursion`.
2. **Test Coverage**: Ensure all new features or bug fixes are accompanied by unit tests.
3. **Clean Build**: Ensure `go test -v -race ./...` and `make build` pass with 0 errors.
4. **Documentation**: Update `README.md`, `AGENTS.md`, and `docs/index.html` when modifying flags or schemas.
5. **Licensing**: All contributions to this project are submitted under the **Apache License, Version 2.0**.
