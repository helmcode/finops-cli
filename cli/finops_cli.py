import os
import click
from cli.src.cost_calculator import EC2CostCalculator

# Set up colors for better output - Minimalist palette
class Colors:
    # Primary colors
    PRIMARY = '\033[38;5;39m'  # Soft blue
    SECONDARY = '\033[38;5;247m'  # Medium gray
    SUCCESS = '\033[38;5;34m'  # Soft green
    WARNING = '\033[38;5;208m'  # Orange
    DANGER = '\033[38;5;196m'  # Red

    # Text colors
    TEXT = '\033[38;5;250m'  # Light gray
    TEXT_BOLD = '\033[1;38;5;255m'  # White bold
    TEXT_MUTED = '\033[38;5;240m'  # Dark gray

    # Background colors
    BG_DARK = '\033[48;5;235m'  # Dark background

    # Utility
    RESET = '\033[0m'
    BOLD = '\033[1m'
    UNDERLINE = '\033[4m'

    # Aliases for compatibility
    HEADER = PRIMARY
    BLUE = PRIMARY
    CYAN = '\033[38;5;44m'
    GREEN = SUCCESS
    FAIL = DANGER
    ENDC = RESET

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
        aws-finops analyze

        # Analyze a specific region
        aws-finops analyze --region eu-west-1

        # Show potential Reserved Instance savings
        aws-finops analyze --show-reserved-savings

        # Show detailed instance information
        aws-finops analyze --detailed
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
            click.echo(f" AWS FINOPS COST ANALYSIS ".center(60, '='))
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
