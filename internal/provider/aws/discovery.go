package aws

import (
	"context"
	"log/slog"

	"github.com/helmcode/finops-cli/internal/provider"
	"github.com/helmcode/finops-cli/internal/provider/aws/discovery"
)

// InitDiscoveryRegistry creates and populates the resource discovery registry
// with all v1 adapters using the provider's AWS clients.
func (p *AWSProvider) InitDiscoveryRegistry() *discovery.Registry {
	reg := discovery.NewRegistry()

	reg.Register(discovery.NewEC2Discoverer(func(region string) discovery.EC2API {
		return p.EC2ClientForRegion(region)
	}))
	reg.Register(discovery.NewRDSDiscoverer(func(region string) discovery.RDSAPI {
		return p.RDSClientForRegion(region)
	}))
	reg.Register(discovery.NewS3Discoverer(p.s3Client))
	reg.Register(discovery.NewLambdaDiscoverer(func(region string) discovery.LambdaAPI {
		return p.LambdaClientForRegion(region)
	}))
	reg.Register(discovery.NewECSDiscoverer(func(region string) discovery.ECSAPI {
		return p.ECSClientForRegion(region)
	}))
	reg.Register(discovery.NewElastiCacheDiscoverer(func(region string) discovery.ElastiCacheAPI {
		return p.ElastiCacheClientForRegion(region)
	}))
	reg.Register(discovery.NewNATDiscoverer(func(region string) discovery.EC2API {
		return p.EC2ClientForRegion(region)
	}))
	reg.Register(discovery.NewCloudFrontDiscoverer(p.cfClient))

	return reg
}

// DiscoverResources finds active resources for a given service in a region.
// This delegates to the discovery registry which maps service names to
// specific discovery adapters.
func (p *AWSProvider) DiscoverResources(service, region string) ([]provider.Resource, error) {
	if p.registry == nil {
		p.registry = p.InitDiscoveryRegistry()
	}

	discoverer := p.registry.Lookup(service)
	if discoverer == nil {
		slog.Debug("no resource discoverer available", "service", service)
		return nil, nil
	}

	ctx := context.Background()
	resources, err := discoverer.Discover(ctx, p.accountID, region)
	if err != nil {
		slog.Warn("resource discovery failed", "service", service, "region", region, "error", err)
		return nil, err
	}

	slog.Debug("discovered resources", "service", service, "region", region, "count", len(resources))
	return resources, nil
}
