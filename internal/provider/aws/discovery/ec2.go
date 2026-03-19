package discovery

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/helmcode/finops-cli/internal/provider"
)

// EC2API defines the EC2 operations needed for discovery.
type EC2API interface {
	DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
	DescribeVolumes(ctx context.Context, params *ec2.DescribeVolumesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVolumesOutput, error)
	DescribeNatGateways(ctx context.Context, params *ec2.DescribeNatGatewaysInput, optFns ...func(*ec2.Options)) (*ec2.DescribeNatGatewaysOutput, error)
}

// EC2ClientFactory creates EC2 clients for specific regions.
type EC2ClientFactory func(region string) EC2API

// EC2Discoverer discovers EC2 instances and EBS volumes.
type EC2Discoverer struct {
	clientFactory EC2ClientFactory
}

// NewEC2Discoverer creates a new EC2 resource discoverer.
func NewEC2Discoverer(factory EC2ClientFactory) *EC2Discoverer {
	return &EC2Discoverer{clientFactory: factory}
}

func (d *EC2Discoverer) ServiceName() string {
	return "Amazon Elastic Compute Cloud - Compute"
}

func (d *EC2Discoverer) Discover(ctx context.Context, accountID, region string) ([]provider.Resource, error) {
	client := d.clientFactory(region)
	var resources []provider.Resource

	// Discover instances
	instances, err := d.discoverInstances(ctx, client, accountID, region)
	if err != nil {
		return nil, fmt.Errorf("discovering EC2 instances: %w", err)
	}
	resources = append(resources, instances...)

	// Discover volumes
	volumes, err := d.discoverVolumes(ctx, client, accountID, region)
	if err != nil {
		return nil, fmt.Errorf("discovering EBS volumes: %w", err)
	}
	resources = append(resources, volumes...)

	return resources, nil
}

func (d *EC2Discoverer) discoverInstances(ctx context.Context, client EC2API, accountID, region string) ([]provider.Resource, error) {
	var resources []provider.Resource

	output, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{})
	if err != nil {
		return nil, err
	}

	for _, reservation := range output.Reservations {
		for _, inst := range reservation.Instances {
			lifecycle := "on-demand"
			if inst.InstanceLifecycle == ec2types.InstanceLifecycleTypeSpot {
				lifecycle = "spot"
			}

			spec := map[string]string{
				"instance_type": string(inst.InstanceType),
				"lifecycle":     lifecycle,
			}
			if inst.PlatformDetails != nil {
				spec["platform"] = *inst.PlatformDetails
			}
			specJSON, _ := json.Marshal(spec)

			tags := extractTags(inst.Tags)
			tagsJSON, _ := json.Marshal(tags)

			name := ""
			for _, tag := range inst.Tags {
				if tag.Key != nil && *tag.Key == "Name" && tag.Value != nil {
					name = *tag.Value
				}
			}

			instanceID := ""
			if inst.InstanceId != nil {
				instanceID = *inst.InstanceId
			}

			resources = append(resources, provider.Resource{
				Provider:     "aws",
				AccountID:    accountID,
				Service:      "Amazon Elastic Compute Cloud - Compute",
				ResourceID:   instanceID,
				ResourceType: "ec2:instance",
				Name:         name,
				Region:       region,
				Spec:         string(specJSON),
				Tags:         string(tagsJSON),
				State:        string(inst.State.Name),
			})
		}
	}

	return resources, nil
}

func (d *EC2Discoverer) discoverVolumes(ctx context.Context, client EC2API, accountID, region string) ([]provider.Resource, error) {
	var resources []provider.Resource

	output, err := client.DescribeVolumes(ctx, &ec2.DescribeVolumesInput{})
	if err != nil {
		return nil, err
	}

	for _, vol := range output.Volumes {
		spec := map[string]interface{}{
			"volume_type": string(vol.VolumeType),
			"size_gb":     vol.Size,
			"iops":        vol.Iops,
			"encrypted":   vol.Encrypted,
		}
		specJSON, _ := json.Marshal(spec)

		tags := extractTags(vol.Tags)
		tagsJSON, _ := json.Marshal(tags)

		name := ""
		for _, tag := range vol.Tags {
			if tag.Key != nil && *tag.Key == "Name" && tag.Value != nil {
				name = *tag.Value
			}
		}

		volumeID := ""
		if vol.VolumeId != nil {
			volumeID = *vol.VolumeId
		}

		resources = append(resources, provider.Resource{
			Provider:     "aws",
			AccountID:    accountID,
			Service:      "Amazon Elastic Compute Cloud - Compute",
			ResourceID:   volumeID,
			ResourceType: "ec2:volume",
			Name:         name,
			Region:       region,
			Spec:         string(specJSON),
			Tags:         string(tagsJSON),
			State:        string(vol.State),
		})
	}

	return resources, nil
}

func extractTags(tags []ec2types.Tag) map[string]string {
	result := make(map[string]string)
	for _, tag := range tags {
		if tag.Key != nil && tag.Value != nil {
			result[*tag.Key] = *tag.Value
		}
	}
	return result
}
