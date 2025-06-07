from decimal import Decimal
from typing import Dict
from tabulate import tabulate
from ..models.cost_models import CostSummary, InstanceTypeCosts
from .colors import Colors


class EC2CostReporter:
    """Handles the presentation of EC2 cost information."""

    def __init__(self, region: str, use_colors: bool = True):
        """Initialize the EC2CostReporter.

        Args:
            region: AWS region being reported on
            use_colors: Whether to use ANSI color codes in the output
        """
        self.region = region
        self.use_colors = use_colors

    def colorize(self, text: str, color: str) -> str:
        """Apply color to text if use_colors is True.

        Args:
            text: Text to colorize
            color: Color code to apply

        Returns:
            Colored text if use_colors is True, else original text
        """
        return Colors.colorize(text, color, self.use_colors)

    def print_cost_report(self, summary: CostSummary) -> None:
        """Print a detailed cost report.

        Args:
            summary: The cost summary to report on
        """
        self._print_header()
        self._print_instance_details(summary.instance_costs)
        self._print_cost_summary(summary)
        self._print_cost_breakdown(summary.instance_costs)
        self.print_reserved_savings_analysis(summary)
        self._print_footer()

    def _print_header(self) -> None:
        """Print the report header."""
        title = "EC2 INSTANCE COST REPORT"
        print(f"\n{self.colorize('='*60, Colors.SECONDARY)}")
        print(self.colorize(title.center(60), Colors.TEXT_BOLD + Colors.UNDERLINE))
        print(f"{self.colorize('='*60, Colors.SECONDARY)}\n")

    def _print_instance_details(self, instance_costs: Dict[str, InstanceTypeCosts]) -> None:
        """Print details about each instance type.

        Args:
            instance_costs: Dictionary mapping instance types to their cost data
        """
        if not instance_costs:
            print(self.colorize("No instance data available.", Colors.WARNING))
            return

        print(self.colorize("INSTANCE DETAILS:", Colors.TEXT_BOLD))

        # Prepare table data
        table_data = []
        for instance_type, costs in sorted(instance_costs.items()):
            table_data.append([
                self.colorize(instance_type, Colors.TEXT_BOLD),
                costs.on_demand_count,
                costs.spot_count,
                costs.reserved_count,
                f"${float(costs.hourly_rate):.4f}",
                f"${float(costs.monthly_cost):,.2f}"
            ])

        # Print table
        headers = [
            self.colorize("TYPE", Colors.TEXT_MUTED + Colors.UNDERLINE),
            self.colorize("ON-DEMAND", Colors.TEXT_MUTED + Colors.UNDERLINE),
            self.colorize("SPOT", Colors.TEXT_MUTED + Colors.UNDERLINE),
            self.colorize("RESERVED", Colors.TEXT_MUTED + Colors.UNDERLINE),
            self.colorize("HOURLY RATE", Colors.TEXT_MUTED + Colors.UNDERLINE),
            self.colorize("MONTHLY COST", Colors.TEXT_MUTED + Colors.UNDERLINE)
        ]

        print(tabulate(table_data, headers=headers, tablefmt="simple_grid"))
        print()  # Add spacing

    def _print_cost_summary(self, summary: CostSummary) -> None:
        """Print a summary of costs.

        Args:
            summary: The cost summary to display
        """
        print(self.colorize("COST SUMMARY:", Colors.TEXT_BOLD))

        # Format costs with 2 decimal places and thousand separators
        total_instances = sum(cost.total_instances for cost in summary.instance_costs.values())

        print(f"{self.colorize('•', Colors.SECONDARY)} {self.colorize('Total Instances:', Colors.TEXT)} "
              f"{self.colorize(str(total_instances), Colors.TEXT_BOLD)}")

        print(f"{self.colorize('•', Colors.SECONDARY)} {self.colorize('Total Monthly Cost:', Colors.TEXT)} "
              f"{self.colorize(f'${float(summary.total_monthly_cost):,.2f}', Colors.TEXT_BOLD)}")

        print(f"{self.colorize('•', Colors.SECONDARY)} {self.colorize('Total Annual Cost:', Colors.TEXT)} "
              f"{self.colorize(f'${float(summary.total_annual_cost):,.2f}', Colors.TEXT_BOLD)}")

        # Add a note about the cost calculation
        print(f"\n{self.colorize('ℹ', Colors.SECONDARY)} "
              f"{self.colorize('Note:', Colors.TEXT_BOLD)} "
              f"{self.colorize('Costs are estimates based on 730 hours/month and 8760 hours/year.', Colors.TEXT_MUTED)}")

        print()  # Add spacing

    def _print_cost_breakdown(self, instance_costs: Dict[str, InstanceTypeCosts]) -> None:
        """Print a breakdown of costs by instance type.

        Args:
            instance_costs: Dictionary mapping instance types to their cost data
        """
        if not instance_costs:
            return

        print(self.colorize("COST BREAKDOWN BY INSTANCE TYPE:", Colors.TEXT_BOLD))

        # Calculate total monthly cost for percentage calculations
        total_monthly_cost = sum(cost.monthly_cost for cost in instance_costs.values())

        # Prepare table data
        table_data = []
        for instance_type, costs in sorted(instance_costs.items(),
                                         key=lambda x: x[1].monthly_cost,
                                         reverse=True):
            if costs.monthly_cost == 0:
                continue

            percentage = (costs.monthly_cost / total_monthly_cost) * 100

            # Determine color based on percentage
            if percentage > 20:
                cost_color = Colors.DANGER
            elif percentage > 10:
                cost_color = Colors.WARNING
            else:
                cost_color = Colors.SUCCESS

            table_data.append([
                self.colorize(instance_type, Colors.TEXT_BOLD),
                f"${float(costs.monthly_cost):,.2f}",
                self.colorize(f"{percentage:.1f}%", cost_color)
            ])

        # Print table
        headers = [
            self.colorize("INSTANCE TYPE", Colors.TEXT_MUTED + Colors.UNDERLINE),
            self.colorize("MONTHLY COST", Colors.TEXT_MUTED + Colors.UNDERLINE),
            self.colorize("% OF TOTAL", Colors.TEXT_MUTED + Colors.UNDERLINE)
        ]

        print(tabulate(table_data, headers=headers, tablefmt="simple_grid"))
        print()  # Add spacing

    def print_reserved_savings_analysis(self, summary: CostSummary) -> None:
        """Print an analysis of potential savings from Reserved Instances.

        Args:
            summary: The cost summary to analyze
        """
        if not summary or not summary.instance_costs:
            return

        # Header with emojis and minimalist style
        print(f"\n{self.colorize('='*60, Colors.SECONDARY)}")
        print(self.colorize("💰  RESERVED INSTANCE SAVINGS ANALYSIS  💰".center(60),
                            Colors.TEXT_BOLD + Colors.BOLD))
        print(self.colorize('='*60, Colors.SECONDARY))
        print(f"\n{self.colorize('ℹ️  This analysis shows potential savings from converting On-Demand to Reserved Instances', Colors.TEXT)}")
        print(f"{self.colorize('   Savings are calculated using the', Colors.TEXT)} {self.colorize('1-year No Upfront', Colors.TEXT_BOLD)} {self.colorize('payment option', Colors.TEXT)} "
              f"({self.colorize('40% off on-demand', Colors.SUCCESS)})\n")

        total_reserved_savings = Decimal('0')
        total_instances = 0
        instance_data = []

        # Process instance data
        for instance_type, instance_cost in summary.instance_costs.items():
            ondemand_count = instance_cost.on_demand_count
            if ondemand_count == 0:
                continue

            hourly_price = instance_cost.hourly_rate
            reserved_hourly_price = hourly_price * Decimal('0.6')  # 40% off for reserved

            monthly_hours = Decimal('730')  # 24 hours * 365 days / 12 months
            total_ondemand_equivalent = hourly_price * Decimal(str(ondemand_count)) * monthly_hours
            total_reserved_cost = reserved_hourly_price * Decimal(str(ondemand_count)) * monthly_hours
            reserved_savings = total_ondemand_equivalent - total_reserved_cost
            total_reserved_savings += reserved_savings
            total_instances += ondemand_count

            savings_pct = float((reserved_savings / total_ondemand_equivalent * 100) if total_ondemand_equivalent > 0 else 0)
            savings_color = Colors.SUCCESS if savings_pct > 20 else Colors.WARNING
            savings_emoji = "💸" if savings_pct > 20 else "📉"

            instance_data.append([
                self.colorize(instance_type, Colors.TEXT_BOLD),
                ondemand_count,
                f"${float(hourly_price):.4f}",
                f"${float(reserved_hourly_price):.4f}",
                f"{savings_emoji} {self.colorize(f'${reserved_savings:,.2f}', savings_color)}",
                self.colorize(f"{savings_pct:.1f}%", savings_color)
            ])

        # Print table with instance details
        if instance_data:
            headers = [
                self.colorize("INSTANCE TYPE", Colors.TEXT_MUTED + Colors.UNDERLINE),
                self.colorize("COUNT", Colors.TEXT_MUTED + Colors.UNDERLINE),
                self.colorize("ON-DEMAND RATE", Colors.TEXT_MUTED + Colors.UNDERLINE),
                self.colorize("RESERVED RATE", Colors.TEXT_MUTED + Colors.UNDERLINE),
                self.colorize("MONTHLY SAVINGS", Colors.TEXT_MUTED + Colors.UNDERLINE),
                self.colorize("SAVINGS %", Colors.TEXT_MUTED + Colors.UNDERLINE)
            ]
            print(tabulate(instance_data, headers=headers, tablefmt="simple_grid"))

        # Print total potential savings
        if total_instances > 0:
            # Get the total monthly cost from the summary object
            total_monthly_cost = float(summary.total_monthly_cost)

            # Convert total_monthly_cost to Decimal for calculations
            total_monthly_cost_dec = Decimal(str(total_monthly_cost))

            # Calculate the percentage of savings using Decimal
            total_ondemand_equivalent = total_reserved_savings + (total_monthly_cost_dec - total_reserved_savings)
            if total_ondemand_equivalent > 0:
                avg_savings_pct = float((total_reserved_savings / total_ondemand_equivalent) * 100)
            else:
                avg_savings_pct = 0.0

            print(f"\n{self.colorize('📊  SUMMARY OF POTENTIAL SAVINGS', Colors.TEXT_BOLD + Colors.BOLD)}")
            print(f"{self.colorize('├─ ', Colors.SECONDARY)}{self.colorize('Total On-Demand Instances:', Colors.TEXT)} "
                  f"{self.colorize(str(total_instances), Colors.TEXT_BOLD)}")

            # Calculate and format costs using Decimal
            monthly_ondemand = total_reserved_savings + (total_monthly_cost_dec - total_reserved_savings)
            monthly_ondemand_float = float(monthly_ondemand)
            print(f"{self.colorize('├─ ', Colors.SECONDARY)}{self.colorize('Monthly On-Demand Cost:', Colors.TEXT)} "
                  f"{self.colorize(f'${monthly_ondemand_float:,.2f}', Colors.TEXT_BOLD)}")

            monthly_reserved = total_monthly_cost_dec - total_reserved_savings
            monthly_reserved_float = float(monthly_reserved)
            print(f"{self.colorize('├─ ', Colors.SECONDARY)}{self.colorize('Monthly Reserved Cost:', Colors.TEXT)} "
                  f"{self.colorize(f'${monthly_reserved_float:,.2f}', Colors.TEXT_BOLD)}")

            total_reserved_savings_float = float(total_reserved_savings)
            print(f"{self.colorize('├─ ', Colors.SECONDARY)}{self.colorize('Total Monthly Savings:', Colors.TEXT)} "
                  f"💵 {self.colorize(f'${total_reserved_savings_float:,.2f}', Colors.SUCCESS + Colors.BOLD)} "
                  f"{self.colorize(f'({avg_savings_pct:.1f}%)', Colors.SUCCESS)}")

            annual_savings = total_reserved_savings * Decimal('12')
            annual_savings_float = float(annual_savings)
            print(f"{self.colorize('└─ ', Colors.SECONDARY)}{self.colorize('Annual Savings:', Colors.TEXT)} "
                  f"🏦 {self.colorize(f'${annual_savings_float:,.2f}', Colors.SUCCESS + Colors.BOLD)}")

            print(f"\n{self.colorize('💡  RECOMMENDATION', Colors.TEXT_BOLD + Colors.BOLD)}")
            print(f"Consider converting {self.colorize('On-Demand', Colors.TEXT_BOLD)} instances to "
                  f"{self.colorize('Reserved Instances', Colors.PRIMARY)} to save approximately "
                  f"{self.colorize(f'${total_reserved_savings:,.2f}', Colors.SUCCESS)} per month "
                  f"({self.colorize(f'{avg_savings_pct:.1f}%', Colors.SUCCESS)}).\n"
                  f"{self.colorize('   →', Colors.SECONDARY)} {self.colorize('Tip:', Colors.TEXT_BOLD)} Consider 3-year terms "
                  f"for additional savings ({self.colorize('up to 60% off', Colors.SUCCESS)}).")
        else:
            print(f"\n{self.colorize('Note: No On-Demand instances found for Reserved Instance analysis.', Colors.TEXT_MUTED)}")

        print(self.colorize('='*60, Colors.SECONDARY))

    def _print_footer(self) -> None:
        """Print the report footer."""
        print(f"\n{self.colorize('Report generated by FinOps CLI', Colors.TEXT_MUTED)}")
        print(f"{self.colorize('For detailed cost analysis, visit AWS Cost Explorer', Colors.TEXT_MUTED)}")
