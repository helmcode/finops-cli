package aws

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// GetActiveRegions returns the list of enabled (opted-in) regions for the account.
func (p *AWSProvider) GetActiveRegions() ([]string, error) {
	ctx := context.Background()

	output, err := p.ec2Client.DescribeRegions(ctx, &ec2.DescribeRegionsInput{
		AllRegions: boolPtr(false),
		Filters: []ec2types.Filter{
			{
				Name:   strPtr("opt-in-status"),
				Values: []string{"opt-in-not-required", "opted-in"},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("describing regions: %w", err)
	}

	regions := make([]string, 0, len(output.Regions))
	for _, r := range output.Regions {
		if r.RegionName != nil {
			regions = append(regions, *r.RegionName)
		}
	}

	slog.Info("found active regions", "count", len(regions))
	return regions, nil
}

func strPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}
