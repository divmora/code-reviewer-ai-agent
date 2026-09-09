# Code Reviewer AI Agent Product Roadmap

This document serves as the **living product roadmap** for Code Reviewer AI Agent.
- **Adding Items**: Whenever a new capability, enhancement, or edge-case improvement is identified for the future, add it here under the appropriate category.
- **Removing Items**: Once a feature is fully implemented, verified with tests, and committed, **remove it from this roadmap** immediately to keep active focus on upcoming tasks.

---

## 1. VCS Providers & Ecosystem Integrations

- [ ] **Bitbucket Server / Data Center On-Premises API**
  - Add native support for Bitbucket Server REST APIs with bearer token / HTTP basic authentication and self-signed TLS certificates.
  - Implement inline pull request comments with Bitbucket anchor positioning.

- [ ] **Azure DevOps (ADO) Pull Request Integration**
  - Support Azure DevOps Git repositories and pull requests via Azure DevOps REST API.
  - Post threaded review comments and update pull request status policies.

- [ ] **Gerrit Code Review Adapter**
  - Integrate Gerrit change review workflows with patchset diff parsing and inline robotic comments.

---

## 2. AST Parsing & Semantic Code Analysis

- [ ] **Tree-Sitter Multi-Language Universal AST Slicer**
  - Expand beyond Go, Python, TypeScript, PHP, Java, and Rust to support C++, C#, Kotlin, Swift, Scala, and Elixir via Tree-sitter bindings or WebAssembly parsers.
  - Provide fine-grained symbol extraction for nested lambdas, closures, and interface definitions.

- [ ] **Cross-File Symbol & Dependency Reference Resolution**
  - Trace modified function calls into unedited caller/callee files to detect breaking signature changes across the repository.
  - Construct lightweight call graphs for touched packages during review execution.

---

## 3. AI Review Intelligence & Custom Rules Engine

- [ ] **Rego (Open Policy Agent) & CEL Lint Rules Engine**
  - Allow teams to define declarative architectural guardrails in Rego or Common Expression Language (CEL) alongside natural language rules.
  - Run deterministic static policy checks prior to invoking the LLM review pipeline.

- [ ] **Conversational Review Continuity & Thread Memory**
  - Maintain context across multiple review iterations within the same pull/merge request.
  - Detect author replies to previous AI comments and resolve discussions when fixes are verified in subsequent commits.

- [ ] **Automated Test Generation for Suggestions**
  - Generate corresponding unit tests (e.g. `_test.go`, `.test.ts`, `test_*.py`) alongside 1-click code suggestions.

---

## 4. Performance, Caching & Scalability

- [ ] **Distributed & Local Caching Engine**
  - Cache AST parse trees, token budgets, and diff chunk embeddings using Redis or local SQLite.
  - Accelerate iterative reviews for large repositories where only minor commits are pushed.

- [ ] **Chunked Parallel Reviewer Worker Pools**
  - Partition massive pull requests (> 50 changed files or > 2,000 diff lines) across parallel subagent reviewer workers.
  - Merge findings and deduplicate scorecard metrics into a cohesive final review report.

- [ ] **Streaming Inline Suggestion Publishing**
  - Stream reviewer findings as they are generated rather than batch-publishing after the entire review cycle completes.

---

## 5. Notifications, Webhooks & Enterprise Tooling

- [ ] **Slack & Microsoft Teams Interactive Webhooks**
  - Dispatch concise review summary cards with Quality/Security/Maintainability scorecards to Slack or Teams channels.
  - Include quick links to high-severity findings and 1-click suggestion threads.

- [ ] **SARIF & SonarQube Exporters**
  - Export review findings in OASIS SARIF (Static Analysis Results Interchange Format) for direct integration into GitHub Code Scanning and SonarQube dashboards.

- [ ] **Executive PR Summary Podcast / Voice Overview**
  - Optional audio summary synthesis for high-level architectural walkthroughs on major cross-team pull requests.
