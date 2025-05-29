import boto3
import json
from typing import Optional

class EC2Pricing:
    """Class to handle EC2 pricing information."""

    def __init__(self, region: str = 'us-east-1'):
        """Initialize the EC2Pricing client.

        Args:
            region: AWS region to get pricing for
        """
        self.client = boto3.client('pricing', region_name='us-east-1')  # Pricing API is only available in us-east-1
        self.region = region

    def get_ec2_ondemand_price(
        self,
        instance_type: str,
        operating_system: str = 'Linux',
        tenancy: str = 'Shared',
        pre_installed_sw: str = 'NA',
        capacity_status: str = 'Used'
    ) -> Optional[float]:
        """Get the current on-demand price for an EC2 instance type.

        Args:
            instance_type: EC2 instance type (e.g., 't3.micro')
            operating_system: OS type (Linux/Windows)
            tenancy: Shared/Dedicated/Host
            pre_installed_sw: NA/NA
            capacity_status: Used/AllocatedCapacityReservation

        Returns:
            float: The hourly price in USD, or None if not found
        """
        filters = [
            {'Type': 'TERM_MATCH', 'Field': 'serviceCode', 'Value': 'AmazonEC2'},
            {'Type': 'TERM_MATCH', 'Field': 'instanceType', 'Value': instance_type},
            {'Type': 'TERM_MATCH', 'Field': 'operatingSystem', 'Value': operating_system},
            {'Type': 'TERM_MATCH', 'Field': 'tenancy', 'Value': tenancy},
            {'Type': 'TERM_MATCH', 'Field': 'preInstalledSw', 'Value': pre_installed_sw},
            {'Type': 'TERM_MATCH', 'Field': 'capacitystatus', 'Value': capacity_status},
            {'Type': 'TERM_MATCH', 'Field': 'location', 'Value': self._get_region_name()}
        ]

        try:
            response = self.client.get_products(
                ServiceCode='AmazonEC2',
                Filters=filters,
                MaxResults=1
            )

            if 'PriceList' in response and response['PriceList']:
                price_item = json.loads(response['PriceList'][0])

                # Navigate through the complex JSON structure to find the price
                terms = price_item.get('terms', {})
                on_demand = terms.get('OnDemand', {})

                for _, term in on_demand.items():
                    price_dimensions = term.get('priceDimensions', {})
                    for _, dimension in price_dimensions.items():
                        price_per_unit = dimension.get('pricePerUnit', {})
                        return float(price_per_unit.get('USD', 0))

        except Exception as e:
            print(f"Error getting price for {instance_type}: {str(e)}")

        return None

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
