from typing import List, Dict, Any
import boto3
from botocore.exceptions import ClientError

class EC2Inventory:
    """Class to handle EC2 instance inventory."""

    def __init__(self, region: str = 'us-east-1'):
        """Initialize the EC2 client.

        Args:
            region: AWS region to get inventory from
        """
        self.region = region
        self.client = boto3.client('ec2', region_name=region)

    def get_all_instances(self) -> List[Dict[str, Any]]:
        """Get all EC2 instances in the region.

        Returns:
            List of dictionaries containing instance information
        """
        instances = []

        try:
            # Get all instances
            paginator = self.client.get_paginator('describe_instances')
            for page in paginator.paginate():
                for reservation in page['Reservations']:
                    for instance in reservation['Instances']:
                        instance_info = {
                            'InstanceId': instance['InstanceId'],
                            'InstanceType': instance['InstanceType'],
                            'State': instance['State']['Name'],
                            'LaunchTime': instance['LaunchTime'].strftime('%Y-%m-%d %H:%M:%S'),
                            'VpcId': instance.get('VpcId', 'N/A'),
                            'SubnetId': instance.get('SubnetId', 'N/A'),
                            'PrivateIpAddress': instance.get('PrivateIpAddress', 'N/A'),
                            'PublicIpAddress': instance.get('PublicIpAddress', 'N/A'),
                            'InstanceLifecycle': instance.get('InstanceLifecycle', 'on-demand').lower(),
                            'Tags': {tag['Key']: tag['Value'] for tag in instance.get('Tags', []) 
                                    if tag['Key'] in ('Name', 'Environment', 'Owner', 'aws:ec2spot:fleet-request-id')}
                        }

                        # Check if instance is spot
                        if 'aws:ec2spot:fleet-request-id' in instance_info['Tags']:
                            instance_info['InstanceLifecycle'] = 'spot'

                        instances.append(instance_info)

            # Get reserved instances information
            self._add_reserved_instances_info(instances)

        except ClientError as e:
            print(f"Error getting EC2 instances: {str(e)}")

        return instances

    def _add_reserved_instances_info(self, instances: List[Dict[str, Any]]) -> None:
        """Add reserved instances information to the instances list.

        Args:
            instances: List of instance dictionaries to update
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
                if instance['State'] != 'running':
                    continue

                instance_type = instance['InstanceType']
                if instance_type in ri_mapping and ri_mapping[instance_type] > 0:
                    instance['InstanceLifecycle'] = 'reserved'
                    ri_mapping[instance_type] -= 1

        except Exception as e:
            print(f"Warning: Could not get reserved instances info: {str(e)}")

    def get_instance_types_usage(self) -> Dict[str, Dict]:
        """Get count of instances by instance type and lifecycle.

        Returns:
            Dictionary with instance types as keys and usage info as values
        """
        instances = self.get_all_instances()
        instance_types = {}

        for instance in instances:
            if instance['State'] != 'running':
                continue

            instance_type = instance['InstanceType']
            lifecycle = instance.get('InstanceLifecycle', 'on-demand')

            if instance_type not in instance_types:
                instance_types[instance_type] = {
                    'on-demand': 0,
                    'spot': 0,
                    'reserved': 0,
                    'total': 0
                }

            instance_types[instance_type][lifecycle] += 1
            instance_types[instance_type]['total'] += 1

        return instance_types
