package discovery

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecs"

	"github.com/helmcode/finops-cli/internal/provider"
)

// ECSAPI defines the ECS operations needed for discovery.
type ECSAPI interface {
	ListClusters(ctx context.Context, params *ecs.ListClustersInput, optFns ...func(*ecs.Options)) (*ecs.ListClustersOutput, error)
	DescribeClusters(ctx context.Context, params *ecs.DescribeClustersInput, optFns ...func(*ecs.Options)) (*ecs.DescribeClustersOutput, error)
}

// ECSClientFactory creates ECS clients for specific regions.
type ECSClientFactory func(region string) ECSAPI

// ECSDiscoverer discovers ECS clusters and services.
type ECSDiscoverer struct {
	clientFactory ECSClientFactory
}

// NewECSDiscoverer creates a new ECS resource discoverer.
func NewECSDiscoverer(factory ECSClientFactory) *ECSDiscoverer {
	return &ECSDiscoverer{clientFactory: factory}
}

func (d *ECSDiscoverer) ServiceName() string {
	return "Amazon Elastic Container Service"
}

func (d *ECSDiscoverer) Discover(ctx context.Context, accountID, region string) ([]provider.Resource, error) {
	client := d.clientFactory(region)

	listOutput, err := client.ListClusters(ctx, &ecs.ListClustersInput{})
	if err != nil {
		return nil, fmt.Errorf("listing ECS clusters: %w", err)
	}

	if len(listOutput.ClusterArns) == 0 {
		return nil, nil
	}

	descOutput, err := client.DescribeClusters(ctx, &ecs.DescribeClustersInput{
		Clusters: listOutput.ClusterArns,
	})
	if err != nil {
		return nil, fmt.Errorf("describing ECS clusters: %w", err)
	}

	var resources []provider.Resource
	for _, cluster := range descOutput.Clusters {
		spec := map[string]interface{}{
			"running_tasks":      cluster.RunningTasksCount,
			"pending_tasks":      cluster.PendingTasksCount,
			"active_services":    cluster.ActiveServicesCount,
			"registered_instances": cluster.RegisteredContainerInstancesCount,
		}
		specJSON, _ := json.Marshal(spec)

		resources = append(resources, provider.Resource{
			Provider:     "aws",
			AccountID:    accountID,
			Service:      "Amazon Elastic Container Service",
			ResourceID:   safeStr(cluster.ClusterArn),
			ResourceType: "ecs:cluster",
			Name:         safeStr(cluster.ClusterName),
			Region:       region,
			Spec:         string(specJSON),
			Tags:         "{}",
			State:        safeStr(cluster.Status),
		})
	}

	return resources, nil
}
