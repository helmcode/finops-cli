package discovery

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/cloudfront"

	"github.com/helmcode/finops-cli/internal/provider"
)

// CloudFrontAPI defines the CloudFront operations needed for discovery.
type CloudFrontAPI interface {
	ListDistributions(ctx context.Context, params *cloudfront.ListDistributionsInput, optFns ...func(*cloudfront.Options)) (*cloudfront.ListDistributionsOutput, error)
}

// CloudFrontDiscoverer discovers CloudFront distributions.
type CloudFrontDiscoverer struct {
	client CloudFrontAPI
}

// NewCloudFrontDiscoverer creates a new CloudFront resource discoverer.
func NewCloudFrontDiscoverer(client CloudFrontAPI) *CloudFrontDiscoverer {
	return &CloudFrontDiscoverer{client: client}
}

func (d *CloudFrontDiscoverer) ServiceName() string {
	return "Amazon CloudFront"
}

func (d *CloudFrontDiscoverer) Discover(ctx context.Context, accountID, region string) ([]provider.Resource, error) {
	output, err := d.client.ListDistributions(ctx, &cloudfront.ListDistributionsInput{})
	if err != nil {
		return nil, fmt.Errorf("listing CloudFront distributions: %w", err)
	}

	if output.DistributionList == nil {
		return nil, nil
	}

	var resources []provider.Resource
	for _, dist := range output.DistributionList.Items {
		spec := map[string]interface{}{
			"domain_name":    safeStr(dist.DomainName),
			"status":         safeStr(dist.Status),
			"http_version":   string(dist.HttpVersion),
			"price_class":    string(dist.PriceClass),
			"enabled":        dist.Enabled,
		}
		specJSON, _ := json.Marshal(spec)

		state := "disabled"
		if dist.Enabled != nil && *dist.Enabled {
			state = "enabled"
		}

		resources = append(resources, provider.Resource{
			Provider:     "aws",
			AccountID:    accountID,
			Service:      "Amazon CloudFront",
			ResourceID:   safeStr(dist.ARN),
			ResourceType: "cloudfront:distribution",
			Name:         safeStr(dist.Id),
			Region:       "global",
			Spec:         string(specJSON),
			Tags:         "{}",
			State:        state,
		})
	}

	return resources, nil
}
