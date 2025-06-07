from typing import List, Dict, Any, Optional
from decimal import Decimal

from cli.utils import CSVExporter
from cli.models import InstanceTypeCosts, CostSummary

class CostDataExporter:
    """Class to handle exporting cost data to various formats."""

    def __init__(self, calculator):
        """Initialize the exporter with a calculator instance.

        Args:
            calculator: Instance of EC2CostCalculator
        """
        self.calculator = calculator

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
        instances = self.calculator.inventory.get_all_instances()
        result = []

        for instance in instances:
            if instance.state.lower() != 'running':
                continue

            instance_type = instance.instance_type
            lifecycle = instance.lifecycle

            hourly_price = self.calculator.pricing.get_instance_price(instance_type, lifecycle)
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
                'annual_cost': f"${annual_cost:,.2f}"
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
        summary = self.calculator.get_cost_summary()
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
        summary = self.calculator.get_cost_summary()
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
