---
name: finops
description: Cloud FinOps CLI for cost analysis, anomaly detection, resource discovery, and optimization. Use when the user needs to analyze cloud costs, find cost anomalies, compare spending periods, discover cloud resources, generate cost reports, or perform any cloud financial operations task. Triggers on requests like "analyze AWS costs", "find cost spikes", "show top services by spend", "compare this month vs last month", "list cloud resources", "generate a cost report", "what am I spending on", "are there any cost anomalies", "show cost trends", or any cloud cost optimization task.
allowed-tools: Bash(finops), Bash(finops *)
---

# FinOps CLI

Cloud cost analysis and optimization tool. Scans AWS accounts (single or Organization with multiple accounts) for cost data and resources, stores everything locally in SQLite, and generates reports with trend analysis, anomaly detection, and period comparisons.

## Prerequisites

- The `finops` binary must be installed and available in PATH
- Valid AWS credentials configured (via environment variables, AWS profiles, or IAM roles)
- AWS permissions needed: `ce:GetCostAndUsage`, `sts:GetCallerIdentity`, `organizations:DescribeOrganization`, `organizations:ListAccounts`, and read-only access to EC2, RDS, S3, Lambda, ECS, EKS, ElastiCache, CloudFront for resource discovery

## Core Workflow

The CLI follows a **scan → report → analyze** pattern:

1. **Scan** — Download cost data and discover resources from AWS into local SQLite
2. **Report** — Generate reports from the local database (never calls AWS APIs)
3. **Analyze** — Interpret JSON output to provide insights and recommendations

**CRITICAL:** Always use `--output json --file -` for report commands, and pipe through `2>/dev/null | sed '/^Report saved to:/d'` to get clean JSON on stdout. The CLI prints a `Report saved to: -` status line to stdout after the JSON which will break JSON parsers if not stripped.

```bash
# Standard pattern for getting clean JSON from any report command:
finops report summary --output json --file - 2>/dev/null | sed '/^Report saved to:/d'
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
- Auto-detects AWS Organizations — scans all member accounts if in org mode. Works identically for single-account setups (scans only the current account).
- Incremental sync: first run fetches full history, subsequent runs fetch from last sync date.
- Permission errors on individual accounts in an organization are logged as warnings; scan continues with remaining accounts.
- Idempotent: safe to run multiple times (upserts via unique constraints).
- Discovers resources only for services with actual spend (zero-cost services are skipped).
- Use `-v` (verbose) to see per-account details including record counts and skipped accounts.

**Resource discovery services:** EC2 instances, EBS volumes, RDS databases, S3 buckets, Lambda functions, ECS/EKS clusters, ElastiCache clusters, NAT Gateways, CloudFront distributions.

**Services without resource discovery** (cost is tracked but no resource detail): CloudTrail, CodePipeline, Glue, KMS, CloudWatch, Route 53, SES, SNS, SQS, WAF, API Gateway, OpenSearch, QuickSight, Bedrock, DynamoDB, Redshift, Athena, Step Functions, Elastic Load Balancing, and others.

### `finops report summary`

Full cost overview with totals, top services, cost by region, cost by account, commitment utilization, and resource counts.

```bash
finops report summary --output json --file - 2>/dev/null | sed '/^Report saved to:/d'
```

### `finops report top-services`

Ranked list of top N services by total cost. Returns the same JSON structure as `summary` but with only `top_services` populated — `cost_by_account`, `monthly_spend`, and `cost_by_region` will be `null`.

```bash
finops report top-services --output json --file - --limit 20 2>/dev/null | sed '/^Report saved to:/d'
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--limit` | int | `10` | Number of top services to show |

### `finops report trend`

Cost evolution over time with trend direction indicator.

```bash
finops report trend --output json --file - 2>/dev/null | sed '/^Report saved to:/d'
finops report trend --output json --file - --service "Amazon Relational Database Service" 2>/dev/null | sed '/^Report saved to:/d'
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--service` | string | — | Filter by service name (empty = all services combined) |

**Note:** If the service name does not match any data, the response will have `"data_points": null` (not an empty array) and `"direction": "flat"`. Always use the exact service name from `top_services` output.

### `finops report anomalies`

Detects cost spikes using z-score statistical analysis (threshold: |z| >= 2.0).

```bash
finops report anomalies --output json --file - 2>/dev/null | sed '/^Report saved to:/d'
```

Severity levels: `high` (|z| >= 4.0), `medium` (|z| >= 3.0), `low` (|z| >= 2.0). Negative deviations indicate unexpected drops; positive indicate spikes.

### `finops report compare`

Side-by-side comparison of two time periods with absolute and percentage changes.

**Known issue (v0.1.x):** The date range separator `:` conflicts with Unix path separator on macOS/Linux, causing parse failures. Until fixed, use the `trend` report to compare periods visually, or use `summary` data to calculate differences manually.

```bash
finops report compare --output json --file - --current "2026-01-01:2026-03-01" --previous "2025-10-01:2025-12-01" 2>/dev/null | sed '/^Report saved to:/d'
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--current` | string | *required* | Current period `YYYY-MM-DD:YYYY-MM-DD` |
| `--previous` | string | *required* | Previous period `YYYY-MM-DD:YYYY-MM-DD` |

### `finops report resources`

Lists discovered resources with spec details and associated service cost context. Supports combined filters.

```bash
finops report resources --output json --file - 2>/dev/null | sed '/^Report saved to:/d'
finops report resources --output json --file - --service "AWS Lambda" --region "eu-west-1" 2>/dev/null | sed '/^Report saved to:/d'
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--service` | string | — | Filter by service name |
| `--region` | string | — | Filter by region |

### `finops db stats`

Shows database size, record counts, last sync date, and retention setting. Use this to verify data freshness before generating reports.

```bash
finops db stats
```

Example output:
```
Database Statistics

Size:                288.00 KB
Cost records:        709
Resources:           258
Sync history:        12
Last sync:           2026-03-19T21:25:48Z
Retention:           12 (default) months
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

**CRITICAL:** Always use `--output json --file - 2>/dev/null | sed '/^Report saved to:/d'` to get clean, parseable JSON on stdout.

### Summary JSON

```json
{
  "report_type": "summary",
  "generated_at": "2026-03-19T21:26:02Z",
  "period": {
    "start": "2025-09-01",
    "end": "2026-03-01"
  },
  "total_spend": 85459.92,
  "currency": "USD",
  "active_services": 39,
  "resources_discovered": 258,
  "cost_by_account": [
    {
      "account_id": "557981700545",
      "total_amount": 59978.97,
      "currency": "USD",
      "percentage": 70.2,
      "resource_count": 258,
      "top_services": [
        { "service": "Amazon Relational Database Service", "amount": 10396.41 },
        { "service": "Amazon OpenSearch Service", "amount": 7700.55 }
      ]
    }
  ],
  "monthly_spend": [
    { "period": "2025-09-01", "amount": 11697.77 },
    { "period": "2025-10-01", "amount": 14529.19 }
  ],
  "top_services": [
    {
      "service": "Amazon Relational Database Service",
      "total_amount": 14572.27,
      "currency": "USD",
      "percentage": 17.1
    }
  ],
  "cost_by_region": [
    {
      "region": "eu-west-1",
      "total_amount": 52000.00,
      "currency": "USD",
      "resource_count": 195,
      "service_costs": [
        { "service": "Amazon Relational Database Service", "amount": 14000.00 }
      ],
      "resources": [
        {
          "service": "Amazon Relational Database Service",
          "resource_type": "rds:db",
          "resource_id": "arn:aws:rds:eu-west-1:557981700545:db:rds-app-pro",
          "name": "rds-app-pro",
          "account_id": "557981700545"
        }
      ]
    }
  ],
  "commitments": {
    "total_committed": 3258.00,
    "total_used": 3257.99,
    "total_savings": 0,
    "avg_utilization_pct": 100,
    "currency": "USD",
    "has_data": true,
    "spot_instance_count": 0,
    "types": [
      {
        "type": "savings_plan",
        "total_commitment": 3258.00,
        "used_commitment": 3257.99,
        "on_demand_equivalent": 6995.14,
        "net_savings": 0
      },
      {
        "type": "reserved_instance",
        "total_commitment": 0,
        "used_commitment": 0,
        "on_demand_equivalent": 70622.58,
        "net_savings": 0
      }
    ]
  }
}
```

**Key notes:**
- `cost_by_region[].resources[]` has a **reduced field set** compared to the resources report: only `service`, `resource_type`, `resource_id`, `name`, `account_id` (no `state`, `spec`, `tags`, `region`).
- `commitments` is present when commitment data exists. `has_data` indicates if there is any commitment information.
- `top_services` within `cost_by_account` shows the top 5 services per account.

### Top-Services JSON

Uses the **same schema as Summary** but only `top_services` is populated. Fields `cost_by_account`, `monthly_spend`, and `cost_by_region` will be `null`:

```json
{
  "report_type": "summary",
  "generated_at": "2026-03-19T21:26:04Z",
  "period": { "start": "2025-09-01", "end": "2026-03-01" },
  "total_spend": 85459.92,
  "currency": "USD",
  "active_services": 5,
  "resources_discovered": 0,
  "cost_by_account": null,
  "monthly_spend": null,
  "top_services": [
    { "service": "Tax", "total_amount": 15450.32, "currency": "USD", "percentage": 18.1 },
    { "service": "Amazon Relational Database Service", "total_amount": 14572.27, "currency": "USD", "percentage": 17.1 }
  ],
  "cost_by_region": null
}
```

### Trend JSON

```json
{
  "report_type": "trend",
  "generated_at": "2026-03-19T21:26:04Z",
  "service": "Amazon Relational Database Service",
  "direction": "down",
  "data_points": [
    { "period": "2025-09-01", "amount": 2000.71 },
    { "period": "2025-10-01", "amount": 2307.75 },
    { "period": "2025-11-01", "amount": 2403.52 },
    { "period": "2025-12-01", "amount": 3125.51 },
    { "period": "2026-01-01", "amount": 2581.33 },
    { "period": "2026-02-01", "amount": 2153.44 }
  ]
}
```

- `direction`: `"up"` (>5% increase last month), `"down"` (>5% decrease), `"flat"` (within 5%).
- `service`: empty string `""` when no `--service` filter is applied (shows all combined).
- `data_points`: `null` (not `[]`) when the service name doesn't match any data.

### Anomalies JSON

```json
{
  "report_type": "anomalies",
  "generated_at": "2026-03-19T21:26:03Z",
  "anomalies": [
    {
      "period": "2026-02-01",
      "service": "AWS Lambda",
      "expected": 1.35,
      "actual": 5.44,
      "deviation": 2.14,
      "severity": "low"
    },
    {
      "period": "2025-09-01",
      "service": "Amazon Elastic Compute Cloud - Compute",
      "expected": 857.6,
      "actual": 323.43,
      "deviation": -2.0,
      "severity": "low"
    }
  ]
}
```

- `severity`: `"high"` (|z| >= 4.0), `"medium"` (|z| >= 3.0), `"low"` (|z| >= 2.0).
- `deviation` can be **negative** (unexpected cost drop) or **positive** (cost spike). Both are anomalies.
- `expected` is the historical mean cost for that service.

### Compare JSON

```json
{
  "report_type": "compare",
  "generated_at": "2026-03-19T20:07:45Z",
  "current_period": { "start": "2026-01-01", "end": "2026-03-01" },
  "previous_period": { "start": "2025-10-01", "end": "2025-12-01" },
  "total_current": 29410.86,
  "total_previous": 28929.97,
  "total_change": 480.89,
  "total_change_pct": 1.7,
  "currency": "USD",
  "service_deltas": [
    {
      "service": "Amazon Relational Database Service",
      "previous_amount": 5529.03,
      "current_amount": 4734.77,
      "absolute_change": -794.26,
      "percent_change": -14.4,
      "currency": "USD"
    }
  ]
}
```

### Resources JSON

```json
{
  "report_type": "resources",
  "generated_at": "2026-03-19T21:26:43Z",
  "total_count": 7,
  "resources": [
    {
      "service": "Amazon Relational Database Service",
      "resource_type": "rds:db",
      "resource_id": "arn:aws:rds:eu-west-1:557981700545:db:rds-app-pro",
      "name": "rds-app-pro",
      "region": "eu-west-1",
      "state": "available",
      "account_id": "557981700545",
      "spec": "{\"engine\":\"postgres\",\"engine_version\":\"17.7\",\"instance_class\":\"db.r8g.large\",\"multi_az\":true,\"storage_encrypted\":false,\"storage_gb\":100}",
      "tags": "{}"
    },
    {
      "service": "AWS Lambda",
      "resource_type": "lambda:function",
      "resource_id": "arn:aws:lambda:eu-west-1:557981700545:function:my-function",
      "name": "my-function",
      "region": "eu-west-1",
      "account_id": "557981700545",
      "spec": "{\"code_size\":3137,\"handler\":\"handler.lambda_handler\",\"memory_mb\":128,\"runtime\":\"python3.12\",\"timeout_s\":63}",
      "tags": "{}"
    }
  ]
}
```

**Key notes:**
- `state` is **optional/null** — only present for EC2 (`running`/`stopped`), RDS (`available`), CloudFront (`enabled`/`disabled`), ElastiCache (`available`). Lambda, S3, and NAT Gateways do not have state.
- `spec` and `tags` are **JSON-encoded strings** — parse them as nested JSON for structured data.
- `spec` content varies by resource type (see Resource Types below).
- `tags` is often `"{}"` (empty JSON object string) when no tags are set.

## Resource Types and Spec Fields

| resource_type | spec fields |
|---|---|
| `ec2:instance` | `instance_type`, `architecture`, `platform`, `vpc_id`, `state` |
| `ec2:volume` | `volume_type`, `size_gb`, `iops`, `encrypted`, `state` |
| `ec2:nat-gateway` | `state`, `vpc_id`, `subnet_id` |
| `rds:db` | `engine`, `engine_version`, `instance_class`, `multi_az`, `storage_encrypted`, `storage_gb` |
| `s3:bucket` | `creation_date` |
| `lambda:function` | `runtime`, `memory_mb`, `timeout_s`, `handler`, `code_size` |
| `elasticache:cluster` | `engine`, `engine_version`, `cache_node_type`, `num_cache_nodes` |
| `cloudfront:distribution` | `status`, `domain_name`, `price_class`, `http_version` |

## AWS Service Names Reference

Service names in the CLI are **exact Cost Explorer names**. Use the full name when filtering. Common services:

| Short name | Full Cost Explorer name |
|---|---|
| EC2 Compute | `Amazon Elastic Compute Cloud - Compute` |
| EC2 Other | `EC2 - Other` |
| RDS | `Amazon Relational Database Service` |
| S3 | `Amazon Simple Storage Service` |
| CloudFront | `Amazon CloudFront` |
| Lambda | `AWS Lambda` |
| EKS | `Amazon Elastic Container Service for Kubernetes` |
| ElastiCache | `Amazon ElastiCache` |
| OpenSearch | `Amazon OpenSearch Service` |
| ELB | `Amazon Elastic Load Balancing` |
| VPC | `Amazon Virtual Private Cloud` |
| WAF | `AWS WAF` |
| CloudWatch | `AmazonCloudWatch` |
| DynamoDB | `Amazon DynamoDB` |
| Route 53 | `Amazon Route 53` |
| KMS | `AWS Key Management Service` |
| SQS | `Amazon Simple Queue Service` |
| Savings Plans | `Savings Plans for AWS Compute usage` |
| Tax | `Tax` |

**Tip:** Always run `finops report top-services --output json --file - --limit 50` first to discover the exact service names available in the dataset, then use those names for `--service` filters.

## Common Agent Patterns

### Full Cost Audit

Scan data, then generate key reports for a complete picture:

```bash
finops scan --provider aws
finops report summary --output json --file - 2>/dev/null | sed '/^Report saved to:/d'
finops report anomalies --output json --file - 2>/dev/null | sed '/^Report saved to:/d'
finops report top-services --output json --file - --limit 20 2>/dev/null | sed '/^Report saved to:/d'
```

Parse the summary for total spend and distribution, check anomalies for spikes, and identify the highest-cost services.

### Investigate a Cost Spike

When anomalies are detected, drill into the specific service:

```bash
# 1. Detect anomalies
finops report anomalies --output json --file - 2>/dev/null | sed '/^Report saved to:/d'

# 2. Check the trend for the flagged service (use exact name from anomaly output)
finops report trend --output json --file - --service "AWS Lambda" 2>/dev/null | sed '/^Report saved to:/d'

# 3. List resources for that service to identify what's running
finops report resources --output json --file - --service "AWS Lambda" 2>/dev/null | sed '/^Report saved to:/d'
```

### Resource Right-Sizing Analysis

Find potentially over-provisioned resources:

```bash
# Get all EC2 instances
finops report resources --output json --file - --service "Amazon Elastic Compute Cloud - Compute" 2>/dev/null | sed '/^Report saved to:/d'

# Get all RDS databases
finops report resources --output json --file - --service "Amazon Relational Database Service" 2>/dev/null | sed '/^Report saved to:/d'
```

Parse `spec` JSON strings to analyze instance types, memory allocation, and storage. Cross-reference with cost data from the summary to identify high-cost, potentially over-provisioned resources.

### Stopped/Idle Resource Detection

```bash
finops report resources --output json --file - --service "Amazon Elastic Compute Cloud - Compute" 2>/dev/null | sed '/^Report saved to:/d'
```

Filter resources where `state` is `"stopped"` — these incur EBS storage costs without providing compute value. Also check for EBS volumes with `state: "available"` (unattached volumes).

### Multi-Account Cost Breakdown

For AWS Organizations, the summary groups costs by account:

```bash
finops scan --provider aws
finops report summary --output json --file - 2>/dev/null | sed '/^Report saved to:/d'
```

Analyze `cost_by_account` to identify which accounts drive the most spend, and `cost_by_account[].top_services` for per-account service breakdown.

### Commitment Utilization Check

```bash
finops report summary --output json --file - 2>/dev/null | sed '/^Report saved to:/d'
```

Check `commitments`:
- `avg_utilization_pct` below 80% → over-provisioned commitments, money being wasted
- `avg_utilization_pct` at 100% → fully utilized, potentially room for more savings
- Compare `on_demand_equivalent` vs `total_commitment` per type to quantify actual savings

### Regional Cost Analysis

```bash
finops report summary --output json --file - 2>/dev/null | sed '/^Report saved to:/d'
```

Analyze `cost_by_region` with `service_costs` breakdown to identify:
- Regions with unexpectedly high costs
- Services running in expensive regions that could be relocated
- Resource sprawl across too many regions

### Data Freshness Check

Always verify data is available before generating reports:

```bash
finops db stats
```

If `Cost records: 0` or `Last sync` is stale, run `finops scan --provider aws` first.

## Error Handling

| Error | Meaning | Resolution |
|---|---|---|
| `required flag(s) "provider" not set` | Missing `--provider` | Add `--provider aws` |
| `unsupported provider "X"` | Invalid provider | Only `aws` is supported |
| `--months must be between 1 and 12` | Invalid months range | Use 1-12 |
| `no data found. Run 'finops scan --provider aws' first` | Empty database | Run scan first |
| `invalid --current: expected format YYYY-MM-DD:YYYY-MM-DD` | Compare date parse error (known bug on macOS/Linux) | See compare command notes |
| Access denied on specific account | IAM permissions | Scan continues with other accounts; check verbose output |

Errors return exit code 1. Success returns exit code 0 (even with empty results).

## Important Notes

- **Reports are local-only.** They read from SQLite and never call AWS APIs. Only `finops scan` contacts AWS.
- **Run scan first.** Reports will error with "no data found" if the database is empty.
- **Incremental sync.** Subsequent scans only fetch new data since the last sync date.
- **Service names are exact AWS Cost Explorer names.** Use `top-services` to discover the exact names before using `--service` filters.
- **Date format.** Dates use `YYYY-MM-DD`. Report period defaults to last 6 months.
- **Database location.** Data is stored at `~/.finops/data.db`.
- **Retention.** Default 12 months. Change with `finops db prune --retention N`.
- **Multi-account errors are non-fatal.** If some accounts lack permissions, scan continues with warnings and partial data.
- **Monetary values** are float64 rounded to 2 decimal places. Currency is always in the JSON output.
- **`spec` and `tags` in resources** are JSON-encoded strings inside the JSON — they must be parsed as nested JSON.
- **Null vs empty arrays.** When no data matches a filter, arrays may be `null` instead of `[]`. Always handle both cases.
- **`Report saved to: -`** is always printed to stdout. Use `sed '/^Report saved to:/d'` to strip it for clean JSON parsing.
