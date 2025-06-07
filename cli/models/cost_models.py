from dataclasses import dataclass, field
from decimal import Decimal
from enum import Enum
from typing import Dict, Any, Union


class InstanceLifecycle(Enum):
    """Represents the lifecycle of an EC2 instance."""
    ON_DEMAND = 'on-demand'
    RESERVED = 'reserved'
    SPOT = 'spot'


@dataclass
class InstanceCost:
    """Represents the cost information for a single instance."""
    instance_id: str
    instance_type: str
    name: str
    lifecycle: InstanceLifecycle
    state: str
    hourly_rate: Decimal
    monthly_cost: Decimal
    annual_cost: Decimal
    region: str

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'InstanceCost':
        """Create an InstanceCost from a dictionary."""
        return cls(
            instance_id=data['InstanceId'],
            instance_type=data['InstanceType'],
            name=data.get('Name', ''),
            lifecycle=InstanceLifecycle(data['Lifecycle'].lower()),
            state=data['State'],
            hourly_rate=Decimal(str(data['HourlyCost'])),
            monthly_cost=Decimal(str(data['MonthlyCost'])),
            annual_cost=Decimal(str(data['AnnualCost'])),
            region=data.get('Region', 'unknown')
        )


@dataclass
class InstanceTypeCosts:
    """Represents cost information for a specific instance type."""
    instance_type: str
    total_instances: int = 0
    on_demand_count: int = 0
    reserved_count: int = 0
    spot_count: int = 0
    hourly_rate: Decimal = Decimal('0.0')
    monthly_cost: Decimal = Decimal('0.0')
    annual_cost: Decimal = Decimal('0.0')
    ondemand_equivalent: Decimal = Decimal('0.0')
    savings: Decimal = Decimal('0.0')

    @property
    def savings_percentage(self) -> float:
        """Calculate the percentage of savings compared to on-demand."""
        if self.ondemand_equivalent == 0:
            return 0.0
        return float((self.savings / self.ondemand_equivalent) * 100)


@dataclass
class CostSummary:
    """Summary of costs across all instances."""
    total_instances: int = 0
    total_monthly_cost: Decimal = Decimal('0.0')
    total_ondemand_cost: Decimal = Decimal('0.0')
    total_reserved_cost: Decimal = Decimal('0.0')
    total_spot_cost: Decimal = Decimal('0.0')
    monthly_savings: Decimal = Decimal('0.0')
    instance_costs: Dict[str, 'InstanceTypeCosts'] = field(default_factory=dict)

    @property
    def total_annual_cost(self) -> Decimal:
        """Calculate the total annual cost."""
        return self.total_monthly_cost * 12

    @property
    def total_annual_savings(self) -> Decimal:
        """Calculate the total annual savings."""
        return self.monthly_savings * 12


@dataclass
class SavingsOpportunity:
    """Represents a potential savings opportunity."""
    instance_type: str
    current_pricing: str
    recommended_pricing: str
    instance_count: int
    current_monthly_cost: Decimal
    potential_monthly_cost: Decimal
    monthly_savings: Decimal
    savings_percentage: float

    @property
    def annual_savings(self) -> Decimal:
        """Calculate the annual savings."""
        return self.monthly_savings * 12


# Type aliases for better type hints
InstanceCosts = Dict[str, InstanceTypeCosts]
PricingData = Dict[str, Dict[str, Union[float, int]]]
