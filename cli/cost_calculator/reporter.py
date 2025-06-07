from typing import List, Any, Callable
from decimal import Decimal

from cli.models import InstanceLifecycle, InstanceTypeCosts, CostSummary
from cli.ui import EC2CostReporter, Colors

class EC2CostReporterExtended(EC2CostReporter):
    """Extended reporter for EC2 cost analysis with additional formatting."""

    def __init__(self, region: str, use_colors: bool = True):
        """Initialize the reporter.

        Args:
            region: AWS region being analyzed
            use_colors: Whether to use ANSI color codes in the output
        """
        super().__init__(region=region, use_colors=use_colors)

    def _add_instance_cost_row(self, table_data: List[List[Any]], instance_type: str, 
                             instance_cost: InstanceTypeCosts, lifecycle_str: str, 
                             count: int, colorize: Callable) -> None:
        """Add a row to the instance cost table.

        Args:
            table_data: List to append the row data to
            instance_type: Type of the instance
            instance_cost: InstanceTypeCosts object with cost information
            lifecycle_str: Lifecycle type ('on-demand', 'reserved', 'spot')
            count: Number of instances
            colorize: Colorize function to apply colors
        """
        lifecycle_map = {
            'on-demand': (InstanceLifecycle.ON_DEMAND, '🔄 On-Demand', Colors.TEXT),
            'reserved': (InstanceLifecycle.RESERVED, '🔒 Reserved (40% off)', Colors.SUCCESS),
            'spot': (InstanceLifecycle.SPOT, '✨ Spot (30% off)', Colors.PRIMARY)
        }

        lifecycle_enum, lifecycle_display, lifecycle_color = lifecycle_map.get(
            lifecycle_str, (None, lifecycle_str.upper(), Colors.TEXT)
        )

        # Get the hourly price and convert to Decimal for calculations
        hourly_price = instance_cost.hourly_rate

        # Calculate costs using Decimal for precision
        monthly_cost = hourly_price * Decimal('730') * Decimal(str(count))
        annual_cost = monthly_cost * Decimal('12')

        # Convert to float only for display
        table_data.append([
            colorize(instance_type, Colors.TEXT_BOLD) if lifecycle_str == 'on-demand' else '',
            colorize(lifecycle_display, lifecycle_color),
            count,
            colorize(f"${float(hourly_price):.4f}", Colors.TEXT_BOLD),
            colorize(f"${float(monthly_cost):,.2f}", Colors.TEXT_BOLD),
            colorize(f"${float(annual_cost):,.2f}", Colors.TEXT_BOLD)
        ])

    def print_cost_report(self, summary: CostSummary, detailed: bool = True, 
                         show_reserved_savings: bool = False, use_colors: bool = True) -> None:
        """Print a formatted cost report to the console.

        Args:
            summary: Cost summary to report on
            detailed: Whether to show detailed instance information
            show_reserved_savings: Whether to show potential savings from Reserved Instances
            use_colors: Whether to use ANSI color codes in the output
        """
        # Set colors based on parameter
        self.use_colors = use_colors

        # Print header
        self._print_header()

        # Print instance details if requested
        if detailed and summary.instance_costs:
            self._print_instance_details(summary.instance_costs)

        # Print cost summary
        self._print_cost_summary(summary)

        # Print cost breakdown if requested
        if detailed and summary.instance_costs:
            self._print_cost_breakdown(summary.instance_costs)

        # Print reserved savings analysis if requested
        if show_reserved_savings:
            self.print_reserved_savings_analysis(summary)

        # Print footer
        self._print_footer()
