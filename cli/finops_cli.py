import os
import click
from cli.cost_calculator.calculator import EC2CostCalculator
from cli.ui.colors import Colors

@click.group(invoke_without_command=True)
@click.version_option(
    prog_name=Colors.BOLD + 'AWS FinOps CLI' + Colors.ENDC,
    message=Colors.CYAN + '%(prog)s ' + Colors.GREEN + '%(version)s' + Colors.ENDC
)
@click.pass_context
def cli(ctx):
    """AWS FinOps CLI - Analyze and optimize your AWS costs.

    This tool helps you understand your EC2 instance costs and identify
    potential savings through Reserved Instances and other optimizations.

    By default, it analyzes the 'us-east-1' region unless you specify another
    region using the --region flag or set the AWS_DEFAULT_REGION environment variable.
    """
    if ctx.invoked_subcommand is None:
        click.echo(ctx.get_help())
        click.echo(f"\n{Colors.WARNING}Error:{Colors.ENDC} Missing command. Use 'aws-finops analyze --help' for usage.")
        ctx.exit(1)

@cli.group()
def export():
    """Export EC2 cost and instance data to various formats."""
    pass

@export.command('ec2-instances')
@click.option('--region',
              default=lambda: os.environ.get('AWS_DEFAULT_REGION', 'us-east-1'),
              show_default='from AWS_DEFAULT_REGION or us-east-1',
              help='AWS region to analyze (e.g., us-west-2, eu-west-1)')
@click.option('--output', '-o', 'output_file',
              help='Output file path (default: ec2_instances_<region>_<timestamp>.csv)')
@click.option('--no-color',
              is_flag=True,
              help='Disable colored output')
def export_instances(region, output_file, no_color):
    """Export EC2 instance details to CSV.

    This command exports detailed information about all EC2 instances in the specified
    region to a CSV file, including instance type, pricing model, and cost information.

    Examples:

        # Export instances to default CSV file
        finops export ec2-instances

        # Export instances to a specific file
        finops export ec2-instances --output my_instances.csv

        # Export instances from a specific region
        finops export ec2-instances --region eu-west-1
    """
    try:
        calculator = EC2CostCalculator(region)
        output_path = calculator.export_instances_to_csv(output_file)

        if not no_color:
            message = f"{Colors.SUCCESS}✓ Successfully exported instance data to {Colors.BOLD}{output_path}{Colors.ENDC}"
        else:
            message = f"Successfully exported instance data to {output_path}"

        click.echo(message)

    except Exception as e:
        error_msg = f"Error exporting instance data: {str(e)}"
        if not no_color:
            error_msg = f"{Colors.FAIL}{error_msg}{Colors.ENDC}"
        click.echo(error_msg, err=True)
        raise click.Abort()

@export.command('ec2-costs')
@click.option('--region',
              default=lambda: os.environ.get('AWS_DEFAULT_REGION', 'us-east-1'),
              show_default='from AWS_DEFAULT_REGION or us-east-1',
              help='AWS region to analyze (e.g., us-west-2, eu-west-1)')
@click.option('--output', '-o', 'output_file',
              help='Output file path (default: ec2_costs_<region>_<timestamp>.csv)')
@click.option('--no-color',
              is_flag=True,
              help='Disable colored output')
def export_costs(region, output_file, no_color):
    """Export EC2 cost breakdown to CSV.

    This command exports a cost breakdown by instance type to a CSV file,
    including hourly rates, monthly costs, and potential savings.

    Examples:

        # Export costs to default CSV file
        finops export ec2-costs

        # Export costs to a specific file
        finops export ec2-costs --output my_costs.csv
    """
    try:
        calculator = EC2CostCalculator(region)
        output_path = calculator.export_costs_to_csv(output_file)

        if not no_color:
            message = f"{Colors.SUCCESS}✓ Successfully exported cost data to {Colors.BOLD}{output_path}{Colors.ENDC}"
        else:
            message = f"Successfully exported cost data to {output_path}"

        click.echo(message)

    except Exception as e:
        error_msg = f"Error exporting cost data: {str(e)}"
        if not no_color:
            error_msg = f"{Colors.FAIL}{error_msg}{Colors.ENDC}"
        click.echo(error_msg, err=True)
        raise click.Abort()

@export.command('ec2-savings')
@click.option('--region',
              default=lambda: os.environ.get('AWS_DEFAULT_REGION', 'us-east-1'),
              show_default='from AWS_DEFAULT_REGION or us-east-1',
              help='AWS region to analyze (e.g., us-west-2, eu-west-1)')
@click.option('--output', '-o', 'output_file',
              help='Output file path (default: ec2_savings_<region>_<timestamp>.csv)')
@click.option('--no-color',
              is_flag=True,
              help='Disable colored output')
def export_savings(region, output_file, no_color):
    """Export EC2 savings analysis to CSV.

    This command exports a detailed analysis of potential savings from
    converting On-Demand instances to Reserved Instances.

    Examples:

        # Export savings analysis to default CSV file
        finops export ec2-savings

        # Export to a specific file
        finops export ec2-savings --output my_savings.csv
    """
    try:
        calculator = EC2CostCalculator(region)
        output_path = calculator.export_savings_to_csv(output_file)

        if not no_color:
            message = f"{Colors.SUCCESS}✓ Successfully exported savings analysis to {Colors.BOLD}{output_path}{Colors.ENDC}"
        else:
            message = f"Successfully exported savings analysis to {output_path}"

        click.echo(message)

    except Exception as e:
        error_msg = f"Error exporting savings analysis: {str(e)}"
        if not no_color:
            error_msg = f"{Colors.FAIL}{error_msg}{Colors.ENDC}"
        click.echo(error_msg, err=True)
        raise click.Abort()

@cli.command()
@click.option('--region',
              default=lambda: os.environ.get('AWS_DEFAULT_REGION', 'us-east-1'),
              show_default='from AWS_DEFAULT_REGION or us-east-1',
              help='AWS region to analyze (e.g., us-west-2, eu-west-1)')
@click.option('--show-reserved-savings', 'show_ri',
              is_flag=True,
              help='Show potential savings from converting On-Demand instances to Reserved Instances')
@click.option('--detailed',
              is_flag=True,
              help='Show detailed instance information')
@click.option('--no-color',
              is_flag=True,
              help='Disable colored output')
def analyze(region, show_ri, detailed, no_color):
    """Analyze EC2 instance costs and identify potential savings.

    This command provides a detailed cost analysis of your EC2 instances in the
    specified region, including hourly, monthly, and annual costs. It can also
    identify potential savings from using Reserved Instances.

    Examples:

        # Basic cost analysis for default region
        finops analyze

        # Analyze a specific region
        finops analyze --region eu-west-1

        # Show potential Reserved Instance savings
        finops analyze --show-reserved-savings

        # Show detailed instance information
        finops analyze --detailed
    """
    try:
        # Initialize calculator with region
        calculator = EC2CostCalculator(region)

        # Print header
        if not no_color:
            click.echo(f"{Colors.HEADER}{'='*60}{Colors.ENDC}")
            click.echo(f"{Colors.HEADER} AWS FINOPS COST ANALYSIS {Colors.ENDC}".center(60, '='))
            click.echo(f"{Colors.HEADER} Region: {Colors.BOLD}{region}{Colors.ENDC}".ljust(60, ' '))
            click.echo(f"{Colors.HEADER}{'='*60}{Colors.ENDC}")
        else:
            click.echo(f"="*60)
            click.echo(" AWS FINOPS COST ANALYSIS ".center(60, '='))
            click.echo(f" Region: {region}".ljust(60, ' '))
            click.echo(f"="*60)

        # Generate and display the report
        calculator.print_cost_report(
            show_reserved_savings=show_ri,
            detailed=detailed,
            use_colors=not no_color
        )

    except Exception as e:
        error_msg = f"Error analyzing costs: {str(e)}"
        if not no_color:
            error_msg = f"{Colors.FAIL}{error_msg}{Colors.ENDC}"
        click.echo(error_msg, err=True)
        raise click.Abort()

if __name__ == '__main__':
    cli()
