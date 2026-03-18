# FinOps CLI — Architecture

## Overview

Go-based CLI tool for cloud FinOps analysis. Scans cloud provider accounts, downloads cost data, discovers active resources, stores everything locally in SQLite, and generates rich reports with charts and anomaly detection.

**Current scope:** AWS (Azure and GCP planned for future phases).

## Design Principles

1. **Local-first**: All analysis runs against local SQLite data. AWS API calls only happen during `scan`. Reports never trigger cloud API calls.
2. **Cost Explorer as the source of truth**: Cost Explorer determines which services, regions, and accounts have spend. Only resources with actual cost are stored — no noise.
3. **Incremental sync**: First run downloads 6 months of history. Subsequent runs only fetch from the last synced date to today. Data accumulates naturally up to the retention limit.
4. **Graceful degradation**: In multi-account organizations, permission errors on individual accounts produce warnings, not failures. The scan continues with accessible accounts.
5. **Multi-cloud ready**: Provider interface abstraction allows adding Azure/GCP without touching core logic.

## CLI Commands

```
finops (binary)
│
├── scan          Download costs + discover resources
│   --provider    aws (required, future: azure, gcp)
│   --region      us-east-1 | all (default: all)
│   --months      1-12 (default: 6)
│   --from/--to   specific range (max 12 months, validated)
│   --account     filter accounts: 1 or comma-separated list (orgs only)
│   --verbose     show detailed output including skipped accounts
│
├── report        Generate reports from local data
│   summary       General overview with charts
│   top-services  Top N services by cost
│   trend         Temporal trend for a service
│   anomalies     Anomalous cost spike detection
│   compare       Compare two periods
│   resources     Discovered resources + cost context
│   --output      html (default) | csv | pdf
│   --file        output path (default: auto-open in browser for html)
│   --limit       number of items for top-services (default: 10)
│   --service     filter by service name (for trend, resources)
│   --region      filter by region (for resources)
│   --current     period for comparison (for compare)
│   --previous    period for comparison (for compare)
│
├── db            Local database management
│   stats         DB info (size, record counts, last sync date)
│   prune         Modify data retention
│   --retention   months (default: 12, min: 1, no max but warns if >24)
│
└── version       CLI version
```

## Scan Flow

```
finops scan --provider aws --region all
│
├─ 1. sts:GetCallerIdentity
│     → Identify current account ID
│
├─ 2. organizations:DescribeOrganization
│     ├─ ✅ Success → AWS Organization detected
│     │   └─ organizations:ListAccounts → list all member accounts
│     │       → Apply --account filter if provided
│     │
│     └─ ❌ AccessDenied / OrgNotFound
│         → Single account mode (current account only)
│
├─ 3. Determine sync range
│     ├─ Has previous sync data? → incremental (last synced date → today)
│     └─ No previous data? → initial sync (last N months, default 6)
│     └─ Validate range ≤ 12 months, error if exceeded
│
├─ 4. For each account (with graceful error handling):
│     │
│     ├─ Cost Explorer: GetCostAndUsage
│     │   GROUP_BY: [SERVICE, REGION]
│     │   Granularity: MONTHLY
│     │   → Returns only services/regions with actual spend
│     │
│     ├─ If --region all:
│     │   ec2:DescribeRegions (opt-in filter)
│     │   → Only process regions that are both enabled AND have spend
│     │
│     ├─ Resource Discovery (for each service with spend):
│     │   ├─ Check registry: do we have a discoverer for this service?
│     │   ├─ YES → call service-specific API (DescribeInstances, etc.)
│     │   └─ NO → log info, cost data still saved without resource detail
│     │
│     ├─ SQLite: INSERT OR REPLACE
│     │   ├─ cost_records (monthly costs)
│     │   └─ resources (discovered resources)
│     │
│     └─ On AccessDenied for an account:
│         ⚠️ WARNING (do not stop), continue with next account
│
├─ 5. Auto-prune records older than retention limit
│
└─ 6. Summary output:
       "Scan completed: 2/3 accounts processed, 1 skipped (no permissions)"
       "Cost records: 156 synced | Resources: 47 discovered"
       "Run with --verbose for details"
```

## Multi-Account Behavior

### Auto-Detection

The CLI automatically detects whether credentials belong to an Organization management/delegated-admin account or a standalone account:

1. `sts:GetCallerIdentity` → current account ID
2. `organizations:DescribeOrganization` → if success, it's an org
3. `organizations:ListAccounts` → enumerate member accounts

### Permission Handling

In organizations, the user may have cost visibility into some accounts but not others. The CLI handles this gracefully:

- Each account is processed independently
- `AccessDeniedException` on a specific account → warning, skip, continue
- Final summary reports: accounts processed, accounts skipped, reasons
- `--verbose` flag shows per-account details
- `--account 111,222,333` flag filters to specific accounts only

### Cost Explorer in Organizations

- Management account credentials: consolidated view of all accounts
- Member account credentials: only that account's costs
- `GROUP_BY LINKED_ACCOUNT` used when in org mode to separate per-account data

## Resource Discovery

### Dynamic Discovery via Registry Pattern

Cost Explorer tells us which services have spend. A registry of discovery adapters provides resource-level detail for supported services.

```
Cost Explorer output:
  "Amazon Elastic Compute Cloud": $2,300
  "Amazon RDS": $800
  "Amazon Managed Blockchain": $50

Registry lookup:
  EC2 → ✅ EC2Discoverer → DescribeInstances
  RDS → ✅ RDSDiscoverer → DescribeDBInstances
  Managed Blockchain → ❌ No adapter
    → Cost saved, no resource detail
    → Info message to user
```

### Discovery Adapter Interface

Each adapter implements:
- Map a Cost Explorer service name to the corresponding AWS API
- Call the service API to list individual resources
- Return normalized Resource structs with type, spec, tags, state

### v1 Adapters (initial set, grows over time)

- EC2 (instances + EBS volumes)
- RDS
- S3
- Lambda
- ECS/EKS
- ElastiCache
- NAT Gateway
- CloudFront

The adapter list is not a filter — it's a capability set. If a service has spend but no adapter, the cost is still recorded. The adapter only adds resource-level detail.

## Data Model (SQLite)

Database location: `~/.finops/data.db`

### cost_records

Monthly cost data per service/region/account.

| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER PK | Auto-increment |
| provider | TEXT NOT NULL | aws, azure, gcp |
| account_id | TEXT NOT NULL | AWS account ID |
| service | TEXT NOT NULL | Cost Explorer service name |
| region | TEXT | Region (NULL if global, e.g., S3) |
| period_start | TEXT NOT NULL | First day of period (2026-01-01) |
| period_end | TEXT NOT NULL | First day of next period (2026-02-01) |
| granularity | TEXT NOT NULL | MONTHLY (default, only option in v1) |
| amount | REAL NOT NULL | Cost amount |
| currency | TEXT NOT NULL | USD (default) |
| synced_at | TEXT NOT NULL | Timestamp of sync |

**Unique constraint:** `(provider, account_id, service, region, period_start, granularity)`
Enables idempotent `INSERT OR REPLACE` on every sync.

### resources

Discovered resources linked to services with spend.

| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER PK | Auto-increment |
| provider | TEXT NOT NULL | aws |
| account_id | TEXT NOT NULL | AWS account ID |
| service | TEXT NOT NULL | Matches cost_records.service |
| resource_id | TEXT NOT NULL | i-abc123, ARN, bucket name |
| resource_type | TEXT NOT NULL | ec2:instance, rds:db, s3:bucket |
| name | TEXT | Name tag if exists |
| region | TEXT | Resource region |
| spec | TEXT (JSON) | {instance_type, engine, storage_gb, ...} |
| tags | TEXT (JSON) | {env: "prod", team: "backend"} |
| state | TEXT | running, stopped, available |
| discovered_at | TEXT NOT NULL | Timestamp of discovery |

**Unique constraint:** `(provider, account_id, resource_id)`

The `spec` field is intentionally JSON to accommodate different resource types without requiring a table per service.

### sync_history

Audit trail of scan operations.

| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER PK | Auto-increment |
| provider | TEXT NOT NULL | aws |
| account_id | TEXT NOT NULL | Account scanned |
| region | TEXT | Region (NULL if all) |
| period_start | TEXT NOT NULL | Start of synced range |
| period_end | TEXT NOT NULL | End of synced range |
| cost_records | INTEGER | Number of cost records synced |
| resources_found | INTEGER | Number of resources discovered |
| started_at | TEXT NOT NULL | Scan start timestamp |
| completed_at | TEXT | Scan end timestamp (NULL if failed) |

### config

Persistent CLI configuration.

| Column | Type | Description |
|--------|------|-------------|
| key | TEXT PK | retention_months, default_provider, ... |
| value | TEXT NOT NULL | Configuration value |

## Smart Sync Logic

### Incremental Sync

```
scan invoked
│
├─ Query sync_history: what's the latest period_end for this provider/account?
│
├─ Has previous sync?
│   ├─ YES: fetch from last period_end to today (incremental)
│   └─ NO: fetch last 6 months (or --months N)
│
├─ Validate: requested range ≤ 12 months
│   └─ If exceeded → error with clear message
│
└─ After sync: auto-prune records older than retention
```

### Report Data Freshness

```
report invoked
│
├─ Check: do we have data covering the requested range?
│   ├─ YES → use local data, zero AWS calls
│   ├─ PARTIAL → warning: "Data available until 2026-01. Run 'finops scan' to update."
│   └─ NONE → error: "No data found. Run 'finops scan --provider aws' first."
│
└─ Reports NEVER call AWS APIs
```

## Data Retention

- **Default retention:** 12 months
- **Default sync range:** 6 months (first run)
- Retention is enforced automatically after every scan (auto-prune)
- `finops db prune --retention N` changes the retention period
  - Stored in config table, persists across runs
  - Warns if N > 24 (unusual, likely unintended)
  - Minimum: 1 month
- Retention only controls how long data lives in the DB
- Retention does NOT affect AWS API queries (those are capped at 12 months by Cost Explorer)
- If user requests a comparison involving months beyond what's in DB but within what was previously synced (and pruned), the CLI explains: "Data for 2024-06 is no longer available (retention: 12 months). Adjust retention with 'finops db prune --retention N' to keep older data."

## Report Generation

### Output Formats

| Format | Description |
|--------|-------------|
| HTML (default) | Self-contained HTML file with Chart.js embedded inline (no CDN). Auto-opens in browser. |
| CSV | Standard CSV export for spreadsheet analysis |
| PDF | Generated from HTML rendering (headless browser conversion) |

### Report Types

**summary**: Full overview — total spend, monthly trend bar chart, top services pie chart, anomaly alerts, resource counts by service.

**top-services**: Ranked list of services by total cost over the selected period. Includes monthly average and trend direction.

**trend**: Line chart showing cost evolution over time for a specific service or all services.

**anomalies**: Statistical detection of cost spikes using z-score over moving average. Highlights months where spend deviates significantly from the norm.

**compare**: Side-by-side comparison of two periods. Shows absolute and percentage changes per service.

**resources**: List of discovered resources for a service/region, with spec details (instance type, state, etc.) and aggregate cost context (service-level, not per-resource in v1).

### Resource ↔ Cost Cross-Reference (Phased)

**v1 — Service-level context:**
"You have 15 EC2 instances in us-east-1 costing $2,300/month total. 3x m5.xlarge, 8x t3.medium, 4x t3.micro."

**v2 (future) — Resource-level cost:**
"Instance i-abc123 (m5.xlarge) costs $X/month." Requires AWS Cost and Usage Reports (CUR) or cost allocation tags.

## Project Structure

```
finops-cli/
├── cmd/                              # CLI commands (cobra)
│   ├── root.go                       # Root command + global flags
│   ├── scan.go                       # finops scan
│   ├── report.go                     # finops report + subcommands
│   ├── db.go                         # finops db
│   └── version.go                    # finops version
│
├── internal/
│   ├── provider/                     # Multi-cloud abstraction
│   │   ├── provider.go               # Provider interface definitions
│   │   └── aws/
│   │       ├── client.go             # AWS session/client factory
│   │       ├── costs.go              # Cost Explorer API integration
│   │       ├── regions.go            # Active region detection
│   │       ├── organization.go       # Org detection + account listing
│   │       └── discovery/            # Resource discovery adapters
│   │           ├── registry.go       # Adapter registry
│   │           ├── ec2.go            # EC2 instances + EBS
│   │           ├── rds.go            # RDS instances
│   │           ├── s3.go             # S3 buckets
│   │           ├── lambda.go         # Lambda functions
│   │           ├── ecs.go            # ECS/EKS clusters
│   │           ├── elasticache.go    # ElastiCache clusters
│   │           ├── nat.go            # NAT Gateways
│   │           └── cloudfront.go     # CloudFront distributions
│   │
│   ├── store/                        # SQLite persistence layer (sqlc generated)
│   │   ├── db.go                     # sqlc generated: DB interface + DBTX
│   │   ├── models.go                 # sqlc generated: Go structs from schema
│   │   ├── cost_records.sql.go       # sqlc generated: cost record queries
│   │   ├── resources.sql.go          # sqlc generated: resource queries
│   │   ├── sync_history.sql.go       # sqlc generated: sync history queries
│   │   ├── config.sql.go             # sqlc generated: config queries
│   │   ├── migrate.go                # Custom migration runner (embed + sql)
│   │   └── store.go                  # Store wrapper (connection, init, prune)
│   │
│   ├── analysis/                     # Analysis engine
│   │   ├── summary.go               # Summary aggregations
│   │   ├── trends.go                # Trend calculations
│   │   ├── anomaly.go               # Anomaly detection (z-score)
│   │   └── compare.go              # Period comparison
│   │
│   └── report/                       # Report generation
│       ├── html.go                   # HTML generator with embedded Chart.js
│       ├── csv.go                    # CSV export
│       ├── pdf.go                    # HTML → PDF conversion (optional, needs Chrome)
│       └── templates/                # Go HTML templates (embedded via embed)
│           ├── base.html             # Shared layout + Chart.js inline
│           ├── summary.html
│           ├── top_services.html
│           ├── trend.html
│           ├── anomalies.html
│           ├── compare.html
│           └── resources.html
│
├── db/
│   ├── schema.sql                    # Full database schema (sqlc input)
│   ├── queries/                      # SQL queries (sqlc input)
│   │   ├── cost_records.sql          # Cost record queries
│   │   ├── resources.sql             # Resource queries
│   │   ├── sync_history.sql          # Sync history queries
│   │   └── config.sql               # Config queries
│   └── migrations/                   # Ordered migration files (embedded)
│       ├── 001_initial_schema.sql
│       └── ...
│
├── sqlc.yaml                         # sqlc configuration
├── main.go                           # Entry point
├── go.mod
├── go.sum
├── Makefile                          # Build, test, lint, sqlc generate targets
├── .goreleaser.yaml                  # Cross-compilation + release config
├── VERSION                           # Triggers release workflow
├── LICENSE
├── ARCHITECTURE.md                   # This file
└── CLAUDE.md                         # Development guidelines
```

## Provider Interface

```go
// Core abstractions that enable multi-cloud support

type Provider interface {
    Name() string                                          // "aws", "azure", "gcp"
    DetectAccountMode() (AccountMode, error)               // single vs organization
    ListAccounts(filter []string) ([]Account, error)       // accounts to scan
    FetchCosts(params CostParams) ([]CostRecord, error)    // cost data
    DiscoverResources(service, region string) ([]Resource, error)
    GetActiveRegions() ([]string, error)
}

type AccountMode struct {
    IsOrganization bool
    ManagementID   string
    Accounts       []Account
}

type CostParams struct {
    AccountID   string
    Start       time.Time
    End         time.Time
    Granularity string    // "MONTHLY"
    GroupBy     []string  // ["SERVICE", "REGION"]
}
```

## Validation Rules

| Rule | Behavior |
|------|----------|
| --months > 12 | Error: "maximum range is 12 months (AWS Cost Explorer limit)" |
| --from/--to range > 12 months | Error: "range exceeds 12 months (N months requested, max 12)" |
| --from after --to | Error: "--from must be before --to" |
| --from in the future | Error: "--from cannot be in the future" |
| --account on single account | Warning: "ignored, single account detected" |
| --retention > 24 | Warning: "unusual retention period, confirm with --force" |
| --retention < 1 | Error: "minimum retention is 1 month" |
| Report range not in DB | Warning with guidance to run scan |
| Report range partially in DB | Warning showing available range |

## Tech Stack

### Go Version

Go 1.23+ (latest stable).

### Dependencies

#### CLI Framework

| Package | Purpose |
|---------|---------|
| `github.com/spf13/cobra` | CLI framework (subcommands, flags, autocompletion, help). Standard in Go ecosystem (kubectl, helm, gh). |

No viper — configuration lives in SQLite config table.

#### AWS SDK v2

| Package | Purpose |
|---------|---------|
| `github.com/aws/aws-sdk-go-v2/config` | Credential loading (env vars, ~/.aws/credentials, IAM roles) |
| `github.com/aws/aws-sdk-go-v2/service/sts` | GetCallerIdentity — identify account |
| `github.com/aws/aws-sdk-go-v2/service/organizations` | Org detection + list member accounts |
| `github.com/aws/aws-sdk-go-v2/service/costexplorer` | GetCostAndUsage — cost data |
| `github.com/aws/aws-sdk-go-v2/service/ec2` | Instance/EBS/NAT discovery + active regions |
| `github.com/aws/aws-sdk-go-v2/service/rds` | RDS instance discovery |
| `github.com/aws/aws-sdk-go-v2/service/s3` | S3 bucket discovery |
| `github.com/aws/aws-sdk-go-v2/service/lambda` | Lambda function discovery |
| `github.com/aws/aws-sdk-go-v2/service/ecs` | ECS/EKS cluster discovery |
| `github.com/aws/aws-sdk-go-v2/service/elasticache` | ElastiCache cluster discovery |
| `github.com/aws/aws-sdk-go-v2/service/cloudfront` | CloudFront distribution discovery |

Always v2. v1 is legacy.

#### SQLite + sqlc

| Package | Purpose |
|---------|---------|
| `modernc.org/sqlite` | Pure Go SQLite driver (no CGO). Enables trivial cross-compilation for Linux/macOS/Windows without a C compiler. |
| `database/sql` (stdlib) | Standard database interface |
| `github.com/sqlc-dev/sqlc` | Build-time code generator: write SQL, get type-safe Go functions. Zero runtime dependency. |

Migrations: SQL files embedded via Go `embed` directive + minimal custom runner. No goose/migrate (overkill for 3-4 migrations).

#### Terminal Output

| Package | Purpose |
|---------|---------|
| `github.com/charmbracelet/lipgloss` | Styled terminal output (colors, borders, formatted tables) for scan summaries and db stats |
| `github.com/briandowns/spinner` | Progress spinner during AWS API calls |

No bubbletea (full TUI framework) in v1 — reserved for v2 interactive reports.

#### Report Generation

| Technology | Purpose |
|------------|---------|
| `html/template` (stdlib) | HTML report generation with Go's built-in template engine |
| `embed` (stdlib) | Embed Chart.js, HTML templates, and CSS into the binary — single self-contained binary |
| **Chart.js** (~200KB min) | Client-side charting library embedded inline in HTML. Bar, line, pie charts. No CDN. |
| `github.com/chromedp/chromedp` | HTML → PDF via headless Chrome. Optional: only works if Chrome/Chromium is installed. Clear error message if not available. |

PDF is optional (Option C): HTML is the primary format, PDF is a bonus if Chrome is available.

#### Logging

| Package | Purpose |
|---------|---------|
| `log/slog` (stdlib) | Structured logging (Go 1.21+). Info/Warn/Error/Debug levels, text or JSON format. |

#### Testing

| Package | Purpose |
|---------|---------|
| `testing` (stdlib) | Test framework |
| `github.com/stretchr/testify` | Readable assertions (assert.Equal, require.NoError) |

AWS service calls are accessed through interfaces for easy mocking in tests.

#### Build & Release

| Tool | Purpose |
|------|---------|
| `Makefile` | Local development targets: build, test, lint, sqlc generate |
| `goreleaser` | Cross-compilation + packaging for releases (linux/darwin/windows, amd64/arm64). Integrates with GitHub Actions. No Docker — binary distribution only. |
| `golangci-lint` | Aggregated linter (govet, errcheck, staticcheck, etc.) |

### Dependency Summary

```
go.mod (~16 direct dependencies)
│
├── CLI
│   └── github.com/spf13/cobra
│
├── AWS (v2)
│   ├── config, sts, organizations, costexplorer
│   ├── ec2, rds, s3, lambda
│   └── ecs, elasticache, cloudfront
│
├── Storage
│   └── modernc.org/sqlite
│
├── Terminal
│   ├── github.com/charmbracelet/lipgloss
│   └── github.com/briandowns/spinner
│
├── Reports
│   └── github.com/chromedp/chromedp (PDF only, optional)
│
└── Testing
    └── github.com/stretchr/testify
```

stdlib covers the rest: html/template, embed, database/sql, log/slog, encoding/json.

### Deliberately Excluded

| Excluded | Reason |
|----------|--------|
| viper | Config lives in SQLite, no YAML/TOML/env parsing needed |
| GORM / sqlx | sqlc generates type-safe code from real SQL, no ORM magic |
| mattn/go-sqlite3 | Requires CGO, complicates cross-compilation |
| Docker | CLI distributed as native binaries via goreleaser. Docker adds friction (volume mounts for ~/.aws, ~/.finops). Can be added later if needed for CI/CD use cases. |
| bubbletea | Full TUI is v2, not v1 |
| goose / golang-migrate | Overkill for 3-4 simple migrations |
| zerolog / zap | slog is in stdlib and sufficient |

## Future Roadmap

### v1 (Current)
- AWS support (Cost Explorer + resource discovery)
- SQLite storage with smart sync
- HTML/CSV/PDF reports with charts
- Monthly granularity
- Service-level cost ↔ resource cross-reference

### v2
- Daily granularity option
- Interactive HTML reports (mini local server with drill-down)
- Resource-level cost attribution (via CUR or cost allocation tags)
- Cost optimization recommendations (stopped instances, unused EBS, etc.)

### v3
- Azure support
- GCP support
- Multi-cloud unified reports
- Budget alerts and threshold notifications
