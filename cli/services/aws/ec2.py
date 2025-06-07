from typing import Dict, List, Optional
import logging
import boto3
from botocore.exceptions import ClientError, BotoCoreError
from cli.models.ec2 import EC2Instance, InstanceLifecycle

logger = logging.getLogger(__name__)

class EC2Service:
    """Service for interacting with AWS EC2."""

    def __init__(self, region: str = 'us-east-1', profile_name: Optional[str] = None):
        """Initialize the EC2 service.

        Args:
            region: AWS region to operate in
            profile_name: Optional AWS profile name for authentication
        """
        self.region = region
        self.profile_name = profile_name
        self._client = None
        self._session = None

    @property
    def session(self):
        """Lazily create a boto3 session."""
        if self._session is None:
            session_kwargs = {'region_name': self.region}
            if self.profile_name:
                session_kwargs['profile_name'] = self.profile_name
            self._session = boto3.Session(**session_kwargs)
        return self._session

    @property
    def client(self):
        """Lazily create an EC2 client."""
        if self._client is None:
            self._client = self.session.client('ec2')
        return self._client

    def get_all_instances(self) -> List[EC2Instance]:
        """Get all EC2 instances in the region.

        Returns:
            List of EC2Instance objects

        Raises:
            ClientError: If there's an error with the AWS API call
            BotoCoreError: For general boto3 errors
        """
        try:
            paginator = self.client.get_paginator('describe_instances')
            instances = []

            for page in paginator.paginate():
                for reservation in page.get('Reservations', []):
                    for instance_data in reservation.get('Instances', []):
                        try:
                            instance = EC2Instance.from_boto3_instance(instance_data, self.region)
                            instances.append(instance)
                        except (KeyError, ValueError) as e:
                            logger.warning("Skipping instance due to parsing error: %s", e)

            # Add reserved instances information
            self._add_reserved_instances_info(instances)
            return instances

        except (ClientError, BotoCoreError) as e:
            logger.error("Error retrieving EC2 instances: %s", e)
            raise

    def _add_reserved_instances_info(self, instances: List[EC2Instance]) -> None:
        """Add reserved instances information to the instances list.

        Args:
            instances: List of EC2Instance objects to update
        """
        try:
            # Get all reserved instances in the region
            reserved_instances = self.client.describe_reserved_instances(
                Filters=[
                    {'Name': 'state', 'Values': ['active']},
                    {'Name': 'scope', 'Values': ['Region']}
                ]
            )

            # Create a mapping of instance type to reserved instance count
            ri_mapping = {}
            for ri in reserved_instances.get('ReservedInstances', []):
                if ri['State'] == 'active':
                    instance_type = ri['InstanceType']
                    count = ri['InstanceCount']
                    ri_mapping[instance_type] = ri_mapping.get(instance_type, 0) + count

            # Mark instances as reserved if applicable
            for instance in instances:
                if instance.state.lower() != 'running':
                    continue

                if instance.instance_type in ri_mapping and ri_mapping[instance.instance_type] > 0:
                    instance.lifecycle = InstanceLifecycle.RESERVED
                    ri_mapping[instance.instance_type] -= 1

        except Exception as e:
            logger.warning("Could not get reserved instances info: %s", e)

    def get_instance_types_usage(self) -> Dict[str, Dict]:
        """Get count of instances by instance type and lifecycle.

        Returns:
            Dictionary with instance types as keys and usage info as values
        """
        instances = self.get_all_instances()
        instance_types = {}

        for instance in instances:
            if instance.state.lower() != 'running':
                continue

            instance_type = instance.instance_type

            if instance_type not in instance_types:
                instance_types[instance_type] = {
                    InstanceLifecycle.ON_DEMAND.value: 0,
                    InstanceLifecycle.SPOT.value: 0,
                    InstanceLifecycle.RESERVED.value: 0,
                    'total': 0
                }

            instance_types[instance_type][instance.lifecycle.value] += 1
            instance_types[instance_type]['total'] += 1

        return instance_types
