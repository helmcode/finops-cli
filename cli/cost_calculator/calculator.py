import logging
from typing import Dict, List, Any, Optional
from decimal import Decimal

from cli.services import PricingService, EC2Service, PriceCalculator
from cli.models import InstanceTypeCosts, CostSummary

logger = logging.getLogger(__name__)

class EC2CostCalculator:
    """Class to calculate EC2 instance costs."""

    def __init__(self, region: str = 'us-east-1', profile_name: Optional[str] = None):
        """Initialize the cost calculator.

        Args:
            region: AWS region to analyze
            profile_name: Optional AWS profile name for authentication
        """
        self.region = region
        pricing_service = PricingService(region=region, profile_name=profile_name)
        self.pricing = PriceCalculator(pricing_service, region=region)
        self.inventory = EC2Service(region=region, profile_name=profile_name)
        logger.debug("Initialized EC2CostCalculator for region %s", region)

    def calculate_instance_costs(self) -> List[Dict[str, Any]]:
        """Calculate costs for all EC2 instances in the region.

        Returns:
            List of dictionaries with cost information per instance
        """
        instances = self.inventory.get_all_instances()
        result = []

        for instance in instances:
            if instance.state.lower() != 'running':
                continue

            instance_type = instance.instance_type
            hourly_price = self.pricing.get_instance_price(instance_type, instance.lifecycle)

            # Calculate monthly and annual costs (assuming 730 hours per month, 8760 per year)
            monthly_cost = hourly_price * 730
            annual_cost = hourly_price * 8760

            instance_info = {
                'InstanceId': instance.instance_id,
                'Name': instance.tags.get('Name', 'N/A'),
                'InstanceType': instance_type,
                'Lifecycle': instance.lifecycle.value.upper(),
                'State': instance.state,
                'HourlyCost': hourly_price,
                'MonthlyCost': monthly_cost,
                'AnnualCost': annual_cost,
                'Region': self.region
            }

            result.append(instance_info)
            logger.debug("Calculated costs for instance %s: %s", instance.instance_id, instance_info)

        return result

    def get_cost_summary(self) -> CostSummary:
        """Get a summary of EC2 costs by instance type and lifecycle.

        Returns:
            CostSummary: Object containing cost summary information including breakdown by lifecycle
        """
        instance_usage = self.inventory.get_instance_types_usage()
        logger.debug("Instance usage summary: %s", instance_usage)

        summary = CostSummary()
        summary.instance_costs = {}

        for instance_type, data in instance_usage.items():
            # Skip if no running instances of this type
            if data['total'] == 0:
                continue

            # Get base on-demand price
            ondemand_hourly = self.pricing.get_ec2_ondemand_price(instance_type)

            if ondemand_hourly is None:
                logger.warning("Could not get price for instance type %s", instance_type)
                continue

            # Convert to Decimal for precise calculations
            ondemand_hourly = Decimal(str(ondemand_hourly))

            # Calculate costs by pricing model
            ondemand_monthly = ondemand_hourly * Decimal('730')  # Hours in a month
            reserved_hourly = ondemand_hourly * Decimal('0.6')  # 40% off
            spot_hourly = ondemand_hourly * Decimal('0.7')      # 30% off

            # Get instance counts
            ondemand_count = data.get('on-demand', 0)
            reserved_count = data.get('reserved', 0)
            spot_count = data.get('spot', 0)

            # Calculate total costs for this instance type
            total_ondemand_cost = Decimal(str(ondemand_count)) * ondemand_monthly
            total_reserved_cost = Decimal(str(reserved_count)) * (reserved_hourly * Decimal('730'))
            total_spot_cost = Decimal(str(spot_count)) * (spot_hourly * Decimal('730'))

            total_monthly_cost = total_ondemand_cost + total_reserved_cost + total_spot_cost

            # Calculate what the cost would be if all instances were on-demand
            total_ondemand_equivalent = Decimal(str(data['total'])) * ondemand_monthly

            # Calculate savings
            savings = total_ondemand_equivalent - total_monthly_cost

            # Create instance type costs
            instance_costs = InstanceTypeCosts(
                instance_type=instance_type,
                total_instances=data['total'],
                on_demand_count=ondemand_count,
                reserved_count=reserved_count,
                spot_count=spot_count,
                hourly_rate=ondemand_hourly,
                monthly_cost=total_monthly_cost,
                annual_cost=total_monthly_cost * Decimal('12'),
                ondemand_equivalent=total_ondemand_equivalent,
                savings=savings
            )

            # Update summary
            summary.instance_costs[instance_type] = instance_costs
            summary.total_instances += data['total']
            summary.total_monthly_cost += total_monthly_cost
            summary.total_ondemand_cost += total_ondemand_equivalent
            summary.total_reserved_cost += total_reserved_cost
            summary.total_spot_cost += total_spot_cost
            summary.monthly_savings += savings

        return summary

    def print_cost_report(self, detailed: bool = True, show_reserved_savings: bool = False, 
                           use_colors: bool = True) -> None:
        """Print a formatted cost report to the console.

        Args:
            detailed: Whether to show detailed instance information
            show_reserved_savings: Whether to show potential savings from Reserved Instances
            use_colors: Whether to use ANSI color codes in the output
        """
        from .reporter import EC2CostReporterExtended

        # Get the cost summary
        summary = self.get_cost_summary()

        # Create and use the reporter
        reporter = EC2CostReporterExtended(region=self.region, use_colors=use_colors)
        reporter.print_cost_report(
            summary=summary,
            detailed=detailed,
            show_reserved_savings=show_reserved_savings,
            use_colors=use_colors
        )

    def export_costs_to_csv(self, output_file: str = None) -> str:
        """Export cost data to a CSV file.

        Args:
            output_file: Path to the output CSV file. If not provided, a default name will be used.

        Returns:
            str: Path to the generated CSV file.
        """
        from .exporter import CostDataExporter

        # Create an instance of the exporter
        exporter = CostDataExporter(self)

        # Get the cost data
        cost_data = exporter.get_costs_data()

        # Generate a default filename if none provided
        if not output_file:
            from datetime import datetime
            timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
            output_file = f"ec2_costs_{self.region}_{timestamp}.csv"

        # Export to CSV
        from cli.utils import CSVExporter
        CSVExporter.export_to_csv(cost_data, output_file)

        return output_file

    def export_instances_to_csv(self, output_file: str = None) -> str:
        """Export instance data to a CSV file.

        Args:
            output_file: Path to the output CSV file. If not provided, a default name will be used.

        Returns:
            str: Path to the generated CSV file.
        """
        from .exporter import CostDataExporter

        # Create an instance of the exporter
        exporter = CostDataExporter(self)

        # Get the instance data
        instance_data = exporter.get_instances_data()

        # Generate a default filename if none provided
        if not output_file:
            from datetime import datetime
            timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
            output_file = f"ec2_instances_{self.region}_{timestamp}.csv"

        # Export to CSV
        from cli.utils import CSVExporter
        CSVExporter.export_to_csv(instance_data, output_file)

        return output_file

    def export_savings_to_csv(self, output_file: str = None) -> str:
        """Export savings analysis to a CSV file.

        This method exports a detailed analysis of potential savings from
        converting On-Demand instances to Reserved Instances.

        Args:
            output_file: Path to the output CSV file. If not provided, a default name will be used.

        Returns:
            str: Path to the generated CSV file.
        """
        from .exporter import CostDataExporter
        from decimal import Decimal

        # Create an instance of the exporter
        exporter = CostDataExporter(self)

        # Get the cost summary for savings analysis
        summary = self.get_cost_summary()

        # Get the savings data
        savings_data = []
        for instance_type, instance_cost in summary.instance_costs.items():
            if instance_cost.on_demand_count == 0:
                continue

            hourly_price = instance_cost.hourly_rate
            reserved_hourly = hourly_price * Decimal('0.6')  # 40% off for reserved

            monthly_ondemand = hourly_price * Decimal('730') * instance_cost.on_demand_count
            monthly_reserved = reserved_hourly * Decimal('730') * instance_cost.on_demand_count
            monthly_savings = monthly_ondemand - monthly_reserved

            savings_percent = (monthly_savings / monthly_ondemand * 100) if monthly_ondemand > 0 else 0

            savings_data.append({
                'instance_type': instance_type,
                'on_demand_count': instance_cost.on_demand_count,
                'on_demand_rate': float(hourly_price),
                'reserved_rate': float(reserved_hourly),
                'monthly_ondemand_cost': float(monthly_ondemand),
                'monthly_reserved_cost': float(monthly_reserved),
                'monthly_savings': float(monthly_savings),
                'savings_percent': float(savings_percent),
                'annual_savings': float(monthly_savings * 12)
            })

        # Generate a default filename if none provided
        if not output_file:
            from datetime import datetime
            timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
            output_file = f"ec2_savings_{self.region}_{timestamp}.csv"

        # Export to CSV
        from cli.utils import CSVExporter
        CSVExporter.export_to_csv(savings_data, output_file)

        return output_file
