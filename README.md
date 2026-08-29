# Code Reviewer AI Agent

[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/divmora/code-reviewer-ai-agent)

A context-aware, production-grade **AI Code Reviewer Agent CLI** built in Go and powered by the **[Divmora LocalHarness SDK](https://github.com/divmora/localharness)**.

Code Reviewer analyzes pull requests and local git diffs across **GitLab (Cloud & Self-Hosted)**, **GitHub**, **Bitbucket**, and local workspaces. It generates CodeRabbit-grade reviews with quality scorecards, file walkthroughs, and 1-click committable code suggestions.

---

## Features

- **Context-Aware AST Scope Slicing**: Automatically extracts enclosing functions, classes, and types around changed diff lines without overflowing context windows.
- **Strict Anti-Hallucination Guardrails**: Review comments are strictly bounded to lines modified in the diff; legacy code in surrounding context is treated strictly as read-only.
- **1-Click Committable Suggestions**: Formats code replacements in GitLab (```` ```suggestion:-0+0 ````) and GitHub (```` ```suggestion ````) syntax.
- **MR/PR Description & Scorecard Updates**: Automatically updates descriptions with a Quality/Security/Maintainability scorecard table, walkthrough, and commit SHA watermark.
- **Scoped Score Labels**: Updates GitLab scoped labels (`Quality Score::XX`, `Security Score::XX`, `Maintainability Score::XX`) with dynamic color coding (Green/Orange/Red).
- **Duplicate Review Prevention**: Inspects the description for `<!-- review_sha: ... -->` and skips re-reviewing unchanged commits unless `--force` is provided.
- **Stale Diff Consistency Engine**: Exponential backoff retry logic to handle asynchronous diff generation in GitLab/GitHub after fresh pushes.
- **Production Edge-Case Protections**:
  - **Noise Filter**: Auto-ignores lockfiles (`go.sum`, `package-lock.json`), minified files (`*.min.js`), and flags binary files.
  - **Comment Budgeting**: Caps inline comments to top priority issues (`--max-comments 15`) to prevent reviewer fatigue.
  - **Secret Scanner**: Pre-scans for exposed API keys and private keys.
  - **Draft/WIP PR Guardrails**: Skips draft pull requests unless `--ignore-drafts=false`.
- **Universal Multi-VCS Ingress**: Smart URL parsing for GitLab (SaaS & Self-Hosted with `--skip-tls-verify`), GitHub, Bitbucket, and Local Git diffs.
- **Zenith & ZenithUI Ingestion Protocol**: Ingests custom prompts and rules via `--prompt`, `--rules-json`, `--rules`, or direct Go SDK calls.

---

## Installation & Build

```bash
git clone https://github.com/divmora/code-reviewer-ai-agent.git
cd code-reviewer-ai-agent

# Build binary
make build
```

---

## Usage Examples

### 1. Review a GitLab Merge Request (Cloud or Self-Hosted)
```bash
# Self-hosted GitLab with custom domain & internal SSL
go run . --url https://gitlab.corp.internal/group/repo/-/merge_requests/42 \
         --token $GITLAB_TOKEN \
         --skip-tls-verify \
         --post

# GitLab.com SaaS
go run . --url https://gitlab.com/group/repo/-/merge_requests/1 \
         --token $GITLAB_TOKEN \
         --post
```

### 2. Review a GitHub Pull Request
```bash
go run . --url https://github.com/myorg/myrepo/pull/123 \
         --token $GITHUB_TOKEN \
         --post
```

### 3. Review a Bitbucket Pull Request
```bash
go run . --url https://bitbucket.org/myorg/myrepo/pull-requests/9 \
         --token $BITBUCKET_TOKEN \
         --post
```

### 4. Review Local Workspace Changes
```bash
# Review uncommitted changes in current directory
go run . --workspace . --diff

# Review branch comparison against main
go run . --workspace . --against origin/main

# Export review report to Markdown or JSON
go run . --workspace . --diff --format markdown --output review.md
```

### 5. Zenith & Custom Rule Ingestion
```bash
# Pass custom prompt from Zenith UI
go run . --url https://gitlab.corp.internal/group/repo/-/merge_requests/42 \
         --prompt "Always enforce try-catch with report(\$e) and check DB transactions" \
         --post

# Pass structured JSON rules
go run . --url https://gitlab.corp.internal/group/repo/-/merge_requests/42 \
         --rules-json '{"rules":[{"name":"Strict Auth","description":"All endpoints must use auth middleware"}]}'
```

---

## CLI Flags Reference

| Flag | Default | Description |
| :--- | :--- | :--- |
| `--url` | `""` | Pull Request / Merge Request URL (GitLab self-hosted/cloud, GitHub, Bitbucket) |
| `--workspace` | `.` | Local workspace directory to review |
| `--diff` | `false` | Review local uncommitted changes |
| `--against` | `""` | Target branch to diff against (e.g., `origin/main`) |
| `--provider` | `auto` | Force provider: `gitlab`, `github`, `bitbucket`, `local` |
| `--base-url` | `""` | Custom host URL for self-hosted GitLab |
| `--token` | `$TOKEN` | VCS API Token (or via `GITLAB_TOKEN`, `GITHUB_TOKEN`, `BITBUCKET_TOKEN`) |
| `--skip-tls-verify` | `false` | Skip SSL certificate validation (for internal self-hosted GitLab) |
| `--head-sha` | `""` | Expected commit SHA (verifies diff freshness) |
| `--force` | `false` | Force re-review even if commit SHA watermark indicates already reviewed |
| `--post` | `false` | Post review comments, summary, and commit status to PR/MR |
| `--update-description` | `true` | Update MR/PR description with Scorecard, Walkthrough, and SHA watermark |
| `--update-labels` | `true` | Update MR/PR labels with Quality, Security, and Maintainability scores |
| `--update-title` | `false` | Update MR/PR title with suggested conventional commit title |
| `--auto-approve` | `false` | Automatically approve MR/PR if all scores >= 80 and no critical bugs |
| `--ignore-drafts` | `true` | Skip reviewing Draft / WIP pull requests |
| `--max-comments` | `15` | Maximum number of inline comments to post (prioritizes critical findings) |
| `--format` | `terminal` | Output format: `terminal`, `markdown`, `json`, `sarif` |
| `--output` | `""` | File path to save review report (e.g. `review.md`) |
| `--profile` | `balanced` | Review sensitivity profile: `chill`, `balanced`, `assertive` |
| `--prompt` | `""` | Custom project prompt (passed from Zenith UI or CLI) |
| `--rules-json` | `""` | Inline JSON structured rules payload from Zenith UI |
| `--rules` | `""` | Path to custom YAML/JSON rules file |
| `--rule` | `""` | Ad-hoc inline rule instruction |
| `--litellm-base-url` | `""` | Custom LiteLLM proxy URL (or env `CODE_REVIEWER_LITELLM_BASE_URL` / `LITELLM_BASE_URL`) |
| `--litellm-api-key` | `""` | Custom LiteLLM API key (or env `CODE_REVIEWER_LITELLM_API_KEY` / `LITELLM_API_KEY`) |
| `--litellm-model` | `""` | Custom model identifier (or env `CODE_REVIEWER_LITELLM_MODEL` / `LITELLM_MODEL`) |
| `--litellm-endpoint` | `""` | Named endpoint in `~/.divmora/config/litellm.json` (or env `CODE_REVIEWER_LITELLM_ENDPOINT` / `LITELLM_ENDPOINT`) |
| `--verbose` | `false` | Enable verbose debug logging |

---

## Model & LLM Resolution Priority Flow

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

## Docker Usage

You can run the agent inside Docker or with docker-compose:

```bash
# Build the Docker image
docker build -t ghcr.io/divmora/code-reviewer-ai-agent:latest .

# Review a GitLab MR via Docker
docker run --rm \
  -v ~/.divmora:/root/.divmora \
  ghcr.io/divmora/code-reviewer-ai-agent:latest \
  --url "https://gitlab.pixelvide.com/group/repo/-/merge_requests/123" \
  --token "$GITLAB_TOKEN" \
  --skip-tls-verify \
  --post
```

---

## Testing

```bash
make test
```

---

## License

MIT License. See [LICENSE](LICENSE) for details.
