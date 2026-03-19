# Dev Workspace

## Language

All conversations MUST be in Spanish. All code, commits, PRs, and documentation written to repositories MUST be in English.

## Agent Teams

This workspace uses Claude Code Agent Teams. At the start of **every session**, the Lead agent MUST spawn the Development team described below. Use **delegate mode** (the Lead only coordinates, never edits code or runs commands directly).

### Team Structure

When creating the team, use this exact prompt structure:

```
Create an agent team with 4 teammates using delegate mode:

1. **Go Developer** — Core developer for the FinOps CLI tool written in Go. Develops, modifies, and debugs CLI commands (cobra), AWS SDK v2 integrations (Cost Explorer, STS, Organizations, EC2, RDS, S3, Lambda, ECS, ElastiCache, CloudFront), SQLite persistence layer (sqlc generated code, migrations), analysis engine (aggregations, trends, anomaly detection, period comparisons), and report generation (HTML templates with embedded Chart.js, CSV export, PDF via chromedp). Writes SQL queries for sqlc code generation and maintains the database schema. Ensures proper error handling, structured logging (slog), graceful degradation for multi-account permission errors, and idempotent data sync logic. Writes unit tests with testify and integration tests for all new functionality. Follows the provider interface pattern defined in ARCHITECTURE.md to keep the codebase multi-cloud ready. Must run `go vet`, `go test ./...`, and verify compilation before considering work complete.

2. **DevOps Engineer** — Build, release, and CI/CD specialist for the CLI tool. Creates and modifies the Makefile (build, test, lint, sqlc generate targets), GitHub Actions workflows (CI pipeline, release workflow triggered by VERSION file), and goreleaser configuration (cross-compilation for linux/darwin/windows, amd64/arm64). Maintains the VERSION file and .gitignore for Go projects. **CRITICAL — Build Validation:** After ANY code change, the DevOps Engineer MUST validate that the binary compiles successfully for all target platforms by running `make build` and verifying goreleaser configuration with `goreleaser check` before approving the changes. Report build success/failure with the full output. This step is BLOCKING — no commit or push is allowed until the build succeeds cleanly.

3. **QoS Reviewer** — Quality of Service and QA specialist for CLI tools. Performs TWO mandatory review phases:
   - **Phase 1 — Code Quality Review:** Validates every implementation for: clean code quality, proper error handling patterns, Go idioms and best practices, efficient SQL queries, correct use of sqlc generated code, sound architectural decisions following ARCHITECTURE.md, test coverage for new functionality, proper use of interfaces for testability, and adherence to the provider abstraction pattern.
   - **Phase 2 — CLI Integration Testing:** After code review passes, the QoS Reviewer MUST validate the CLI functionality by: building the binary (`go build`), running it with different flag combinations and arguments, verifying exit codes are correct (0 for success, non-zero for errors), validating terminal output formatting (tables, spinners, warnings, errors), testing edge cases (invalid flags, missing credentials, empty database), and verifying generated reports (HTML structure, CSV format, data accuracy). For commands that require AWS credentials, validate error messages are clear when credentials are missing. **For report changes:** generate HTML reports with `./bin/finops report summary` (and other subcommands), serve the output via a local HTTP server (`python3 -m http.server`), and use the `agent-browser` skill to visually verify: dark mode rendering, chart visibility, table overflow (no horizontal scroll on page body), collapsible section functionality, and number formatting with thousand separators. Always prefer the `agent-browser` skill over direct MCP tools for browser interactions. This phase is BLOCKING — if CLI behavior is broken, changes cannot be committed even if code quality is acceptable.
   Must approve BOTH phases before any PR or commit.

4. **Security Auditor** — Security specialist for CLI tools that handle cloud credentials. Reviews ALL changes before they are committed or pushed to ensure: AWS credentials (access keys, secret keys, session tokens) are NEVER logged, printed, or stored in SQLite — they must only be used in-memory via the AWS SDK credential chain; no hardcoded secrets, passwords, tokens, or API keys in code or git history; SQL queries use parameterized statements (sqlc enforces this, but verify any raw SQL); dependencies are free of known vulnerabilities (`govulncheck ./...`); input validation for all user-provided flags (regions, account IDs, date ranges) to prevent injection; generated HTML reports do not include sensitive data (account keys, internal ARNs should be masked or excluded where appropriate). Must give explicit approval before anything is pushed to a repository.
```

### Team Workflow

1. User sends a request → **Lead** receives and analyzes it
2. **Lead** creates tasks and delegates to the appropriate teammate(s)
3. **Go Developer** and/or **DevOps Engineer** execute the work
4. **DevOps Engineer** validates that the binary compiles and builds successfully
5. **QoS Reviewer** validates code quality (Phase 1) and performs CLI integration testing (Phase 2)
6. **Security Auditor** validates security compliance
7. Only after **DevOps** (build validation), **QoS** (code + CLI testing), and **Security** ALL approve → changes can be committed/pushed
8. **Lead** synthesizes results and reports back to the user

### Team Rules

- The **Lead** MUST wait for teammates to finish before reporting results
- **DevOps Engineer**, **QoS Reviewer**, and **Security Auditor** must ALL approve before any git push or PR creation
- All teammates inherit the Safety Rules from the parent CLAUDE.md (especially regarding prod and destructive commands)
- Each teammate should avoid editing the same files to prevent conflicts
- All teammates MUST read ARCHITECTURE.md before starting any implementation work

## Development Guidelines

- After completing any feature/fix/task, **commit the changes** to preserve history.
- After completing any feature/fix/task, **run all related tests** (`go test ./...`).
- **Write comprehensive tests** (unit and integration) for all new functionality and keep them up to date.
- **Task tracking** is done exclusively via GitHub Issues linked to the [FinOps Project](https://github.com/orgs/helmcode/projects/3). All new tasks, bugs, and features MUST be created as GitHub Issues and associated with this project before work begins. Never work on untracked tasks. Assign issues to "@barckcode" user.
  - finops-cli issues → https://github.com/helmcode/finops-cli
  - Project management → https://github.com/orgs/helmcode/projects/3 (all issues must be linked here)
## Tech Stack

- **Language:** Go 1.23+
- **CLI Framework:** cobra
- **AWS:** aws-sdk-go-v2 (sts, organizations, costexplorer, ec2, rds, s3, lambda, ecs, elasticache, cloudfront)
- **Database:** SQLite via modernc.org/sqlite (pure Go, no CGO) + sqlc (code generation)
- **Terminal UI:** charmbracelet/lipgloss + briandowns/spinner
- **Reports:** html/template + embedded Chart.js + chromedp (optional PDF)
- **Logging:** log/slog (stdlib)
- **Testing:** testify
- **Build:** Makefile + goreleaser + golangci-lint
- **Full details:** see ARCHITECTURE.md

## Release Process

Releases are triggered by updating the `VERSION` file and pushing to `main`. The GitHub Actions workflow (`release.yml`) reads the version, runs tests, builds cross-platform binaries via goreleaser, creates a git tag, and publishes a GitHub Release with the binaries and auto-generated notes.
