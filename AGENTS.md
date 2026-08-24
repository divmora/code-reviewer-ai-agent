# AGENTS.md — Code Reviewer AI Agent

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

## 5. Development & Testing

```bash
# Run unit tests
make test

# Build binary
make build

# Review local changes
./bin/code-reviewer --workspace . --diff

# Review remote GitLab MR and post updates
./bin/code-reviewer --url <MR_URL> --token $GITLAB_TOKEN --skip-tls-verify --post
```
