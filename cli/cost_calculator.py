from typing import Dict, List, Any, Optional
import logging
from decimal import Decimal

from cli.services.aws.pricing import PricingService
from cli.services.aws.ec2 import EC2Service
from cli.ui import EC2CostReporter, Colors
from cli.utils import CSVExporter
from cli.models import (
    InstanceLifecycle,
    InstanceTypeCosts,
    CostSummary
)

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
        self.pricing = PricingService(region=region, profile_name=profile_name)
        self.inventory = EC2Service(region=region, profile_name=profile_name)
        logger.debug("Initialized EC2CostCalculator for region %s", region)

    def _get_instance_pricing(self, instance_type: str, lifecycle: InstanceLifecycle) -> float:
        """Get the appropriate price based on instance lifecycle.

        Args:
            instance_type: Type of the EC2 instance
            lifecycle: Instance lifecycle (from InstanceLifecycle enum)

        Returns:
            Hourly price in USD
        """
        try:
            hourly_price = self.pricing.get_ec2_ondemand_price(
                instance_type=instance_type,
                operating_system='Linux',
                tenancy='Shared',
                capacity_status='Used',
                region=self.region
            )

            if hourly_price is None:
                logger.warning("Could not get price for instance type %s", instance_type)
                return 0.0

            if lifecycle == InstanceLifecycle.RESERVED:
                hourly_price *= 0.6
            elif lifecycle == InstanceLifecycle.SPOT:
                hourly_price *= 0.7

            lifecycle_str = str(lifecycle.value if hasattr(lifecycle, 'value') else lifecycle)
            logger.debug("Price for %s (%s): $%.4f/hour", instance_type, lifecycle_str, hourly_price)
            return hourly_price

        except Exception as e:
            logger.error("Error getting price for instance type %s: %s", instance_type, str(e))
            return 0.0

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

            # Get the appropriate price based on instance lifecycle
            hourly_price = self._get_instance_pricing(instance_type, instance.lifecycle)

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

    def _print_reserved_savings_analysis(self, summary: CostSummary, use_colors: bool = True) -> None:
        """Print an analysis of potential savings from Reserved Instances.
        
        Note: This method is kept for backward compatibility. Use EC2CostReporter instead.
        
        Args:
            summary: The cost summary object from get_cost_summary()
            use_colors: Whether to use ANSI color codes in the output
        """
        reporter = EC2CostReporter(region=self.region, use_colors=use_colors)
        reporter.print_reserved_savings_analysis(summary)

    def get_instances_data(self) -> List[Dict[str, Any]]:
        """Get instance data in a format suitable for CSV export.

        Returns:
            List of dictionaries containing instance data with the following fields:
            - instance_id: ID of the instance
            - name: Name tag of the instance
            - instance_type: Type of the instance
            - pricing_model: Pricing model (ON-DEMAND, SPOT, etc.)
            - state: Current state of the instance
            - hourly_rate: Hourly cost rate
            - monthly_cost: Estimated monthly cost
            - annual_cost: Estimated annual cost
        """
        instances = self.inventory.get_all_instances()
        result = []

        for instance in instances:
            if instance.state.lower() != 'running':
                continue

            instance_type = instance.instance_type
            lifecycle = instance.lifecycle
            
            hourly_price = self._get_instance_pricing(instance_type, lifecycle)
            monthly_cost = float(hourly_price) * 730.0  # 730 hours in a month
            annual_cost = float(hourly_price) * 8760.0  # 8760 hours in a year

            instance_info = {
                'instance_id': instance.instance_id,
                'name': instance.tags.get('Name', 'N/A'),
                'instance_type': instance_type,
                'pricing_model': str(lifecycle.value).upper(),
                'state': instance.state,
                'hourly_rate': f"${float(hourly_price):.4f}",
                'monthly_cost': f"${monthly_cost:,.2f}",
                'annual_cost': f"${annual_cost:,.2f}",
                'region': instance.region
            }
            result.append(instance_info)

        return result

    def get_costs_data(self) -> List[Dict[str, Any]]:
        """Get cost summary data in a format suitable for CSV export.

        Returns:
            List of dictionaries containing cost data with the following fields:
            - instance_type: Type of the instance
            - pricing_model: Pricing model (ON-DEMAND, RESERVED, SPOT)
            - count: Number of instances
            - hourly_rate: Cost per hour
            - monthly_cost: Estimated monthly cost
            - annual_cost: Estimated annual cost
        """
        summary = self.get_cost_summary()
        result = []

        for instance_type, instance_cost in summary.instance_costs.items():
            # Only include non-zero counts
            if instance_cost.on_demand_count > 0:
                result.append({
                    'instance_type': instance_type,
                    'pricing_model': 'ON-DEMAND',
                    'count': instance_cost.on_demand_count,
                    'hourly_rate': f"${float(instance_cost.hourly_rate):.4f}",
                    'monthly_cost': f"${float(instance_cost.monthly_cost):.2f}",
                    'annual_cost': f"${float(instance_cost.annual_cost):.2f}"
                })
                
            if instance_cost.reserved_count > 0:
                reserved_hourly = instance_cost.hourly_rate * Decimal('0.6')  # 40% off for reserved
                reserved_monthly = instance_cost.monthly_cost * Decimal('0.6')
                reserved_annual = instance_cost.annual_cost * Decimal('0.6')
                
                result.append({
                    'instance_type': instance_type,
                    'pricing_model': 'RESERVED',
                    'count': instance_cost.reserved_count,
                    'hourly_rate': f"${float(reserved_hourly):.4f}",
                    'monthly_cost': f"${float(reserved_monthly):.2f}",
                    'annual_cost': f"${float(reserved_annual):.2f}"
                })
                
            if instance_cost.spot_count > 0:
                spot_hourly = instance_cost.hourly_rate * Decimal('0.7')  # 30% off for spot
                spot_monthly = instance_cost.monthly_cost * Decimal('0.7')
                spot_annual = instance_cost.annual_cost * Decimal('0.7')
                
                result.append({
                    'instance_type': instance_type,
                    'pricing_model': 'SPOT',
                    'count': instance_cost.spot_count,
                    'hourly_rate': f"${float(spot_hourly):.4f}",
                    'monthly_cost': f"${float(spot_monthly):.2f}",
                    'annual_cost': f"${float(spot_annual):.2f}"
                })

        return result

    def get_savings_data(self) -> List[Dict[str, Any]]:
        """Get potential savings analysis data in a format suitable for CSV export.

        Returns:
            List of dictionaries containing savings data with the following fields:
            - instance_type: Type of the instance
            - current_pricing: Current pricing model (ON-DEMAND, RESERVED, SPOT)
            - recommended_pricing: Recommended pricing model for savings
            - instance_count: Number of instances
            - current_monthly_cost: Current monthly cost
            - potential_monthly_cost: Potential monthly cost after optimization
            - monthly_savings: Potential monthly savings
            - annual_savings: Potential annual savings
            - savings_percentage: Percentage of savings
        """
        summary = self.get_cost_summary()
        result = []

        for instance_type, instance_cost in summary.instance_costs.items():
            # Check for potential savings from On-Demand to Reserved
            if instance_cost.on_demand_count > 0 and instance_cost.on_demand_count > instance_cost.reserved_count:
                current_cost = float(instance_cost.monthly_cost)
                potential_cost = float(instance_cost.monthly_cost * Decimal('0.6'))  # 40% off for reserved
                savings = current_cost - potential_cost
                
                if savings > 0:
                    result.append({
                        'instance_type': instance_type,
                        'current_pricing': 'ON-DEMAND',
                        'recommended_pricing': 'RESERVED',
                        'instance_count': instance_cost.on_demand_count,
                        'current_monthly_cost': f"${current_cost:,.2f}",
                        'potential_monthly_cost': f"${potential_cost:,.2f}",
                        'monthly_savings': f"${savings:,.2f}",
                        'annual_savings': f"${savings * 12:,.2f}",
                        'savings_percentage': '40%'
                    })
            
            # Check for potential savings from On-Demand to Spot (if applicable)
            if instance_cost.on_demand_count > 0 and instance_cost.spot_count > 0:
                current_cost = float(instance_cost.monthly_cost)
                potential_cost = float(instance_cost.monthly_cost * Decimal('0.7'))  # 30% off for spot
                savings = current_cost - potential_cost
                
                if savings > 0:
                    result.append({
                        'instance_type': instance_type,
                        'current_pricing': 'ON-DEMAND',
                        'recommended_pricing': 'SPOT',
                        'instance_count': instance_cost.on_demand_count,
                        'current_monthly_cost': f"${current_cost:,.2f}",
                        'potential_monthly_cost': f"${potential_cost:,.2f}",
                        'monthly_savings': f"${savings:,.2f}",
                        'annual_savings': f"${savings * 12:,.2f}",
                        'savings_percentage': '30%'
                    })

        return result

    def _add_instance_cost_row(self, table_data: List[List[Any]], instance_type: str, 
                             instance_cost: InstanceTypeCosts, lifecycle_str: str, 
                             count: int, colorize: callable) -> None:
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
        hourly_price = Decimal(str(self._get_instance_pricing(instance_type, lifecycle_enum)))
        
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
    
    def export_to_csv(self, data: List[Dict[str, Any]], output_file: str) -> str:
        """Export data to a CSV file.

        Args:
            data: List of dictionaries containing data to export
            output_file: Path to the output CSV file

        Returns:
            str: Path to the generated CSV file
        """
        return CSVExporter.export_to_csv(data, output_file)
    
    def export_instances_to_csv(self, output_file: Optional[str] = None) -> str:
        """Export instance data to CSV.

        Args:
            output_file: Path to the output CSV file. If not provided, a default name will be used.

        Returns:
            str: Path to the generated CSV file
        """
        instances = self.get_instances_data()
        return CSVExporter.export_instances_to_csv(instances, output_file)
    
    def export_costs_to_csv(self, output_file: Optional[str] = None) -> str:
        """Export cost data to CSV.

        Args:
            output_file: Path to the output CSV file. If not provided, a default name will be used.

        Returns:
            str: Path to the generated CSV file
        """
        costs = self.get_costs_data()
        return CSVExporter.export_costs_to_csv(costs, output_file)
    
    def export_savings_to_csv(self, output_file: Optional[str] = None) -> str:
        """Export savings analysis data to CSV.

        Args:
            output_file: Path to the output CSV file. If not provided, a default name will be used.

        Returns:
            str: Path to the generated CSV file
        """
        savings = self.get_savings_data()
        return CSVExporter.export_savings_to_csv(savings, output_file)

    def print_cost_report(self, detailed: bool = True, show_reserved_savings: bool = False, 
                         use_colors: bool = True) -> None:
        """Print a formatted cost report to the console.

        Args:
            detailed: Whether to show detailed instance information
            show_reserved_savings: Whether to show potential savings from Reserved Instances
            use_colors: Whether to use ANSI color codes in the output
        """
        # Get the data
        instances = self.calculate_instance_costs()
        summary = self.get_cost_summary()
        
        # Create and use the reporter
        reporter = EC2CostReporter(region=self.region, use_colors=use_colors)
        
        # Print the report sections
        reporter._print_header()
        
        if summary and summary.instance_costs:
            if detailed:
                reporter._print_instance_details(summary.instance_costs)
            
            reporter._print_cost_summary(summary)
            
            if detailed:
                reporter._print_cost_breakdown(summary.instance_costs)
        
        if show_reserved_savings:
            reporter.print_reserved_savings_analysis(summary)
                
        reporter._print_footer()
