# AWS FinOps CLI Tool

[![Python Version](https://img.shields.io/badge/python-3.7%2B-blue.svg)](https://www.python.org/downloads/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Una herramienta de línea de comandos para análisis de costos de AWS, comenzando con EC2. Obtén información detallada sobre el gasto en instancias EC2, identifica oportunidades de optimización y genera reportes detallados.

**Características principales:**
- Listado detallado de instancias EC2 por región
- Cálculo de costos en tiempo real usando AWS Pricing API
- Análisis de costos por tipo de instancia
- Reportes detallados en formato de tabla
- Fácil de usar desde la línea de comandos

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Usage](#usage)
  - [List Instances](#list-instances)
  - [Cost Summary](#cost-summary)
- [Configuration](#configuration)
- [Development](#development)
- [Contributing](#contributing)
- [License](#license)

## Features

- 📊 **Real-time cost analysis** - Get current costs for all your EC2 instances
- 💰 **Cost breakdown** - View costs by instance type, daily, monthly, and yearly projections
- 📈 **Summary reports** - Aggregate costs by instance type with usage statistics
- 📁 **Export capabilities** - Export data to CSV or JSON for further analysis
- 🔍 **Flexible filtering** - Filter instances by tags, state, or other attributes
- 🚀 **Fast and efficient** - Uses AWS Pricing API with intelligent caching

## Installation

### Prerequisites

- Python 3.7 or higher
- AWS credentials configured (see [AWS Configuration](#aws-configuration))

### From PyPI (recommended)

```bash
pip install ec2-finops
```

### From source

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/ec2-finops.git
   cd ec2-finops
   ```

2. Create a virtual environment (recommended):
   ```bash
   python -m venv venv
   source venv/bin/activate  # On Windows use `venv\Scripts\activate`
   ```

3. Install the package in development mode:
   ```bash
   pip install -e .
   ```

### AWS Configuration

Make sure you have AWS credentials configured. You can use one of the following methods:

1. **AWS CLI** (recommended):
   ```bash
   aws configure
   ```

2. **Environment variables**:
   ```bash
   export AWS_ACCESS_KEY_ID=your_access_key_id
   export AWS_SECRET_ACCESS_KEY=your_secret_access_key
   export AWS_DEFAULT_REGION=us-east-1
   ```

3. **Configuration file** (`~/.aws/credentials`):
   ```ini
   [default]
   aws_access_key_id = your_access_key_id
   aws_secret_access_key = your_secret_access_key
   region = us-east-1
   ```

## Quick Start

1. List all running EC2 instances with cost information:
   ```bash
   ec2-finops instances
   ```

2. Get a cost summary by instance type:
   ```bash
   ec2-finops summary
   ```

3. Export instance data to CSV:
   ```bash
   ec2-finops instances --format csv --output instances.csv
   ```

## Usage

### List Instances

List all running EC2 instances with cost information:

```bash
ec2-finops instances [OPTIONS]
```

**Options:**
- `--region, -r TEXT`: AWS region (default: us-east-1 or from AWS_REGION env var)
- `--profile TEXT`: AWS profile name (default: from AWS_PROFILE env var)
- `--format, -f [table|csv|json]`: Output format (default: table)
- `--output, -o FILENAME`: Output file (for csv/json)
- `--filter, -t TEXT`: Filter by tag (e.g., Environment=production)

**Example:**
```bash
ec2-finops instances --region us-west-2 --format table
```

### Cost Summary

Show cost summary grouped by instance type:

```bash
ec2-finops summary [OPTIONS]
```

**Options:**
- `--region, -r TEXT`: AWS region (default: us-east-1 or from AWS_REGION env var)
- `--profile TEXT`: AWS profile name (default: from AWS_PROFILE env var)
- `--format, -f [table|json]`: Output format (default: table)
- `--output, -o FILENAME`: Output file

**Example:**
```bash
ec2-finops summary --region us-west-2 --format json
```

## Configuration

You can set default values using environment variables:

```bash
export AWS_REGION=us-west-2
export AWS_PROFILE=production
```

Or create a `.env` file in your project directory:
```env
AWS_REGION=us-west-2
AWS_PROFILE=production
```

## Development

### Setting Up for Development

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/yourusername/ec2-finops.git
   cd ec2-finops
   ```

3. Set up a virtual environment:
   ```bash
   python -m venv venv
   source venv/bin/activate  # On Windows use `venv\Scripts\activate`
   ```

4. Install development dependencies:
   ```bash
   pip install -e ".[dev]"
   ```

### Running Tests

```bash
pytest tests/
```

With coverage report:

```bash
pytest --cov=ec2_finops tests/
```

### Building the Package

```bash
python setup.py sdist bdist_wheel
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built with [Click](https://click.palletsprojects.com/) for the CLI interface
- Uses [Boto3](https://boto3.amazonaws.com/v1/documentation/api/latest/index.html) for AWS interactions
- Table formatting powered by [Tabulate](https://github.com/astanin/python-tabulate)

## Quick Start

### Prerequisites

1. **AWS Credentials**: Ensure you have AWS credentials configured. You can use:
   - AWS CLI profiles: `aws configure`
   - Environment variables: `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`
   - IAM roles (when running on EC2)

2. **Required IAM Permissions**:
   ```json
   {
     "Version": "2012-10-17",
     "Statement": [
       {
         "Effect": "Allow",
         "Action": [
           "ec2:DescribeInstances",
           "pricing:GetProducts"
         ],
         "Resource": "*"
       }
     ]
   }
   ```

### Basic Usage

1. **List all instances with costs**:
   ```bash
   ec2-finops instances --region us-east-2
   ```

2. **Get cost summary by instance type**:
   ```bash
   ec2-finops summary --region us-east-2
   ```

3. **Export to CSV**:
   ```bash
   ec2-finops export --format csv --output costs.csv --region us-east-2
   ```

4. **Generate comprehensive report**:
   ```bash
   ec2-finops report --region us-east-2
   ```

## Command Reference

### Global Options

- `--region, -r`: AWS region (default: us-east-1 or from AWS_REGION env var)
- `--profile, -p`: AWS profile name (default: from AWS_PROFILE env var)

### Commands

#### `instances`
List all running instances with detailed cost information.

```bash
ec2-finops instances [OPTIONS]
```

Options:
- `--format, -f`: Output format (table, csv, json) [default: table]
- `--output, -o`: Output file (optional)
- `--filter, -t`: Filter by tag (e.g., "Environment=production")

Example output:
```
┌─────────────────┬──────────┬────────────┬──────────┬────────────┬───────────┬─────────────┬──────────┐
│ Instance ID     │ Type     │ Name       │ Platform │ Hourly     │ Daily     │ Monthly     │ Total    │
├─────────────────┼──────────┼────────────┼──────────┼────────────┼───────────┼─────────────┼──────────┤
│ i-0abc123def    │ t2.micro │ web-server │ Linux    │ $0.0116    │ $0.28     │ $8.35       │ $145.32  │
│ i-0def456ghi    │ m5.large │ database   │ Linux    │ $0.0960    │ $2.30     │ $69.12      │ $1204.45 │
└─────────────────┴──────────┴────────────┴──────────┴────────────┴───────────┴─────────────┴──────────┘

Total Instances: 2
Total Daily Cost: $2.58
Total Monthly Cost: $77.47
```

#### `summary`
Show cost summary grouped by instance type.

```bash
ec2-finops summary [OPTIONS]
```

Options:
- `--format, -f`: Output format (table, json) [default: table]
- `--output, -o`: Output file (optional)

Example output:
```
┌──────────────┬───────┬────────────┬─────────────┬────────────┐
│ Instance Type│ Count │ Daily Cost │ Monthly Cost│ % of Total │
├──────────────┼───────┼────────────┼─────────────┼────────────┤
│ m5.large     │ 5     │ $11.52     │ $345.60     │ 45.2%      │
│ t2.micro     │ 12    │ $3.35      │ $100.51     │ 13.1%      │
│ c5.xlarge    │ 3     │ $12.24     │ $367.20     │ 41.7%      │
└──────────────┴───────┴────────────┴─────────────┴────────────┘
```

#### `export`
Export instance data to CSV or JSON file.

```bash
ec2-finops export [OPTIONS]
```

Options:
- `--format, -f`: Export format (csv, json) [default: csv]
- `--output, -o`: Output file (required)

Example:
```bash
ec2-finops export --format csv --output ec2_costs_20240115.csv
```

#### `report`
Generate a comprehensive cost report including:
- Cost summary
- Costs by instance type
- Top 10 most expensive instances
- Long-running instances (>30 days)

```bash
ec2-finops report [OPTIONS]
```

The report is displayed in the terminal and automatically saved to a timestamped CSV file.

## Advanced Usage

### Using with different AWS profiles

```bash
ec2-finops instances --profile production --region eu-west-1
```

### Filtering by tags

```bash
# Get costs for production environment only
ec2-finops instances --filter "Environment=production"

# Get costs for specific application
ec2-finops instances --filter "Application=web-api"
```

### Combining with other tools

```bash
# Use with jq for JSON processing
ec2-finops instances --format json | jq '.[] | select(.monthly_cost > 100)'

# Use with csvkit for CSV analysis
ec2-finops export -f csv -o costs.csv
csvstat costs.csv
```

### Automation examples

Create a daily cost report:
```bash
#!/bin/bash
# daily_cost_report.sh
DATE=$(date +%Y%m%d)
REGIONS=("us-east-1" "us-east-2" "eu-west-1")

for region in "${REGIONS[@]}"; do
    echo "Generating report for $region..."
    ec2-finops report --region $region
done
```

## Configuration

You can set default values using environment variables:

```bash
export AWS_REGION=us-east-2
export AWS_PROFILE=production
```

Or create a `.env` file in your project directory:
```env
AWS_REGION=us-east-2
AWS_PROFILE=production
```

## Troubleshooting

### Common Issues

1. **"No instances found"**
   - Check your AWS region is correct
   - Verify you have running instances
   - Check IAM permissions

2. **"Could not get price for instance type"**
   - Some instance types might not have pricing in all regions
   - Verify the region name mapping is correct
   - Check network connectivity to AWS Pricing API

3. **"Access Denied" errors**
   - Ensure your IAM user/role has the required permissions
   - Check if you're using the correct AWS profile

### Debug Mode

For debugging, you can enable verbose boto3 logging:
```python
import boto3
import logging
boto3.set_stream_logger('', logging.DEBUG)
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## Roadmap

- [ ] Support for Reserved Instances pricing comparison
- [ ] Savings Plans analysis
- [ ] Multi-region aggregated reports
- [ ] Historical cost tracking
- [ ] Slack/Email notifications for cost alerts
- [ ] Cost optimization recommendations
- [ ] Support for other AWS services (RDS, ELB, etc.)

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- Built with [Click](https://click.palletsprojects.com/) for the CLI interface
- Uses [Boto3](https://boto3.amazonaws.com/v1/documentation/api/latest/index.html) for AWS interactions
- Table formatting powered by [Tabulate](https://github.com/astanin/python-tabulate)

## Support

If you encounter any issues or have questions:

1. Check the [troubleshooting section](#troubleshooting)
2. Search existing [GitHub Issues](https://github.com/yourusername/ec2-finops/issues)
3. Create a new issue with detailed information about your problem

---

Made with ❤️ for the AWS FinOps community