package discovery

import (
	"context"
	"testing"

	"github.com/helmcode/finops-cli/internal/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockDiscoverer is a simple discoverer for testing.
type mockDiscoverer struct {
	serviceName string
	resources   []provider.Resource
	err         error
}

func (m *mockDiscoverer) ServiceName() string { return m.serviceName }
func (m *mockDiscoverer) Discover(ctx context.Context, accountID, region string) ([]provider.Resource, error) {
	return m.resources, m.err
}

func TestRegistryLookup(t *testing.T) {
	r := NewRegistry()

	ec2Mock := &mockDiscoverer{serviceName: "Amazon Elastic Compute Cloud - Compute"}
	rdsMock := &mockDiscoverer{serviceName: "Amazon Relational Database Service"}

	r.Register(ec2Mock)
	r.Register(rdsMock)

	// Lookup existing
	d := r.Lookup("Amazon Elastic Compute Cloud - Compute")
	require.NotNil(t, d)
	assert.Equal(t, "Amazon Elastic Compute Cloud - Compute", d.ServiceName())

	// Lookup non-existing
	d = r.Lookup("Amazon Managed Blockchain")
	assert.Nil(t, d)
}

func TestRegistryHasDiscoverer(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockDiscoverer{serviceName: "Amazon S3"})

	assert.True(t, r.HasDiscoverer("Amazon S3"))
	assert.False(t, r.HasDiscoverer("Amazon Redshift"))
}

func TestRegistrySupportedServices(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockDiscoverer{serviceName: "Amazon EC2"})
	r.Register(&mockDiscoverer{serviceName: "Amazon RDS"})
	r.Register(&mockDiscoverer{serviceName: "Amazon S3"})

	services := r.SupportedServices()
	assert.Len(t, services, 3)
	assert.Contains(t, services, "Amazon EC2")
	assert.Contains(t, services, "Amazon RDS")
	assert.Contains(t, services, "Amazon S3")
}
