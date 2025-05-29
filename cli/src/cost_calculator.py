from typing import Dict, List, Any
from cli.src.pricing import EC2Pricing
from cli.src.ec2_inventory import EC2Inventory
from tabulate import tabulate

class EC2CostCalculator:
    """Class to calculate EC2 instance costs."""

    def __init__(self, region: str = 'us-east-1'):
        """Initialize the cost calculator.

        Args:
            region: AWS region to analyze
        """
        self.region = region
        self.pricing = EC2Pricing(region)
        self.inventory = EC2Inventory(region)

    def _get_instance_pricing(self, instance_type: str, lifecycle: str) -> float:
        """Get the appropriate price based on instance lifecycle.

        Args:
            instance_type: Type of the EC2 instance
            lifecycle: Instance lifecycle ('on-demand', 'spot', or 'reserved')

        Returns:
            Hourly price in USD
        """
        hourly_price = self.pricing.get_ec2_ondemand_price(instance_type)

        if hourly_price is None:
            print(f"Warning: Could not get price for instance type {instance_type}")
            return 0.0

        # Apply discounts based on instance lifecycle
        if lifecycle == 'reserved':
            # Assuming 1-year no upfront reserved instance discount (~40% off on-demand)
            hourly_price *= 0.6
        elif lifecycle == 'spot':
            # Assuming spot instances cost ~70% of on-demand (this is just an estimate)
            hourly_price *= 0.7

        return hourly_price

    def calculate_instance_costs(self) -> List[Dict[str, Any]]:
        """Calculate costs for all EC2 instances in the region.

        Returns:
            List of dictionaries with cost information per instance
        """
        instances = self.inventory.get_all_instances()
        result = []

        for instance in instances:
            if instance['State'] != 'running':
                continue

            instance_type = instance['InstanceType']
            lifecycle = instance.get('InstanceLifecycle', 'on-demand')

            # Get the appropriate price based on instance lifecycle
            hourly_price = self._get_instance_pricing(instance_type, lifecycle)

            # Calculate monthly and annual costs (assuming 730 hours per month, 8760 per year)
            monthly_cost = hourly_price * 730
            annual_cost = hourly_price * 8760

            instance_info = {
                'InstanceId': instance['InstanceId'],
                'Name': instance['Tags'].get('Name', 'N/A'),
                'InstanceType': instance_type,
                'Lifecycle': lifecycle.upper(),
                'State': instance['State'],
                'HourlyCost': hourly_price,
                'MonthlyCost': monthly_cost,
                'AnnualCost': annual_cost,
                'Region': self.region
            }

            result.append(instance_info)

        return result

    def get_cost_summary(self) -> Dict[str, Any]:
        """Get a summary of EC2 costs by instance type and lifecycle.

        Returns:
            Dictionary with cost summary information including breakdown by lifecycle
        """
        instance_usage = self.inventory.get_instance_types_usage()
        summary = {
            'total_instances': 0,
            'total_monthly_cost': 0.0,
            'total_ondemand_cost': 0.0,  # Cost if all instances were on-demand
            'total_reserved_cost': 0.0,
            'total_spot_cost': 0.0,
            'monthly_savings': 0.0,
            'instance_types': {}
        }


        for instance_type, data in instance_usage.items():
            # Skip if no running instances of this type
            if data['total'] == 0:
                continue

            # Get base on-demand price
            ondemand_hourly = self.pricing.get_ec2_ondemand_price(instance_type)

            if ondemand_hourly is None:
                print(f"Warning: Could not get price for instance type {instance_type}")
                continue

            # Calculate costs by pricing model
            ondemand_monthly = ondemand_hourly * 730
            reserved_hourly = ondemand_hourly * 0.6  # 40% off
            spot_hourly = ondemand_hourly * 0.7      # 30% off

            # Calculate total costs for this instance type
            total_ondemand_cost = data.get('on-demand', 0) * ondemand_monthly
            total_reserved_cost = data.get('reserved', 0) * (reserved_hourly * 730)
            total_spot_cost = data.get('spot', 0) * (spot_hourly * 730)

            total_monthly_cost = total_ondemand_cost + total_reserved_cost + total_spot_cost

            # Calculate what the cost would be if all instances were on-demand
            total_ondemand_equivalent = data['total'] * ondemand_monthly

            # Calculate savings
            savings = total_ondemand_equivalent - total_monthly_cost

            # Update summary
            summary['instance_types'][instance_type] = {
                'total': data['total'],
                'on-demand': data.get('on-demand', 0),
                'reserved': data.get('reserved', 0),
                'spot': data.get('spot', 0),
                'hourly_price': ondemand_hourly,
                'monthly_cost': total_monthly_cost,
                'ondemand_equivalent': total_ondemand_equivalent,
                'savings': savings
            }

            summary['total_instances'] += data['total']
            summary['total_monthly_cost'] += total_monthly_cost
            summary['total_ondemand_cost'] += total_ondemand_equivalent
            summary['total_reserved_cost'] += total_reserved_cost
            summary['total_spot_cost'] += total_spot_cost
            summary['monthly_savings'] += savings

        return summary

    def _print_reserved_savings_analysis(self, summary: Dict[str, Any], use_colors: bool = True) -> None:
        """Print an analysis of potential savings from Reserved Instances.

        Args:
            summary: The cost summary dictionary from get_cost_summary()
            use_colors: Whether to use ANSI color codes in the output
        """
        # Define colors - Minimalist palette
        class Colors:
            # Primary colors
            PRIMARY = '\033[38;5;39m' if use_colors else ''  # Soft blue
            SECONDARY = '\033[38;5;247m' if use_colors else ''  # Medium gray
            SUCCESS = '\033[38;5;34m' if use_colors else ''  # Soft green
            WARNING = '\033[38;5;208m' if use_colors else ''  # Orange
            DANGER = '\033[38;5;196m' if use_colors else ''  # Red

            # Text colors
            TEXT = '\033[38;5;250m' if use_colors else ''  # Light gray
            TEXT_BOLD = '\033[1;38;5;255m' if use_colors else ''  # White bold
            TEXT_MUTED = '\033[38;5;240m' if use_colors else ''  # Dark gray

            # Utility
            RESET = '\033[0m' if use_colors else ''
            BOLD = '\033[1m' if use_colors else ''
            UNDERLINE = '\033[4m' if use_colors else ''

            # Aliases for compatibility
            HEADER = PRIMARY
            BLUE = PRIMARY
            CYAN = '\033[38;5;44m' if use_colors else ''
            GREEN = SUCCESS
            FAIL = DANGER
            ENDC = RESET

        def colorize(text: str, color: str) -> str:
            return f"{color}{text}{Colors.ENDC}" if use_colors else text

        # Header with emojis and minimalist style
        print(f"\n{colorize('='*60, Colors.SECONDARY)}")
        print(colorize(f"💰  RESERVED INSTANCE SAVINGS ANALYSIS  💰".center(60), Colors.TEXT_BOLD + Colors.BOLD))
        print(colorize('='*60, Colors.SECONDARY))
        print(f"\n{colorize('ℹ️  This analysis shows potential savings from converting On-Demand to Reserved Instances', Colors.TEXT)}")
        print(f"{colorize('   Savings are calculated using the', Colors.TEXT)} {colorize('1-year No Upfront', Colors.TEXT_BOLD)} {colorize('payment option', Colors.TEXT)} "
              f"({colorize('40% off on-demand', Colors.SUCCESS)})\n")

        total_reserved_savings = 0
        total_instances = 0
        instance_data = []

        # Process instance data
        for instance_type, data in summary['instance_types'].items():
            ondemand_count = data.get('on-demand', 0)
            if ondemand_count == 0:
                continue

            hourly_price = self._get_instance_pricing(instance_type, 'on-demand')
            reserved_hourly_price = self._get_instance_pricing(instance_type, 'reserved')

            monthly_hours = 730  # 24 hours * 365 days / 12 months
            total_ondemand_equivalent = hourly_price * ondemand_count * monthly_hours
            total_reserved_cost = reserved_hourly_price * ondemand_count * monthly_hours
            reserved_savings = total_ondemand_equivalent - total_reserved_cost
            total_reserved_savings += reserved_savings
            total_instances += ondemand_count

            savings_pct = (reserved_savings / total_ondemand_equivalent * 100) if total_ondemand_equivalent > 0 else 0
            savings_color = Colors.SUCCESS if savings_pct > 20 else Colors.WARNING
            savings_emoji = "💸" if savings_pct > 20 else "📉"

            instance_data.append([
                colorize(instance_type, Colors.TEXT_BOLD),
                ondemand_count,
                f"${hourly_price:.4f}",
                f"${reserved_hourly_price:.4f}",
                f"{savings_emoji} {colorize(f'${reserved_savings:,.2f}', savings_color)}",
                colorize(f"{savings_pct:.1f}%", savings_color)
            ])

        # Print table with instance details
        if instance_data:
            headers = [
                colorize("INSTANCE TYPE", Colors.TEXT_MUTED + Colors.UNDERLINE),
                colorize("COUNT", Colors.TEXT_MUTED + Colors.UNDERLINE),
                colorize("ON-DEMAND RATE", Colors.TEXT_MUTED + Colors.UNDERLINE),
                colorize("RESERVED RATE", Colors.TEXT_MUTED + Colors.UNDERLINE),
                colorize("MONTHLY SAVINGS", Colors.TEXT_MUTED + Colors.UNDERLINE),
                colorize("SAVINGS %", Colors.TEXT_MUTED + Colors.UNDERLINE)
            ]
            print(tabulate(instance_data, headers=headers, tablefmt="simple_grid"))

        # Print total potential savings
        if total_instances > 0:
            # Ensure total_monthly_cost is a number, not a string
            total_monthly_cost = float(summary['total_monthly_cost']) if isinstance(summary['total_monthly_cost'], (int, float)) else 0.0

            # Calculate the percentage of savings
            avg_savings_pct = (total_reserved_savings / (total_reserved_savings +
                             (total_monthly_cost - total_reserved_savings)) * 100) \
                            if (total_reserved_savings + (total_monthly_cost - total_reserved_savings)) > 0 else 0

            print(f"\n{colorize('📊  SUMMARY OF POTENTIAL SAVINGS', Colors.TEXT_BOLD + Colors.BOLD)}")
            print(f"{colorize('├─ ', Colors.SECONDARY)}{colorize('Total On-Demand Instances:', Colors.TEXT)} "
                  f"{colorize(str(total_instances), Colors.TEXT_BOLD)}")

            monthly_ondemand = total_reserved_savings + (total_monthly_cost - total_reserved_savings)
            print(f"{colorize('├─ ', Colors.SECONDARY)}{colorize('Monthly On-Demand Cost:', Colors.TEXT)} "
                  f"{colorize(f'${monthly_ondemand:,.2f}', Colors.TEXT_BOLD)}")

            monthly_reserved = total_monthly_cost - total_reserved_savings
            print(f"{colorize('├─ ', Colors.SECONDARY)}{colorize('Monthly Reserved Cost:', Colors.TEXT)} "
                  f"{colorize(f'${monthly_reserved:,.2f}', Colors.TEXT_BOLD)}")

            print(f"{colorize('├─ ', Colors.SECONDARY)}{colorize('Total Monthly Savings:', Colors.TEXT)} "
                  f"💵 {colorize(f'${total_reserved_savings:,.2f}', Colors.SUCCESS + Colors.BOLD)} "
                  f"{colorize(f'({avg_savings_pct:.1f}%)', Colors.SUCCESS)}")

            print(f"{colorize('└─ ', Colors.SECONDARY)}{colorize('Annual Savings:', Colors.TEXT)} "
                  f"🏦 {colorize(f'${total_reserved_savings * 12:,.2f}', Colors.SUCCESS + Colors.BOLD)}")

            print(f"\n{colorize('💡  RECOMMENDATION', Colors.TEXT_BOLD + Colors.BOLD)}")
            print(f"Consider converting {colorize('On-Demand', Colors.TEXT_BOLD)} instances to "
                  f"{colorize('Reserved Instances', Colors.PRIMARY)} to save approximately "
                  f"{colorize(f'${total_reserved_savings:,.2f}', Colors.SUCCESS)} per month "
                  f"({colorize(f'{avg_savings_pct:.1f}%', Colors.SUCCESS)}).\n"
                  f"{colorize('   →', Colors.SECONDARY)} {colorize('Tip:', Colors.TEXT_BOLD)} Consider 3-year terms for additional savings "
                  f"({colorize('up to 60% off', Colors.SUCCESS)}).")
        else:
            print(f"\n{colorize('ℹ️  No On-Demand instances found for Reserved Instance analysis.', Colors.WARNING)}")

        print(colorize("="*60, Colors.SECONDARY))

    def print_cost_report(self, detailed: bool = True, show_reserved_savings: bool = False, 
                        use_colors: bool = True) -> None:
        """Print a formatted cost report to the console.

        Args:
            detailed: Whether to show detailed instance information
            show_reserved_savings: Whether to show potential savings from Reserved Instances
            use_colors: Whether to use ANSI color codes in the output
        """
        # Define colors - Minimalist palette
        class Colors:
            # Primary colors
            PRIMARY = '\033[38;5;39m' if use_colors else ''  # Soft blue
            SECONDARY = '\033[38;5;247m' if use_colors else ''  # Medium gray
            SUCCESS = '\033[38;5;34m' if use_colors else ''  # Soft green
            WARNING = '\033[38;5;208m' if use_colors else ''  # Orange
            DANGER = '\033[38;5;196m' if use_colors else ''  # Red

            # Text colors
            TEXT = '\033[38;5;250m' if use_colors else ''  # Light gray
            TEXT_BOLD = '\033[1;38;5;255m' if use_colors else ''  # White bold
            TEXT_MUTED = '\033[38;5;240m' if use_colors else ''  # Dark gray

            # Utility
            RESET = '\033[0m' if use_colors else ''
            BOLD = '\033[1m' if use_colors else ''
            UNDERLINE = '\033[4m' if use_colors else ''

            # Aliases for compatibility
            HEADER = PRIMARY
            BLUE = PRIMARY
            CYAN = '\033[38;5;44m' if use_colors else ''
            GREEN = SUCCESS
            FAIL = DANGER
            ENDC = RESET

        def colorize(text: str, color: str) -> str:
            return f"{color}{text}{Colors.ENDC}" if use_colors else text

        instances = self.calculate_instance_costs()
        summary = self.get_cost_summary()

        # Print header
        print(f"\n{colorize('='*60, Colors.SECONDARY)}")
        print(colorize(f"🛠️  AWS COST ANALYSIS - {self.region.upper()}  🛠️".center(60), 
                      Colors.TEXT_BOLD + Colors.BOLD))
        print(colorize('='*60, Colors.SECONDARY))

        if show_reserved_savings:
            self._print_reserved_savings_analysis(summary, use_colors=use_colors)

        # Print instance details
        if detailed and instances:
            print(f"\n{colorize('='*60, Colors.SECONDARY)}")
            print(colorize("INSTANCE DETAILS".center(60), Colors.TEXT_BOLD + Colors.UNDERLINE))
            print(colorize('='*60, Colors.SECONDARY))

            table_data = []
            for instance in instances:
                # Determine color based on instance state
                state_color = Colors.SUCCESS if instance['State'].lower() == 'running' else \
                             Colors.WARNING if instance['State'].lower() == 'stopped' else Colors.TEXT

                # Determine color based on pricing model
                pricing_color = {
                    'On-Demand': Colors.TEXT,
                    'Reserved': Colors.SUCCESS,
                    'Spot': Colors.PRIMARY
                }.get(instance['Lifecycle'], Colors.TEXT)

                table_data.append([
                    colorize(instance['InstanceId'][:12] + '...', Colors.TEXT_BOLD),
                    instance['Name'] or '-',
                    instance['InstanceType'],
                    colorize(instance['Lifecycle'], pricing_color),
                    colorize(instance['State'], state_color),
                    colorize(f"${float(instance['HourlyCost']):.4f}", Colors.TEXT_BOLD),
                    colorize(f"${float(instance['MonthlyCost']):,.2f}", Colors.TEXT_BOLD),
                    colorize(f"${float(instance['AnnualCost']):,.2f}", Colors.TEXT_BOLD)
                ])

            headers = [
                colorize("INSTANCE ID", Colors.TEXT_MUTED + Colors.UNDERLINE),
                colorize("NAME", Colors.TEXT_MUTED + Colors.UNDERLINE),
                colorize("TYPE", Colors.TEXT_MUTED + Colors.UNDERLINE),
                colorize("PRICING", Colors.TEXT_MUTED + Colors.UNDERLINE),
                colorize("STATE", Colors.TEXT_MUTED + Colors.UNDERLINE),
                colorize("HOURLY", Colors.TEXT_MUTED + Colors.UNDERLINE),
                colorize("MONTHLY", Colors.TEXT_MUTED + Colors.UNDERLINE),
                colorize("ANNUAL", Colors.TEXT_MUTED + Colors.UNDERLINE)
            ]
            print(tabulate(table_data, headers=headers, tablefmt="simple_grid"))

        # Print summary
        if summary and summary.get('instance_types'):
            print(f"\n{colorize('='*60, Colors.SECONDARY)}")
            print(colorize("COST SUMMARY".center(60), Colors.TEXT_BOLD + Colors.UNDERLINE))
            print(colorize('='*60, Colors.SECONDARY))

            # Instance counts
            ondemand_count = sum(v.get('on-demand', 0) for v in summary['instance_types'].values())
            reserved_count = sum(v.get('reserved', 0) for v in summary['instance_types'].values())
            spot_count = sum(v.get('spot', 0) for v in summary['instance_types'].values())

            print(f"\n{colorize('INSTANCE COUNT', Colors.TEXT_BOLD + Colors.UNDERLINE)}")
            print(f"{colorize('├─ ', Colors.SECONDARY)}{colorize('Total:', Colors.TEXT)} "
                  f"{colorize(str(summary['total_instances']), Colors.TEXT_BOLD)}")
            print(f"{colorize('├─ ', Colors.SECONDARY)}{colorize('On-Demand:', Colors.TEXT)} "
                  f"{colorize(str(ondemand_count), Colors.TEXT_BOLD)}")

        # Monthly projection
        monthly_total = summary['total_monthly_cost']
        print(f"\n{colorize('MONTHLY PROJECTION', Colors.TEXT_BOLD + Colors.UNDERLINE)}")
        print(f"{colorize('├─ ', Colors.SECONDARY)}{colorize('Total:', Colors.TEXT_BOLD)} "
              f"{colorize(f'${monthly_total:,.2f}', Colors.TEXT_BOLD)}")

        if summary.get('total_ondemand_cost', 0) > 0:
            ondemand_monthly = summary['total_ondemand_cost']
            print(f"{colorize('└─ ', Colors.SECONDARY)}{colorize('On-Demand Equivalent:', Colors.TEXT)} "
                  f"{colorize(f'${ondemand_monthly:,.2f}', Colors.TEXT)}")

        # Annual projection
        print(f"\n{colorize('ANNUAL PROJECTION', Colors.TEXT_BOLD + Colors.UNDERLINE)}")
        annual_total = summary['total_monthly_cost'] * 12
        print(f"{colorize('├─ ', Colors.SECONDARY)}{colorize('Total:', Colors.TEXT_BOLD)} "
              f"{colorize(f'${annual_total:,.2f}', Colors.TEXT_BOLD)}")

        if summary.get('total_ondemand_cost', 0) > 0:
            ondemand_annual = summary['total_ondemand_cost'] * 12
            print(f"{colorize('└─ ', Colors.SECONDARY)}{colorize('On-Demand Equivalent:', Colors.TEXT)} "
                  f"{colorize(f'${ondemand_annual:,.2f}', Colors.TEXT)}")

        # Cost breakdown by instance type
        if detailed:
            print(f"\n{colorize('COST BREAKDOWN BY INSTANCE TYPE', Colors.TEXT_BOLD + Colors.UNDERLINE)}")
            print(colorize('─'*60, Colors.SECONDARY))

            table_data = []
            for instance_type, data in sorted(summary['instance_types'].items()):
                # Add a separator between instance types
                if table_data:
                    table_data.append(['─'*12, '─'*20, '─'*6, '─'*10, '─'*12, '─'*12])

                for lifecycle in ['on-demand', 'reserved', 'spot']:
                    count = data.get(lifecycle, 0)
                    if count > 0:
                        hourly_price = self._get_instance_pricing(instance_type, lifecycle)
                        monthly_cost = hourly_price * 730 * count
                        annual_cost = monthly_cost * 12

                        lifecycle_display = {
                            'on-demand': '🔄 On-Demand',
                            'reserved': '🔒 Reserved (40% off)',
                            'spot': '✨ Spot (30% off)'
                        }.get(lifecycle, lifecycle.upper())

                        # Color based on lifecycle
                        lifecycle_color = {
                            'on-demand': Colors.TEXT,
                            'reserved': Colors.SUCCESS,
                            'spot': Colors.PRIMARY
                        }.get(lifecycle, Colors.TEXT)

                        table_data.append([
                            colorize(instance_type, Colors.TEXT_BOLD) if lifecycle == 'on-demand' else '',
                            colorize(lifecycle_display, lifecycle_color),
                            count,
                            colorize(f"${hourly_price:.4f}", Colors.TEXT_BOLD),
                            colorize(f"${monthly_cost:,.2f}", Colors.TEXT_BOLD),
                            colorize(f"${annual_cost:,.2f}", Colors.TEXT_BOLD)
                        ])

                # Add savings information if applicable
                if data.get('savings', 0) > 0 and data.get('ondemand_equivalent', 0) > 0:
                    savings_pct = (data['savings'] / data['ondemand_equivalent'] * 100) \
                                if data['ondemand_equivalent'] > 0 else 0
                    savings_emoji = "💰" if savings_pct > 15 else "📉"

                    table_data.append([
                        '',
                        colorize(f"{savings_emoji} Savings ({savings_pct:.1f}%)", Colors.SUCCESS),
                        '',
                        '',
                        colorize(f"-${data['savings']:,.2f}", Colors.SUCCESS + Colors.BOLD),
                        colorize(f"-${data['savings'] * 12:,.2f}", Colors.SUCCESS + Colors.BOLD)
                    ])

            headers = [
                colorize("INSTANCE TYPE", Colors.TEXT_MUTED + Colors.UNDERLINE),
                colorize("PRICING MODEL", Colors.TEXT_MUTED + Colors.UNDERLINE),
                colorize("COUNT", Colors.TEXT_MUTED + Colors.UNDERLINE),
                colorize("RATE/HOUR", Colors.TEXT_MUTED + Colors.UNDERLINE),
                colorize("MONTHLY", Colors.TEXT_MUTED + Colors.UNDERLINE),
                colorize("ANNUAL", Colors.TEXT_MUTED + Colors.UNDERLINE)
            ]
            print(tabulate(table_data, headers=headers, tablefmt="simple_grid"))

        # Print footer with notes
        print(f"\n{colorize('NOTES', Colors.TEXT_BOLD + Colors.UNDERLINE)}")
        print(colorize('─'*60, Colors.SECONDARY))
        print(f"{colorize('•', Colors.SECONDARY)} Monthly costs are estimates based on 730 hours per month.")
        print(f"{colorize('•', Colors.SECONDARY)} Annual costs are based on 8,760 hours per year.")
        print(f"{colorize('•', Colors.SECONDARY)} Reserved instances: 1-year no upfront (40% off on-demand).")
        print(f"{colorize('•', Colors.SECONDARY)} Spot instances: ~30% savings over on-demand pricing.")

        if show_reserved_savings:
            print(f"{colorize('•', Colors.SECONDARY)} {colorize('Tip:', Colors.TEXT_BOLD)} Consider 3-year terms "
                  f"for additional savings ({colorize('up to 60% off', Colors.SUCCESS)}).")

        print(colorize('='*60, Colors.SECONDARY))
