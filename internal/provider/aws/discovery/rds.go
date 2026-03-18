package discovery

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/rds"

	"github.com/helmcode/finops-cli/internal/provider"
)

// RDSAPI defines the RDS operations needed for discovery.
type RDSAPI interface {
	DescribeDBInstances(ctx context.Context, params *rds.DescribeDBInstancesInput, optFns ...func(*rds.Options)) (*rds.DescribeDBInstancesOutput, error)
}

// RDSClientFactory creates RDS clients for specific regions.
type RDSClientFactory func(region string) RDSAPI

// RDSDiscoverer discovers RDS instances.
type RDSDiscoverer struct {
	clientFactory RDSClientFactory
}

// NewRDSDiscoverer creates a new RDS resource discoverer.
func NewRDSDiscoverer(factory RDSClientFactory) *RDSDiscoverer {
	return &RDSDiscoverer{clientFactory: factory}
}

func (d *RDSDiscoverer) ServiceName() string {
	return "Amazon Relational Database Service"
}

func (d *RDSDiscoverer) Discover(ctx context.Context, accountID, region string) ([]provider.Resource, error) {
	client := d.clientFactory(region)

	output, err := client.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{})
	if err != nil {
		return nil, fmt.Errorf("describing RDS instances: %w", err)
	}

	var resources []provider.Resource
	for _, db := range output.DBInstances {
		spec := map[string]interface{}{
			"instance_class":    safeStr(db.DBInstanceClass),
			"engine":            safeStr(db.Engine),
			"engine_version":    safeStr(db.EngineVersion),
			"storage_gb":        db.AllocatedStorage,
			"multi_az":          db.MultiAZ,
			"storage_encrypted": db.StorageEncrypted,
		}
		specJSON, _ := json.Marshal(spec)

		resources = append(resources, provider.Resource{
			Provider:     "aws",
			AccountID:    accountID,
			Service:      "Amazon Relational Database Service",
			ResourceID:   safeStr(db.DBInstanceArn),
			ResourceType: "rds:db",
			Name:         safeStr(db.DBInstanceIdentifier),
			Region:       region,
			Spec:         string(specJSON),
			Tags:         "{}",
			State:        safeStr(db.DBInstanceStatus),
		})
	}

	return resources, nil
}

func safeStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
