import json
import logging
from typing import Dict, Optional, Any
import boto3
from botocore.exceptions import BotoCoreError, ClientError
from cli.services.exceptions import AWSServiceError
from cli.models.pricing import (
    EC2PriceInfo,
    EC2PricingRequest,
    EC2PriceDimensions,
    EC2PriceTerm
)

logger = logging.getLogger(__name__)

class PricingService:
    """Service for retrieving AWS pricing information."""

    def __init__(self, region: str = 'us-east-1', profile_name: Optional[str] = None):
        """Initialize the PricingService.

        Args:
            region: AWS region (default: us-east-1, as Pricing API is only available there)
            profile_name: Optional AWS profile name for authentication
        """
        self.region = region
        session = boto3.Session(profile_name=profile_name) if profile_name else boto3.Session()
        self.client = session.client('pricing', region_name='us-east-1')
        logger.debug("Initialized PricingService for region %s", region)

    def get_ec2_ondemand_price(
        self,
        instance_type: str,
        operating_system: str = 'Linux',
        tenancy: str = 'Shared',
        pre_installed_sw: str = 'NA',
        capacity_status: str = 'Used',
        region: Optional[str] = None
    ) -> Optional[float]:
        """Get the current on-demand price for an EC2 instance type.

        Args:
            instance_type: EC2 instance type (e.g., 't3.micro')
            operating_system: OS type (Linux/Windows)
            tenancy: Shared/Dedicated/Host
            pre_installed_sw: NA/NA
            capacity_status: Used/AllocatedCapacityReservation
            region: AWS region (defaults to the one set in the constructor)

        Returns:
            float: The hourly price in USD, or None if not found

        Raises:
            AWSServiceError: If there's an error calling the AWS API
        """
        region = region or self.region

        try:
            # Create pricing request with proper enum values
            from cli.models.pricing import OperatingSystem, Tenancy, CapacityStatus

            # Convert string values to enums if they're not already
            os_enum = OperatingSystem(operating_system) if isinstance(operating_system, str) else operating_system
            tenancy_enum = Tenancy(tenancy) if isinstance(tenancy, str) else tenancy
            capacity_enum = CapacityStatus(capacity_status) if isinstance(capacity_status, str) else capacity_status

            pricing_request = EC2PricingRequest(
                instance_type=instance_type,
                operating_system=os_enum,
                tenancy=tenancy_enum,
                pre_installed_sw=pre_installed_sw,
                capacity_status=capacity_enum,
                region=region or self.region
            )

            # Get price info from AWS
            response = self.client.get_products(
                ServiceCode='AmazonEC2',
                Filters=pricing_request.to_filters(),
                MaxResults=1
            )

            if not response.get('PriceList'):
                logger.warning(
                    "No pricing information found for instance type %s in region %s",
                    instance_type, region
                )
                return None

            # Parse the price information
            price_item = json.loads(response['PriceList'][0])
            price_info = self._parse_price_info(price_item)

            # Extract the price from the first available term
            for term_type in ['OnDemand', 'Reserved']:
                for term in price_info.terms.get(term_type, {}).values():
                    for dimension in term.price_dimensions.values():
                        if 'USD' in dimension.price_per_unit:
                            return float(dimension.price_per_unit['USD'])

            logger.warning("No USD price found for instance type %s", instance_type)
            return None

        except (BotoCoreError, ClientError) as e:
            error_msg = f"Error getting price for {instance_type} in {region}: {str(e)}"
            logger.error(error_msg)
            raise AWSServiceError(error_msg) from e
        except json.JSONDecodeError as e:
            error_msg = f"Error parsing pricing response for {instance_type}: {str(e)}"
            logger.error(error_msg)
            raise AWSServiceError(error_msg) from e
        except Exception as e:
            error_msg = f"Unexpected error getting price for {instance_type}: {str(e)}"
            logger.exception(error_msg)
            raise AWSServiceError(error_msg) from e

    def _parse_price_info(self, price_item: Dict[str, Any]) -> EC2PriceInfo:
        """Parse the raw price item into structured data.

        Args:
            price_item: Raw price item from AWS Pricing API

        Returns:
            EC2PriceInfo: Parsed price information
        """
        terms = {}

        # Parse terms (OnDemand, Reserved, etc.)
        for term_type, term_data in price_item.get('terms', {}).items():
            terms[term_type] = {}

            for term_code, term_info in term_data.items():
                price_dimensions = {}

                # Parse price dimensions
                for dim_code, dim_info in term_info.get('priceDimensions', {}).items():
                    price_dimensions[dim_code] = EC2PriceDimensions(
                        rate_code=dim_info.get('rateCode', ''),
                        description=dim_info.get('description', ''),
                        begin_range=dim_info.get('beginRange', '0'),
                        end_range=dim_info.get('endRange', 'Inf'),
                        unit=dim_info.get('unit', ''),
                        price_per_unit=dim_info.get('pricePerUnit', {}),
                        applicable_rates=dim_info.get('appliesTo')
                    )

                # Create the term
                terms[term_type][term_code] = EC2PriceTerm(
                    effective_date=term_info.get('effectiveDate', ''),
                    offer_term_code=term_code,
                    term_attributes=term_info.get('termAttributes', {}),
                    price_dimensions=price_dimensions
                )

        # Create the price info object
        return EC2PriceInfo(
            product=price_item.get('product', {}),
            service_code=price_item.get('serviceCode', ''),
            terms=terms,
            version=price_item.get('version', ''),
            publication_date=price_item.get('publicationDate', '')
        )
