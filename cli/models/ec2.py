from dataclasses import dataclass, field
from enum import Enum
from typing import Any, Dict

class InstanceLifecycle(str, Enum):
    """Represents the lifecycle of an EC2 instance."""
    ON_DEMAND = "on-demand"
    SPOT = "spot"
    RESERVED = "reserved"

@dataclass
class EC2Instance:
    """Represents an EC2 instance with its core attributes."""
    instance_id: str
    instance_type: str
    state: str
    launch_time: str  # Keep as string for backward compatibility
    vpc_id: str = "N/A"
    subnet_id: str = "N/A"
    private_ip: str = "N/A"
    public_ip: str = "N/A"
    lifecycle: InstanceLifecycle = InstanceLifecycle.ON_DEMAND
    tags: Dict[str, str] = field(default_factory=dict)
    region: str = ""

    @classmethod
    def from_boto3_instance(cls, instance_data: Dict[str, Any], region: str) -> 'EC2Instance':
        """Create an EC2Instance from boto3 instance data."""
        tags = {tag['Key']: tag['Value'] for tag in instance_data.get('Tags', [])}

        # Determine lifecycle
        lifecycle = InstanceLifecycle.ON_DEMAND
        if 'InstanceLifecycle' in instance_data:
            lifecycle = InstanceLifecycle(instance_data['InstanceLifecycle'].lower())
        elif any(k.startswith('aws:ec2spot:') for k in tags):
            lifecycle = InstanceLifecycle.SPOT

        return cls(
            instance_id=instance_data['InstanceId'],
            instance_type=instance_data['InstanceType'],
            state=instance_data['State']['Name'],
            launch_time=instance_data['LaunchTime'].strftime('%Y-%m-%d %H:%M:%S'),
            vpc_id=instance_data.get('VpcId', 'N/A'),
            subnet_id=instance_data.get('SubnetId', 'N/A'),
            private_ip=instance_data.get('PrivateIpAddress', 'N/A'),
            public_ip=instance_data.get('PublicIpAddress', 'N/A'),
            lifecycle=lifecycle,
            tags=tags,
            region=region
        )

    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary format for backward compatibility."""
        return {
            'InstanceId': self.instance_id,
            'InstanceType': self.instance_type,
            'State': self.state,
            'LaunchTime': self.launch_time,
            'VpcId': self.vpc_id,
            'SubnetId': self.subnet_id,
            'PrivateIpAddress': self.private_ip,
            'PublicIpAddress': self.public_ip,
            'InstanceLifecycle': self.lifecycle.value,
            'Tags': self.tags,
            'region': self.region
        }
