package discovery

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"github.com/helmcode/finops-cli/internal/provider"
)

// NATDiscoverer discovers NAT Gateways (via EC2 API).
type NATDiscoverer struct {
	clientFactory EC2ClientFactory
}

// NewNATDiscoverer creates a new NAT Gateway resource discoverer.
func NewNATDiscoverer(factory EC2ClientFactory) *NATDiscoverer {
	return &NATDiscoverer{clientFactory: factory}
}

func (d *NATDiscoverer) ServiceName() string {
	return "Amazon Virtual Private Cloud"
}

func (d *NATDiscoverer) Discover(ctx context.Context, accountID, region string) ([]provider.Resource, error) {
	client := d.clientFactory(region)

	output, err := client.DescribeNatGateways(ctx, &ec2.DescribeNatGatewaysInput{})
	if err != nil {
		return nil, fmt.Errorf("describing NAT gateways: %w", err)
	}

	var resources []provider.Resource
	for _, gw := range output.NatGateways {
		spec := map[string]interface{}{
			"connectivity_type": string(gw.ConnectivityType),
		}
		if gw.SubnetId != nil {
			spec["subnet_id"] = *gw.SubnetId
		}
		if gw.VpcId != nil {
			spec["vpc_id"] = *gw.VpcId
		}
		specJSON, _ := json.Marshal(spec)

		name := ""
		tags := make(map[string]string)
		for _, tag := range gw.Tags {
			if tag.Key != nil && tag.Value != nil {
				tags[*tag.Key] = *tag.Value
				if *tag.Key == "Name" {
					name = *tag.Value
				}
			}
		}
		tagsJSON, _ := json.Marshal(tags)

		natID := ""
		if gw.NatGatewayId != nil {
			natID = *gw.NatGatewayId
		}

		resources = append(resources, provider.Resource{
			Provider:     "aws",
			AccountID:    accountID,
			Service:      "Amazon Virtual Private Cloud",
			ResourceID:   natID,
			ResourceType: "ec2:nat-gateway",
			Name:         name,
			Region:       region,
			Spec:         string(specJSON),
			Tags:         string(tagsJSON),
			State:        string(gw.State),
		})
	}

	return resources, nil
}
