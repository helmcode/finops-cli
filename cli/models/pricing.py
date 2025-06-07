from dataclasses import dataclass, field
from enum import Enum
from typing import Dict, Optional


class OperatingSystem(str, Enum):
    """Supported operating systems for EC2 instances."""
    LINUX = 'Linux'
    WINDOWS = 'Windows'
    SUSE = 'SUSE'
    RHEL = 'RHEL'


class Tenancy(str, Enum):
    """Supported tenancy options for EC2 instances."""
    SHARED = 'Shared'
    DEDICATED = 'Dedicated'
    HOST = 'Host'


class CapacityStatus(str, Enum):
    """Capacity status for EC2 instances."""
    USED = 'Used'
    ALLOCATED = 'AllocatedCapacityReservation'


@dataclass
class EC2PriceDimensions:
    """Represents the price dimensions for an EC2 instance."""
    rate_code: str
    description: str
    begin_range: str
    end_range: str
    unit: str
    price_per_unit: Dict[str, str]  # Currency -> Price
    applicable_rates: Optional[Dict] = None


@dataclass
class EC2PriceTerm:
    """Represents a pricing term for an EC2 instance."""
    effective_date: str
    offer_term_code: str
    term_attributes: Dict
    price_dimensions: Dict[str, EC2PriceDimensions]


@dataclass
class EC2PriceInfo:
    """Represents pricing information for an EC2 instance."""
    product: Dict
    service_code: str
    terms: Dict[str, Dict[str, EC2PriceTerm]]  # OnDemand/Reserved -> TermCode -> EC2PriceTerm
    version: str
    publication_date: str


@dataclass
class EC2PricingRequest:
    """Represents a request for EC2 pricing information."""
    instance_type: str
    operating_system: OperatingSystem = OperatingSystem.LINUX
    tenancy: Tenancy = Tenancy.SHARED
    pre_installed_sw: str = 'NA'
    capacity_status: CapacityStatus = CapacityStatus.USED
    region: str = 'us-east-1'

    def to_filters(self) -> list[dict]:
        """Convert the request to AWS Pricing API filters."""
        # Obtener los valores, manejando tanto strings como objetos Enum
        os_value = self.operating_system.value if hasattr(self.operating_system, 'value') else str(self.operating_system)
        tenancy_value = self.tenancy.value if hasattr(self.tenancy, 'value') else str(self.tenancy)
        capacity_value = self.capacity_status.value if hasattr(self.capacity_status, 'value') else str(self.capacity_status)

        return [
            {'Type': 'TERM_MATCH', 'Field': 'serviceCode', 'Value': 'AmazonEC2'},
            {'Type': 'TERM_MATCH', 'Field': 'instanceType', 'Value': self.instance_type},
            {'Type': 'TERM_MATCH', 'Field': 'operatingSystem', 'Value': os_value},
            {'Type': 'TERM_MATCH', 'Field': 'tenancy', 'Value': tenancy_value},
            {'Type': 'TERM_MATCH', 'Field': 'preInstalledSw', 'Value': self.pre_installed_sw},
            {'Type': 'TERM_MATCH', 'Field': 'capacitystatus', 'Value': capacity_value},
            {'Type': 'TERM_MATCH', 'Field': 'location', 'Value': self._get_region_name()}
        ]

    def _get_region_name(self) -> str:
        """Convert AWS region code to full region name for Pricing API."""
        region_map = {
            'us-east-1': 'US East (N. Virginia)',
            'us-east-2': 'US East (Ohio)',
            'us-west-1': 'US West (N. California)',
            'us-west-2': 'US West (Oregon)',
            'eu-west-1': 'EU (Ireland)',
            'eu-west-2': 'EU (London)',
            'eu-central-1': 'EU (Frankfurt)',
            'ap-south-1': 'Asia Pacific (Mumbai)',
            'ap-northeast-1': 'Asia Pacific (Tokyo)',
            'ap-northeast-2': 'Asia Pacific (Seoul)',
            'ap-southeast-1': 'Asia Pacific (Singapore)',
            'ap-southeast-2': 'Asia Pacific (Sydney)',
            'sa-east-1': 'South America (Sao Paulo)',
            'ca-central-1': 'Canada (Central)'
        }
        return region_map.get(self.region, self.region)
