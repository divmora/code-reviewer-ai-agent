# AGENTS.md — Code Reviewer AI Agent

[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/divmora/code-reviewer-ai-agent)
[![Documentation](https://img.shields.io/badge/docs-GitHub%20Pages-blue.svg)](https://divmora.github.io/code-reviewer-ai-agent/)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

This document describes the design, architecture, and operational contract of the **Code Reviewer AI Agent**, built with the **[Divmora LocalHarness SDK](https://github.com/divmora/localharness)**.

---

## 1. Agent Overview

The **Code Reviewer Agent** is a specialized, autonomous AI reviewer that analyzes pull requests, merge requests, and local git changes across **GitLab (SaaS & Multi-instance Self-Hosted)**, **GitHub**, **Bitbucket**, and local workspaces.

It emulates the thoroughness and precision of CodeRabbit, producing structured findings, quality scorecards, file walkthroughs, witty summary poems, and 1-click committable inline suggestions directly in target VCS platforms.

```
┌────────────────────────────────────────────────────────┐
│               Zenith UI / CLI Ingress                  │
│       (--url, --prompt, --rules-json, --token)         │
└──────────────────────────┬─────────────────────────────┘
                           │
                           ▼
┌────────────────────────────────────────────────────────┐
│           1. Ingress & Consistency Engine              │
│       • Smart URL parser (GitLab / GitHub / BB)        │
│       • Already-reviewed watermark check               │
│       • Stale diff exponential retry poller            │
└──────────────────────────┬─────────────────────────────┘
                           │
                           ▼
┌────────────────────────────────────────────────────────┐
│        2. Noise Filtering & AST Scope Slicing          │
│       • Ignores lockfiles, minified & binary files     │
│       • Pre-scans for leaked secrets & API keys        │
│       • Slices enclosing functions/classes with AST    │
│       • Decorates source lines with line numbers       │
└──────────────────────────┬─────────────────────────────┘
                           │
                           ▼
┌────────────────────────────────────────────────────────┐
│     3. Divmora LocalHarness SDK (adk.LocalAgentConfig) │
│       • TokenGuard (120k tokens) & PatchToolArgs       │
│       • Strict Anti-Hallucination review prompt        │
│       • Balanced / Assertive / Chill sensitivity       │
│       • Batch execution with context protection        │
└──────────────────────────┬─────────────────────────────┘
                           │
                           ▼
┌────────────────────────────────────────────────────────┐
│             4. VCS Publishing & Actions                │
│       • 1-Click inline suggestion threads              │
│       • Quality Scorecard & Walkthrough in description │
│       • Scoped score labels (Quality/Security/Maint)   │
│       • Commit status (success / failed)               │
│       • Auto-approval on clean PRs                     │
└────────────────────────────────────────────────────────┘
```

---

## 2. LocalHarness SDK Integration (`pkg/agent`)

The agent is instantiated using `adk.NewLocalAgentConfig()`:

- **Binary Resolution**: Leverages `connection.BinaryResolver{Version: "2.0.0"}` to auto-detect and resolve the localharness binary.
- **Middleware Pipeline**:
  - `middleware.NewTokenGuard(120000, 0.8, logger)`: Enforces token budgets and prevents context exhaustion.
  - `middleware.NewPatchToolArgs(logger)`: Normalizes tool arguments.
- **Safety Policy**:
  - `policy.AllowAll()` with `Capabilities.RunCommand = false` for read-only static analysis safety.

### 2.1 Model & LiteLLM Resolution Priority Flow

```
┌────────────────────────────────────────────────────────┐
│ 1. CLI Flags (Highest Priority)                        │
│    --litellm-base-url / --litellm-api-key              │
│    --litellm-model    / --litellm-endpoint             │
└───────────────────────────┬────────────────────────────┘
                            │ (if omitted)
                            ▼
┌────────────────────────────────────────────────────────┐
│ 2. Tool-Specific Environment Variables                 │
│    CODE_REVIEWER_LITELLM_BASE_URL                      │
│    CODE_REVIEWER_LITELLM_API_KEY                       │
│    CODE_REVIEWER_LITELLM_MODEL                         │
│    CODE_REVIEWER_LITELLM_ENDPOINT                      │
└───────────────────────────┬────────────────────────────┘
                            │ (if omitted)
                            ▼
┌────────────────────────────────────────────────────────┐
│ 3. Generic LiteLLM Environment Variables               │
│    LITELLM_BASE_URL / LITELLM_API_KEY                  │
│    LITELLM_MODEL    / LITELLM_ENDPOINT                 │
└───────────────────────────┬────────────────────────────┘
                            │ (if omitted)
                            ▼
┌────────────────────────────────────────────────────────┐
│ 4. Default / Named Endpoint in litellm.json            │
│    Resolved from ~/.divmora/config/litellm.json        │
└────────────────────────────────────────────────────────┘
```

---

## 3. Core Subsystems

### 3.1 AST Scope Slicing (`pkg/ast`)
- **Problem**: Sending entire files for multi-thousand line repositories breaches model context windows and causes hallucinations on legacy code.
- **Solution**: The AST Scope Slicer parses the syntax tree (native Go parser `go/ast` + generic multi-language scope analyzer for Python, TypeScript, PHP, Java, Rust) and extracts only the enclosing function/method/class definitions around changed lines. Untouched code blocks are replaced with `# ... skipped N lines ...`.

### 3.2 Anti-Hallucination Directives (`pkg/agent/prompt.go`)
- **Only Comment on Changed Lines**: The model is strictly instructed to evaluate only lines modified by the author.
- **Context is Read-Only**: Enclosing context lines are marked strictly as reference for type checks and variable scopes, eliminating complaints about untouched legacy code.

### 3.3 Noise Filtering & Comment Budgeting (`pkg/reviewer`)
- **Filter**: Automatically excludes lockfiles (`package-lock.json`, `go.sum`), minified assets (`*.min.js`), and generated code (`*.pb.go`).
- **Secrets Pre-Scanner**: Flags hardcoded credentials (AWS keys, OpenAI keys, private keys) with immediate High-severity security findings.
- **Comment Budget**: Caps inline comments to the top 10–15 findings (`--max-comments 15`, prioritizing High > Medium > Low) to prevent "Bot Review Fatigue".

### 3.4 Multi-VCS & Duplicate Prevention (`pkg/provider`)
- **Smart URL Parser**: Automatically identifies provider and extracts project paths including multi-level subgroups (e.g. `ifmists/pension/pensions`).
- **Duplicate Prevention (`dedup.go`)**: Checks for `<!-- review_sha: <sha> -->` in the MR description and skips already-reviewed commits unless `--force` is specified.
- **Label Updater (`labels.go`)**: Manages scoped score labels (`Quality Score::XX`, `Security Score::XX`, `Maintainability Score::XX`) with dynamic color assignment (Green $\ge 80$, Orange $60-79$, Red $<60$).
- **1-Click Suggestions**: Formats code replacements in GitLab ```` ```suggestion:-0+0 ```` and GitHub ```` ```suggestion ```` blocks.

---

## 4. Zenith Integration Contract

The standalone agent exposes a clean CLI and programmatic interface for orchestration by **Zenith** (the unified AI platform):

### CLI Ingestion Protocol:
- Custom Prompt: `--prompt "Project specific instructions"` or `ZENITH_CUSTOM_PROMPT`
- Structured Rules: `--rules-json '{"rules":[{"name":"Strict Auth","description":"..."}]}'`
- Rules File: `--rules path/to/.coderabbit.yaml`
- Output Format: `--format json` or `--format markdown`

### Programmatic SDK Interface:
```go
rev := reviewer.NewReviewer(logger)
report, err := rev.RunReview(ctx, provider, targetContext, files, reviewer.ReviewOptions{
    CustomPrompt: zenithCustomPrompt,
    RulesJSON:    zenithRulesJSON,
    Profile:      "balanced",
    MaxComments:  15,
})
```

---

## 5. Development & Verification
 
```bash
make fmt                  # Format Go code with gofmt
make lint                 # Run linter and static analysis
make test                 # Run unit tests with race detector
make test-coverage        # Run tests with HTML coverage output
make build                # Build binary to bin/code-reviewer
make docker-build         # Build local Docker image
make docker-build-multiarch # Build multi-arch Docker image (amd64/arm64)
```

---

## 6. Ecosystem Best Practices & Guidelines

### Coding Conventions
- **Structured Logging**: Use standard library `log/slog` for all log outputs with structured contextual attributes.
- **Error Handling**: Wrap errors using `fmt.Errorf("...: %w", err)` for clean error unwrapping.
- **Context Propagation**: Always propagate `context.Context` through AST parsing, LocalHarness client calls, and VCS provider requests.
- **Dry-Run & Safety**: Never post comments or update MR descriptions unless `--post` is explicitly supplied.

### Living Product Roadmap Management
`ROADMAP.md` is the central living document tracking future capabilities, optimizations, and technical debt:
- **Adding Items**: Whenever you or the user identify a capability, optimization, or edge-case improvement for future work, add it to `ROADMAP.md` under the appropriate category.
- **Removing Items**: Once a feature is fully implemented, verified with tests, and committed, **remove it from `ROADMAP.md`** immediately to keep active focus on upcoming tasks.

### Conventional Commits & Release Automation
- Adhere strictly to the [Conventional Commits](https://www.conventionalcommits.org/) specification (`feat:`, `fix:`, `docs:`, `chore:`, `test:`, `ci:`, `feat!:`) for automated release tagging and changelog generation via Google Release Please.
- Version numbers are maintained in `.release-please-manifest.json` and injected into `pkg/version` at build time via `-ldflags`.

---

## 7. Community & Agent Collaboration

We welcome open-source contributions from both developers and autonomous AI agents. When submitting pull requests:
- Ensure all new AST parsers or VCS providers include comprehensive unit tests.
- Confirm full compliance by running `make fmt`, `make lint`, and `make test`.
- All contributions are licensed under the **Apache License, Version 2.0**.
