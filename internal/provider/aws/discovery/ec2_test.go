package discovery

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockEC2Discovery struct {
	instances *ec2.DescribeInstancesOutput
	volumes   *ec2.DescribeVolumesOutput
	natGWs    *ec2.DescribeNatGatewaysOutput
}

func (m *mockEC2Discovery) DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	return m.instances, nil
}

func (m *mockEC2Discovery) DescribeVolumes(ctx context.Context, params *ec2.DescribeVolumesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVolumesOutput, error) {
	return m.volumes, nil
}

func (m *mockEC2Discovery) DescribeNatGateways(ctx context.Context, params *ec2.DescribeNatGatewaysInput, optFns ...func(*ec2.Options)) (*ec2.DescribeNatGatewaysOutput, error) {
	return m.natGWs, nil
}

func TestEC2Discoverer(t *testing.T) {
	instanceID := "i-abc123"
	nameKey := "Name"
	nameVal := "web-server"
	platform := "Linux/UNIX"

	mock := &mockEC2Discovery{
		instances: &ec2.DescribeInstancesOutput{
			Reservations: []ec2types.Reservation{
				{
					Instances: []ec2types.Instance{
						{
							InstanceId:      &instanceID,
							InstanceType:    ec2types.InstanceTypeM5Xlarge,
							PlatformDetails: &platform,
							State: &ec2types.InstanceState{
								Name: ec2types.InstanceStateNameRunning,
							},
							Tags: []ec2types.Tag{
								{Key: &nameKey, Value: &nameVal},
							},
						},
					},
				},
			},
		},
		volumes: &ec2.DescribeVolumesOutput{
			Volumes: []ec2types.Volume{},
		},
	}

	d := NewEC2Discoverer(func(region string) EC2API { return mock })

	assert.Equal(t, "Amazon Elastic Compute Cloud - Compute", d.ServiceName())

	resources, err := d.Discover(context.Background(), "123456789012", "us-east-1")
	require.NoError(t, err)
	require.Len(t, resources, 1)
	assert.Equal(t, "i-abc123", resources[0].ResourceID)
	assert.Equal(t, "ec2:instance", resources[0].ResourceType)
	assert.Equal(t, "web-server", resources[0].Name)
	assert.Equal(t, "running", resources[0].State)
	assert.Contains(t, resources[0].Spec, "m5.xlarge")
}
