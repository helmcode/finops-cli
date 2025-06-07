import logging
from typing import Optional
from cli.services.aws.pricing import PricingService
from cli.models import InstanceLifecycle

logger = logging.getLogger(__name__)


class PriceCalculator:
    """Service for calculating EC2 instance prices with different lifecycles."""

    def __init__(self, pricing_service: PricingService, region: str):
        """Initialize the price calculator.

        Args:
            pricing_service: Instance of PricingService
            region: AWS region
        """
        self.pricing = pricing_service
        self.region = region

    def get_ec2_ondemand_price(self, instance_type: str, **kwargs) -> Optional[float]:
        """Get the on-demand price for an EC2 instance type.

        Args:
            instance_type: Type of the EC2 instance
            **kwargs: Additional arguments to pass to the pricing service

        Returns:
            Hourly price in USD or None if not found
        """
        # Set default values if not provided
        kwargs.setdefault('operating_system', 'Linux')
        kwargs.setdefault('tenancy', 'Shared')
        kwargs.setdefault('capacity_status', 'Used')
        kwargs.setdefault('region', self.region)

        return self.pricing.get_ec2_ondemand_price(instance_type=instance_type, **kwargs)

    def get_instance_price(
        self,
        instance_type: str,
        lifecycle: InstanceLifecycle
    ) -> float:
        """Get the price for an instance type with the given lifecycle.

        Args:
            instance_type: Type of the EC2 instance
            lifecycle: Instance lifecycle

        Returns:
            Hourly price in USD
        """
        try:
            hourly_price = self.get_ec2_ondemand_price(
                instance_type=instance_type,
                operating_system='Linux',
                tenancy='Shared',
                capacity_status='Used',
                region=self.region
            )

            if hourly_price is None:
                logger.warning("Could not get price for instance type %s", instance_type)
                return 0.0

            # Apply lifecycle-based discounts
            if lifecycle == InstanceLifecycle.RESERVED:
                hourly_price *= 0.6  # 40% discount for reserved instances
            elif lifecycle == InstanceLifecycle.SPOT:
                hourly_price *= 0.7  # 30% discount for spot instances

            logger.debug(
                "Price for %s (%s): $%.4f/hour",
                instance_type,
                lifecycle.value,
                hourly_price
            )
            return hourly_price

        except Exception as e:
            logger.error("Error getting price for %s: %s", instance_type, str(e))
            return 0.0
