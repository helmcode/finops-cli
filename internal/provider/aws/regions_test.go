package aws

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock EC2 client
type mockEC2 struct {
	regions    *ec2.DescribeRegionsOutput
	instances  *ec2.DescribeInstancesOutput
	volumes    *ec2.DescribeVolumesOutput
	natGWs     *ec2.DescribeNatGatewaysOutput
	regionsErr error
}

func (m *mockEC2) DescribeRegions(ctx context.Context, params *ec2.DescribeRegionsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeRegionsOutput, error) {
	if m.regionsErr != nil {
		return nil, m.regionsErr
	}
	return m.regions, nil
}

func (m *mockEC2) DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	return m.instances, nil
}

func (m *mockEC2) DescribeVolumes(ctx context.Context, params *ec2.DescribeVolumesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVolumesOutput, error) {
	return m.volumes, nil
}

func (m *mockEC2) DescribeNatGateways(ctx context.Context, params *ec2.DescribeNatGatewaysInput, optFns ...func(*ec2.Options)) (*ec2.DescribeNatGatewaysOutput, error) {
	return m.natGWs, nil
}

func TestGetActiveRegions(t *testing.T) {
	usEast1 := "us-east-1"
	usWest2 := "us-west-2"
	euWest1 := "eu-west-1"

	p := &AWSProvider{
		ec2Client: &mockEC2{
			regions: &ec2.DescribeRegionsOutput{
				Regions: []ec2types.Region{
					{RegionName: &usEast1},
					{RegionName: &usWest2},
					{RegionName: &euWest1},
				},
			},
		},
	}

	regions, err := p.GetActiveRegions()
	require.NoError(t, err)
	assert.Len(t, regions, 3)
	assert.Contains(t, regions, "us-east-1")
	assert.Contains(t, regions, "us-west-2")
	assert.Contains(t, regions, "eu-west-1")
}

func TestGetActiveRegions_Error(t *testing.T) {
	p := &AWSProvider{
		ec2Client: &mockEC2{
			regionsErr: fmt.Errorf("api error"),
		},
	}

	_, err := p.GetActiveRegions()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "describing regions")
}
