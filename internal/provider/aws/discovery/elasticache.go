package discovery

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/elasticache"

	"github.com/helmcode/finops-cli/internal/provider"
)

// ElastiCacheAPI defines the ElastiCache operations needed for discovery.
type ElastiCacheAPI interface {
	DescribeCacheClusters(ctx context.Context, params *elasticache.DescribeCacheClustersInput, optFns ...func(*elasticache.Options)) (*elasticache.DescribeCacheClustersOutput, error)
}

// ElastiCacheClientFactory creates ElastiCache clients for specific regions.
type ElastiCacheClientFactory func(region string) ElastiCacheAPI

// ElastiCacheDiscoverer discovers ElastiCache clusters.
type ElastiCacheDiscoverer struct {
	clientFactory ElastiCacheClientFactory
}

// NewElastiCacheDiscoverer creates a new ElastiCache resource discoverer.
func NewElastiCacheDiscoverer(factory ElastiCacheClientFactory) *ElastiCacheDiscoverer {
	return &ElastiCacheDiscoverer{clientFactory: factory}
}

func (d *ElastiCacheDiscoverer) ServiceName() string {
	return "Amazon ElastiCache"
}

func (d *ElastiCacheDiscoverer) Discover(ctx context.Context, accountID, region string) ([]provider.Resource, error) {
	client := d.clientFactory(region)

	output, err := client.DescribeCacheClusters(ctx, &elasticache.DescribeCacheClustersInput{})
	if err != nil {
		return nil, fmt.Errorf("describing ElastiCache clusters: %w", err)
	}

	var resources []provider.Resource
	for _, cluster := range output.CacheClusters {
		spec := map[string]interface{}{
			"cache_node_type": safeStr(cluster.CacheNodeType),
			"engine":          safeStr(cluster.Engine),
			"engine_version":  safeStr(cluster.EngineVersion),
			"num_cache_nodes": cluster.NumCacheNodes,
		}
		specJSON, _ := json.Marshal(spec)

		resources = append(resources, provider.Resource{
			Provider:     "aws",
			AccountID:    accountID,
			Service:      "Amazon ElastiCache",
			ResourceID:   safeStr(cluster.ARN),
			ResourceType: "elasticache:cluster",
			Name:         safeStr(cluster.CacheClusterId),
			Region:       region,
			Spec:         string(specJSON),
			Tags:         "{}",
			State:        safeStr(cluster.CacheClusterStatus),
		})
	}

	return resources, nil
}
