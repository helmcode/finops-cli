package discovery

import (
	"context"
	"log/slog"

	"github.com/helmcode/finops-cli/internal/provider"
)

// ResourceDiscoverer defines the interface for service-specific resource discovery.
type ResourceDiscoverer interface {
	// ServiceName returns the Cost Explorer service name this discoverer handles.
	ServiceName() string

	// Discover finds resources for the given account in a specific region.
	Discover(ctx context.Context, accountID, region string) ([]provider.Resource, error)
}

// Registry maps Cost Explorer service names to their resource discoverers.
type Registry struct {
	discoverers map[string]ResourceDiscoverer
}

// NewRegistry creates a registry with all v1 discovery adapters.
func NewRegistry() *Registry {
	r := &Registry{
		discoverers: make(map[string]ResourceDiscoverer),
	}
	return r
}

// Register adds a discoverer to the registry.
func (r *Registry) Register(d ResourceDiscoverer) {
	r.discoverers[d.ServiceName()] = d
	slog.Debug("registered resource discoverer", "service", d.ServiceName())
}

// Lookup returns the discoverer for a Cost Explorer service name, or nil if none exists.
func (r *Registry) Lookup(serviceName string) ResourceDiscoverer {
	return r.discoverers[serviceName]
}

// HasDiscoverer returns true if a discoverer exists for the service.
func (r *Registry) HasDiscoverer(serviceName string) bool {
	_, ok := r.discoverers[serviceName]
	return ok
}

// SupportedServices returns a list of all registered service names.
func (r *Registry) SupportedServices() []string {
	services := make([]string, 0, len(r.discoverers))
	for name := range r.discoverers {
		services = append(services, name)
	}
	return services
}
