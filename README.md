# FinOps CLI

A command-line tool for AWS cloud cost analysis and optimization. Scans your AWS accounts, downloads cost data from Cost Explorer, discovers active resources, stores everything locally in SQLite, and generates rich interactive reports.

**Local-first design** — your data stays on your machine. Reports never call cloud APIs.

## Install

### Agent Skill

Add the FinOps skill to your AI agent (Claude Code, or any [skills.sh](https://skills.sh)-compatible agent):

```bash
npx skills add helmcode/finops-cli
```

This gives your agent the knowledge to use the FinOps CLI effectively — cost analysis, anomaly detection, report generation, and more.

### CLI

#### Quick install (Linux / macOS)

```bash
curl -fsSL https://raw.githubusercontent.com/helmcode/finops-cli/main/install.sh | sh
```

The script auto-detects your OS and architecture, downloads the correct binary, verifies the SHA256 checksum, and installs it.

#### Homebrew (macOS / Linux)

```bash
brew install helmcode/tap/finops-cli
```

#### Go install

```bash
go install github.com/helmcode/finops-cli@latest
```

#### From source

Requires Go 1.25+:

```bash
git clone https://github.com/helmcode/finops-cli.git
cd finops-cli
make build
# Binary at ./bin/finops
```

## Features

- **Multi-account support** — auto-detects AWS Organizations, scans all linked accounts
- **Cost analysis** — monthly spend breakdown by service, region, and account
- **Resource discovery** — finds EC2, RDS, S3, Lambda, ECS/EKS, ElastiCache, NAT Gateways, CloudFront distributions
- **Anomaly detection** — statistical z-score analysis to flag cost spikes
- **Period comparison** — side-by-side cost comparison between any two periods
- **Commitment tracking** — Savings Plans and Reserved Instances utilization and coverage
- **Rich reports** — HTML with embedded charts (Chart.js), CSV, JSON, and PDF output
- **Incremental sync** — only fetches new data on subsequent runs
- **Zero CGO** — pure Go binary, cross-compiles to Linux, macOS, and Windows

## Quick Start

### 1. Configure AWS credentials

The CLI uses the standard AWS credential chain (environment variables, `~/.aws/credentials`, IAM roles, SSO):

```bash
export AWS_PROFILE=your-profile
# or
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...
```

### 2. Scan your account

```bash
# Scan last 6 months (default)
finops scan --provider aws

# Scan a specific region, last 3 months
finops scan --provider aws --region us-east-1 --months 3

# Scan specific accounts in an Organization
finops scan --provider aws --account 111111111111,222222222222

# Scan a custom date range
finops scan --provider aws --from 2025-01-01 --to 2025-06-01
```

### 3. Generate reports

```bash
# Summary report (opens in browser)
finops report summary

# Top 10 most expensive services
finops report top-services --limit 10

# Cost trend for a specific service
finops report trend --service "Amazon EC2"

# Detect anomalies
finops report anomalies

# Compare two periods
finops report compare --current 2025-05-01:2025-06-01 --previous 2025-04-01:2025-05-01

# List discovered resources
finops report resources --service "Amazon EC2" --region us-east-1
```

### 4. Export in different formats

```bash
# CSV for spreadsheets
finops report summary --output csv --file costs.csv

# JSON for programmatic use (designed for AI agents)
finops report summary --output json --file costs.json

# PDF (requires Chrome/Chromium installed)
finops report summary --output pdf --file costs.pdf
```

## Commands

### `finops scan`

Downloads cost data and discovers resources from your cloud provider.

| Flag | Description | Default |
|------|-------------|---------|
| `--provider` | Cloud provider (required) | — |
| `--region` | Region to scan | `all` |
| `--months` | Number of months to sync (1-12) | `6` |
| `--from` | Start date (`YYYY-MM-DD`) | — |
| `--to` | End date (`YYYY-MM-DD`) | — |
| `--account` | Filter account IDs (comma-separated) | — |
| `-v, --verbose` | Show detailed output | `false` |

**How it works:**
1. Detects account mode (single account vs. Organization)
2. Fetches monthly costs grouped by service and region from Cost Explorer
3. Discovers active resources for each service with spend
4. Stores everything in local SQLite (`~/.finops/data.db`)
5. Auto-prunes records older than the retention limit

Subsequent runs are incremental — only new data since the last sync is fetched.

### `finops report`

Generates reports from locally stored data. Never calls AWS APIs.

#### `finops report summary`

Full overview: total spend, monthly trend chart, top services pie chart, cost by region and account, anomaly alerts, commitment utilization, and resource inventory.

#### `finops report top-services`

Ranked list of services by cost with monthly averages and trend direction.

| Flag | Description | Default |
|------|-------------|---------|
| `--limit` | Number of services to show | `10` |

#### `finops report trend`

Line chart showing cost evolution over time.

| Flag | Description | Default |
|------|-------------|---------|
| `--service` | Filter by service name | all services |

#### `finops report anomalies`

Detects months with statistically unusual spend using z-score analysis (>2 standard deviations from moving average).

#### `finops report compare`

Side-by-side comparison of two periods with absolute and percentage changes per service.

| Flag | Description | Default |
|------|-------------|---------|
| `--current` | Current period (`YYYY-MM-DD:YYYY-MM-DD`) | required |
| `--previous` | Previous period (`YYYY-MM-DD:YYYY-MM-DD`) | required |

#### `finops report resources`

Lists discovered resources with specs, tags, and state.

| Flag | Description | Default |
|------|-------------|---------|
| `--service` | Filter by service | all |
| `--region` | Filter by region | all |

**All report subcommands support:**

| Flag | Description | Default |
|------|-------------|---------|
| `--output` | Format: `html`, `csv`, `json`, `pdf` | `html` |
| `--file` | Output file path | auto-generated |

HTML reports auto-open in your browser unless `--file` is specified.

### `finops db`

#### `finops db stats`

Shows database size, record counts, last sync date, and retention setting.

#### `finops db prune`

Changes data retention and removes old records.

| Flag | Description | Default |
|------|-------------|---------|
| `--retention` | Retention period in months (min: 1) | `12` |
| `--force` | Skip confirmation for >24 months | `false` |

### `finops version`

Prints the CLI version.

## AWS Permissions

The CLI requires the following IAM permissions:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "CostExplorer",
      "Effect": "Allow",
      "Action": [
        "ce:GetCostAndUsage",
        "ce:GetSavingsPlansUtilization",
        "ce:GetSavingsPlansCoverage",
        "ce:GetReservationUtilization",
        "ce:GetReservationCoverage"
      ],
      "Resource": "*"
    },
    {
      "Sid": "AccountDetection",
      "Effect": "Allow",
      "Action": [
        "sts:GetCallerIdentity",
        "organizations:DescribeOrganization",
        "organizations:ListAccounts"
      ],
      "Resource": "*"
    },
    {
      "Sid": "ResourceDiscovery",
      "Effect": "Allow",
      "Action": [
        "ec2:DescribeInstances",
        "ec2:DescribeVolumes",
        "ec2:DescribeNatGateways",
        "ec2:DescribeRegions",
        "rds:DescribeDBInstances",
        "s3:ListBuckets",
        "s3:GetBucketLocation",
        "lambda:ListFunctions",
        "ecs:ListClusters",
        "ecs:DescribeClusters",
        "eks:ListClusters",
        "eks:DescribeCluster",
        "elasticache:DescribeCacheClusters",
        "cloudfront:ListDistributions"
      ],
      "Resource": "*"
    }
  ]
}
```

**Notes:**
- Organization permissions (`organizations:*`) are optional — the CLI falls back to single-account mode if access is denied
- Resource discovery permissions are optional — costs are still recorded without them
- For Organizations: the management account (or a delegated admin) needs these permissions

## Discovered Resources

| AWS Service | Resource Types | Details Captured |
|-------------|---------------|------------------|
| EC2 | Instances, EBS Volumes | Instance type, state, spot detection |
| RDS | DB Instances | Engine, instance class, storage |
| S3 | Buckets | Name, region, tags |
| Lambda | Functions | Runtime, memory, code size |
| ECS/EKS | Clusters | Name, status, ARN |
| ElastiCache | Clusters | Engine, node type, node count |
| NAT Gateway | NAT Gateways | AZ, state |
| CloudFront | Distributions | Domain, status |

Services without a discovery adapter still have their costs recorded.

## Output Formats

### HTML (default)

Self-contained dark-mode HTML with embedded Chart.js charts. No external dependencies — works offline. Includes interactive collapsible sections, sortable tables, and number formatting with thousand separators.

### CSV

Standard CSV format for import into spreadsheets or data tools. Summary CSV includes Account ID, Service, Region, Total Cost, and Currency columns.

### JSON

Structured JSON with full data hierarchy — designed for programmatic consumption by scripts and AI agents. Includes all fields available in the HTML report.

### PDF (optional)

Converts the HTML report to PDF via headless Chrome. Requires Chrome or Chromium to be installed on your system. The CLI provides a clear error message if Chrome is not available.

## Data Storage

All data is stored locally in SQLite at `~/.finops/data.db`. The database is auto-created on first run.

| Table | Purpose |
|-------|---------|
| `cost_records` | Monthly cost data per service/region/account |
| `resources` | Discovered resources with specs and tags |
| `sync_history` | Audit trail of scan operations |
| `commitments` | Savings Plans and Reserved Instances data |
| `config` | CLI settings (retention, etc.) |

Data sync is idempotent — re-running a scan for the same period updates existing records without creating duplicates.

## Project Structure

```
finops-cli/
├── cmd/                          # CLI commands (cobra)
│   ├── root.go                   # Root command, global flags
│   ├── scan.go                   # finops scan
│   ├── report.go                 # finops report + subcommands
│   ├── db.go                     # finops db stats/prune
│   └── version.go                # finops version
├── internal/
│   ├── provider/                 # Cloud provider abstraction
│   │   ├── provider.go           # Provider interface
│   │   └── aws/                  # AWS implementation
│   │       ├── client.go         # AWS client factory
│   │       ├── costs.go          # Cost Explorer integration
│   │       ├── commitments.go    # Savings Plans / RI data
│   │       ├── organization.go   # Org detection
│   │       ├── regions.go        # Active region detection
│   │       └── discovery/        # Resource discovery adapters
│   ├── store/                    # SQLite persistence (sqlc generated)
│   ├── analysis/                 # Analysis engine
│   │   ├── summary.go            # Aggregations
│   │   ├── trends.go             # Trend calculations
│   │   ├── anomaly.go            # Z-score anomaly detection
│   │   ├── compare.go            # Period comparison
│   │   └── commitments.go        # Commitment analysis
│   └── report/                   # Report generation
│       ├── html.go               # HTML with Chart.js
│       ├── csv.go                # CSV export
│       ├── json.go               # JSON export
│       ├── pdf.go                # PDF via chromedp
│       └── templates/            # Embedded HTML templates
├── db/
│   ├── schema.sql                # Database schema
│   ├── queries/                  # SQL queries (sqlc input)
│   └── migrations/               # Schema migrations
├── main.go                       # Entry point
├── go.mod
├── Makefile
├── VERSION
├── .goreleaser.yaml
└── sqlc.yaml
```

## Development

### Prerequisites

- Go 1.25+
- [sqlc](https://docs.sqlc.dev/en/latest/overview/install.html) (for code generation)
- [golangci-lint](https://golangci-lint.run/welcome/install/) (for linting)
- [goreleaser](https://goreleaser.com/install/) (for releases)

### Build

```bash
make build       # Build binary to ./bin/finops
make test        # Run tests with race detector
make lint        # Run golangci-lint
make generate    # Regenerate sqlc code
make clean       # Remove build artifacts
make install     # Install to $GOPATH/bin
```

### Architecture

The codebase follows a **provider interface pattern** that keeps the core analysis and reporting logic independent of any specific cloud provider. See [ARCHITECTURE.md](ARCHITECTURE.md) for full details.

Key design decisions:
- **Pure Go SQLite** (`modernc.org/sqlite`) — no CGO, simplifies cross-compilation
- **sqlc** for type-safe database access — SQL queries generate Go code
- **Embedded assets** — Chart.js, HTML templates, and migrations are compiled into the binary
- **Provider abstraction** — designed for multi-cloud support (Azure, GCP planned for future versions)

### Release Process

1. Update the `VERSION` file with the new version
2. Commit and push to `main`
3. GitHub Actions automatically: runs tests, creates a git tag, builds binaries for all platforms via goreleaser, and publishes a GitHub Release

## Tech Stack

| Component | Technology |
|-----------|------------|
| Language | Go 1.25 |
| CLI | [cobra](https://github.com/spf13/cobra) |
| AWS | [aws-sdk-go-v2](https://github.com/aws/aws-sdk-go-v2) |
| Database | [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) + [sqlc](https://sqlc.dev) |
| Terminal UI | [lipgloss](https://github.com/charmbracelet/lipgloss) + [spinner](https://github.com/briandowns/spinner) |
| Charts | [Chart.js](https://www.chartjs.org/) (embedded in HTML) |
| PDF | [chromedp](https://github.com/chromedp/chromedp) (optional) |
| Testing | [testify](https://github.com/stretchr/testify) |
| Build | Makefile + [goreleaser](https://goreleaser.com) |

## License

[MIT](LICENSE) — Copyright (c) 2025 Helmcode
