# FinOps CLI

FinOps CLI is a command-line tool for analyzing and optimizing costs.

## 📊 Features

- *Cost Analysis*: Get detailed cost breakdown by instance type and lifecycle
- *Reserved Instances*: Identify potential savings from Reserved Instances
- *Cost Optimization*: Get recommendations for cost optimization
- *Interactive Interface*: Colorful and user-friendly CLI output

## 🛠️ AWS Configuration

You can configure AWS credentials in two ways:

### Environment variables:

```bash
export AWS_ACCESS_KEY_ID=your_access_key_id
export AWS_SECRET_ACCESS_KEY=your_secret_access_key
export AWS_DEFAULT_REGION=us-east-1
```

### Or you can create a `~/.aws/credentials` file with the following content:

```ini
[default]
aws_access_key_id = your_access_key_id
aws_secret_access_key = your_secret_access_key
aws_region = us-east-1
```

## 📦 Installation

### From source
```bash
git clone https://github.com/helmcode/finops-cli.git
cd finops-cli
pip install -e .
```

### From PyPI (not available yet)
```bash
pip install finops-cli
```

## 📚 Usage

```bash
finops --help
```

### EC2 Cost analysis for specific region
```bash
finops analyze --region eu-west-1
```

### Estimate Reserved Instance savings
```bash
finops analyze --region eu-west-1 --show-reserved-savings
```

## 🤝 Contributing
Contributions are welcome! Please open an issue or submit a PR.

1. Fork the repository
2. Create your feature branch
3. Commit your changes
4. Push to the branch
5. Open a Pull Request

## 📄 License
MIT - See [LICENSE](LICENSE) for details.

## 👷 Support
If you encounter any issues or have questions:

1. Check the troubleshooting section
2. Search existing GitHub Issues
3. Create a new issue with detailed information about your problem

---
Made with ❤️ for the FinOps community