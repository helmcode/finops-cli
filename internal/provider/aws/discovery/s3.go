package discovery

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/helmcode/finops-cli/internal/provider"
)

// S3API defines the S3 operations needed for discovery.
type S3API interface {
	ListBuckets(ctx context.Context, params *s3.ListBucketsInput, optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, error)
}

// S3Discoverer discovers S3 buckets.
type S3Discoverer struct {
	client S3API
}

// NewS3Discoverer creates a new S3 resource discoverer.
func NewS3Discoverer(client S3API) *S3Discoverer {
	return &S3Discoverer{client: client}
}

func (d *S3Discoverer) ServiceName() string {
	return "Amazon Simple Storage Service"
}

func (d *S3Discoverer) Discover(ctx context.Context, accountID, region string) ([]provider.Resource, error) {
	output, err := d.client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, fmt.Errorf("listing S3 buckets: %w", err)
	}

	var resources []provider.Resource
	for _, bucket := range output.Buckets {
		name := ""
		if bucket.Name != nil {
			name = *bucket.Name
		}

		spec := map[string]interface{}{}
		if bucket.CreationDate != nil {
			spec["creation_date"] = bucket.CreationDate.Format("2006-01-02")
		}
		specJSON, _ := json.Marshal(spec)

		resources = append(resources, provider.Resource{
			Provider:     "aws",
			AccountID:    accountID,
			Service:      "Amazon Simple Storage Service",
			ResourceID:   name,
			ResourceType: "s3:bucket",
			Name:         name,
			Region:       "", // S3 is global
			Spec:         string(specJSON),
			Tags:         "{}",
			State:        "active",
		})
	}

	return resources, nil
}
