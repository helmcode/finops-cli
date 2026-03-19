---
name: finops
description: Cloud FinOps CLI for cost analysis, anomaly detection, resource discovery, and optimization. Use when the user needs to analyze cloud costs, find cost anomalies, compare spending periods, discover cloud resources, generate cost reports, or perform any cloud financial operations task. Triggers on requests like "analyze AWS costs", "find cost spikes", "show top services by spend", "compare this month vs last month", "list cloud resources", "generate a cost report", "what am I spending on", "are there any cost anomalies", "show cost trends", or any cloud cost optimization task.
allowed-tools: Bash(finops), Bash(finops *)
---

# FinOps CLI

Cloud cost analysis and optimization tool. Scans AWS accounts for cost data and resources, stores everything locally in SQLite, and generates reports with trend analysis, anomaly detection, and period comparisons.

## Prerequisites

- The `finops` binary must be installed and available in PATH
- Valid AWS credentials configured (via environment variables, AWS profiles, or IAM roles)
- AWS permissions needed: `ce:GetCostAndUsage`, `sts:GetCallerIdentity`, `organizations:DescribeOrganization`, `organizations:ListAccounts`, and read-only access to EC2, RDS, S3, Lambda, ECS, EKS, ElastiCache, CloudFront for resource discovery

## Core Workflow

The CLI follows a **scan → report → analyze** pattern:

1. **Scan** — Download cost data and discover resources from AWS into local SQLite
2. **Report** — Generate reports from the local database (never calls AWS APIs)
3. **Analyze** — Interpret JSON output to provide insights and recommendations

Always use `--output json --file -` for report commands to get structured data on stdout.

```bash
# Step 1: Sync cost data (run once, then periodically)
finops scan --provider aws

# Step 2: Generate reports as JSON to stdout
finops report summary --output json --file -
finops report anomalies --output json --file -

# Step 3: Parse the JSON and reason about the data
```

## Command Reference

### `finops scan`

Downloads costs and discovers resources from cloud providers into local SQLite.

```bash
finops scan --provider aws [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--provider` | string | *required* | Cloud provider (`aws`) |
| `--region` | string | `all` | Region to scan (`all` or specific like `us-east-1`) |
| `--months` | int | `6` | Number of months to sync (1-12) |
| `--from` | string | — | Start date `YYYY-MM-DD` (overrides `--months`) |
| `--to` | string | — | End date `YYYY-MM-DD` (overrides `--months`) |
| `--account` | string | — | Filter accounts (comma-separated IDs, org mode only) |

**Behavior:**
- Auto-detects AWS Organizations — scans all member accounts if in org mode
- Incremental sync: first run fetches full history, subsequent runs fetch from last sync
- Permission errors on individual accounts are logged as warnings; scan continues
- Idempotent: safe to run multiple times (upserts via unique constraints)

**Resource discovery services:** EC2, EBS, RDS, S3, Lambda, ECS, EKS, ElastiCache, NAT Gateway, CloudFront.

### `finops report summary`

Full cost overview with totals, top services, cost by region, cost by account, commitment utilization, and resource counts.

```bash
finops report summary --output json --file -
```

### `finops report top-services`

Ranked list of services by total cost.

```bash
finops report top-services --output json --file - [--limit 10]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--limit` | int | `10` | Number of top services to show |

### `finops report trend`

Cost evolution over time with trend direction indicator.

```bash
finops report trend --output json --file - [--service "Amazon Elastic Compute Cloud"]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--service` | string | — | Filter by service name (empty = all services combined) |

### `finops report anomalies`

Detects cost spikes using z-score statistical analysis (threshold: |z| >= 2.0).

```bash
finops report anomalies --output json --file -
```

Severity levels: `high` (|z| >= 4.0), `medium` (|z| >= 3.0), `low` (|z| >= 2.0).

### `finops report compare`

Side-by-side comparison of two time periods with absolute and percentage changes.

```bash
finops report compare --output json --file - --current "2026-01-01:2026-03-01" --previous "2025-10-01:2025-12-01"
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--current` | string | *required* | Current period `YYYY-MM-DD:YYYY-MM-DD` |
| `--previous` | string | *required* | Previous period `YYYY-MM-DD:YYYY-MM-DD` |

### `finops report resources`

Lists discovered resources with spec details and associated service cost context.

```bash
finops report resources --output json --file - [--service "Amazon S3"] [--region "us-east-1"]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--service` | string | — | Filter by service name |
| `--region` | string | — | Filter by region |

### `finops db stats`

Shows database size, record counts, last sync date, and retention setting.

```bash
finops db stats
```

### `finops db prune`

Modify data retention and prune old records.

```bash
finops db prune --retention 6
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--retention` | int | `12` | Retention period in months (min: 1) |
| `--force` | bool | `false` | Skip confirmation for unusual retention (>24 months) |

### `finops version`

Print the CLI version.

```bash
finops version
```

### Global Flags

| Flag | Description |
|------|-------------|
| `--verbose` / `-v` | Enable verbose output (DEBUG log level) |

### Shared Report Flags

All `finops report *` subcommands accept:

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--output` | string | `html` | Output format: `html`, `csv`, `json`, `pdf` |
| `--file` | string | auto `/tmp` | Output file path. Use `-` for stdout |

## JSON Output Schemas

**IMPORTANT:** Always use `--output json --file -` to get JSON on stdout for parsing.

### Summary JSON

```json
{
  "report_type": "summary",
  "generated_at": "2026-03-19T20:07:45Z",
  "period": {
    "start": "2025-09-01",
    "end": "2026-03-01"
  },
  "total_spend": 39990.57,
  "currency": "USD",
  "active_services": 8,
  "resources_discovered": 150,
  "cost_by_account": [
    {
      "account_id": "123456789012",
      "total_amount": 15000.50,
      "currency": "USD",
      "percentage": 37.5,
      "resource_count": 75,
      "top_services": [
        { "service": "Amazon Elastic Compute Cloud", "amount": 8500.25 }
      ]
    }
  ],
  "monthly_spend": [
    { "period": "2025-09-01", "amount": 6200.00 },
    { "period": "2025-10-01", "amount": 6500.00 }
  ],
  "top_services": [
    {
      "service": "Amazon Elastic Compute Cloud",
      "total_amount": 18000.75,
      "currency": "USD",
      "percentage": 45.0
    }
  ],
  "cost_by_region": [
    {
      "region": "us-east-1",
      "total_amount": 12000.00,
      "currency": "USD",
      "resource_count": 50,
      "service_costs": [
        { "service": "Amazon Elastic Compute Cloud", "amount": 7500.00 }
      ],
      "resources": [
        {
          "service": "Amazon Elastic Compute Cloud",
          "resource_type": "Instance",
          "resource_id": "i-0abc123def456",
          "name": "web-server-1",
          "state": "running",
          "account_id": "123456789012"
        }
      ]
    }
  ],
  "commitments": {
    "total_committed": 10000.00,
    "total_used": 8500.00,
    "total_savings": 1500.00,
    "avg_utilization_pct": 85.0,
    "currency": "USD",
    "has_data": true,
    "permission_warning": false,
    "spot_instance_count": 5,
    "types": [
      {
        "type": "savings_plan",
        "total_commitment": 6000.00,
        "used_commitment": 5500.00,
        "on_demand_equivalent": 6200.00,
        "net_savings": 700.00
      },
      {
        "type": "reserved_instance",
        "total_commitment": 4000.00,
        "used_commitment": 3000.00,
        "on_demand_equivalent": 3800.00,
        "net_savings": 800.00
      }
    ]
  }
}
```

### Trend JSON

```json
{
  "report_type": "trend",
  "generated_at": "2026-03-19T20:07:45Z",
  "service": "Amazon Elastic Compute Cloud",
  "direction": "up",
  "data_points": [
    { "period": "2025-09-01", "amount": 2800.00 },
    { "period": "2025-10-01", "amount": 2950.00 },
    { "period": "2025-11-01", "amount": 3100.00 }
  ]
}
```

`direction` values: `"up"` (>5% increase), `"down"` (>5% decrease), `"flat"` (within 5%).

### Anomalies JSON

```json
{
  "report_type": "anomalies",
  "generated_at": "2026-03-19T20:07:45Z",
  "anomalies": [
    {
      "period": "2025-12-01",
      "service": "AWS Lambda",
      "expected": 500.00,
      "actual": 2000.00,
      "deviation": 3.5,
      "severity": "high"
    }
  ]
}
```

`severity` values: `"high"` (|z| >= 4.0), `"medium"` (|z| >= 3.0), `"low"` (|z| >= 2.0).

### Compare JSON

```json
{
  "report_type": "compare",
  "generated_at": "2026-03-19T20:07:45Z",
  "current_period": { "start": "2026-01-01", "end": "2026-03-01" },
  "previous_period": { "start": "2025-10-01", "end": "2025-12-01" },
  "total_current": 12500.00,
  "total_previous": 11000.00,
  "total_change": 1500.00,
  "total_change_pct": 13.6,
  "currency": "USD",
  "service_deltas": [
    {
      "service": "Amazon Elastic Compute Cloud",
      "previous_amount": 5000.00,
      "current_amount": 5800.00,
      "absolute_change": 800.00,
      "percent_change": 16.0,
      "currency": "USD"
    }
  ]
}
```

### Resources JSON

```json
{
  "report_type": "resources",
  "generated_at": "2026-03-19T20:07:45Z",
  "total_count": 47,
  "resources": [
    {
      "service": "Amazon Elastic Compute Cloud",
      "resource_type": "Instance",
      "resource_id": "i-0abc123def456",
      "name": "web-server-1",
      "region": "us-east-1",
      "state": "running",
      "account_id": "123456789012",
      "spec": "{\"InstanceType\":\"t3.medium\",\"VpcId\":\"vpc-abc123\"}",
      "tags": "{\"Environment\":\"production\",\"Team\":\"backend\"}"
    }
  ]
}
```

Note: `spec` and `tags` are JSON-encoded strings — parse them for structured data.

## Common Agent Patterns

### Full Cost Audit

Scan data, then generate all key reports to build a complete picture:

```bash
finops scan --provider aws
finops report summary --output json --file -
finops report anomalies --output json --file -
finops report top-services --output json --file - --limit 20
```

Parse the summary for total spend and distribution, check anomalies for spikes, and identify the highest-cost services.

### Month-over-Month Comparison

Compare the last two months to identify cost changes:

```bash
finops report compare --output json --file - \
  --current "2026-02-01:2026-03-01" \
  --previous "2026-01-01:2026-02-01"
```

Look at `service_deltas` for the biggest absolute and percentage changes.

### Investigate a Cost Spike

When anomalies are detected, drill into the specific service:

```bash
# 1. Detect anomalies
finops report anomalies --output json --file -

# 2. Check the trend for the flagged service
finops report trend --output json --file - --service "AWS Lambda"

# 3. List resources for that service
finops report resources --output json --file - --service "AWS Lambda"
```

### Resource Inventory by Region

Discover what resources exist in a specific region:

```bash
finops report resources --output json --file - --region "us-east-1"
```

### Multi-Account Analysis

Scan specific accounts in an AWS Organization:

```bash
finops scan --provider aws --account "111111111111,222222222222"
finops report summary --output json --file -
```

The summary JSON groups costs by account in `cost_by_account`.

### Commitment Utilization Check

The summary report includes commitment data (Savings Plans and Reserved Instances):

```bash
finops report summary --output json --file -
```

Check `commitments.avg_utilization_pct` — below 80% suggests over-provisioned commitments. Check `commitments.types` for per-type breakdown.

### Custom Date Range Scan

Scan a specific time window:

```bash
finops scan --provider aws --from "2025-06-01" --to "2025-12-31"
```

### Data Freshness Check

Verify the local database has data before generating reports:

```bash
finops db stats
```

If no data or stale: run `finops scan --provider aws` first.

## Important Notes

- **Reports are local-only.** They read from SQLite and never call AWS APIs. Only `finops scan` contacts AWS.
- **Run scan first.** Reports will error with "no data found" if the database is empty.
- **Incremental sync.** Subsequent scans only fetch new data since the last sync.
- **Service names are AWS Cost Explorer names.** Use the full name (e.g., `"Amazon Elastic Compute Cloud"`, not `"EC2"`). Get exact names from the summary report's `top_services` array.
- **Date format.** Dates use `YYYY-MM-DD`. Period comparisons use `YYYY-MM-DD:YYYY-MM-DD`.
- **Database location.** Data is stored at `~/.finops/data.db`.
- **Retention.** Default 12 months. Change with `finops db prune --retention N`.
- **Multi-account errors are non-fatal.** If some accounts lack permissions, scan continues with warnings.
- **Monetary values** are float64 rounded to 2 decimal places. Currency is always in the JSON.
- **`spec` and `tags` in resources** are JSON strings inside the JSON — parse them as nested JSON.
