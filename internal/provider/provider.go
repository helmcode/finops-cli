package provider

import "time"

// Provider defines the interface for cloud provider integrations.
// Each provider (AWS, Azure, GCP) implements this interface to enable
// multi-cloud support without changing core logic.
type Provider interface {
	// Name returns the provider identifier (e.g., "aws", "azure", "gcp").
	Name() string

	// DetectAccountMode determines if credentials belong to an organization
	// or a single standalone account.
	DetectAccountMode() (AccountMode, error)

	// ListAccounts returns the accounts to scan, applying the optional filter.
	// For single-account mode, returns only the current account.
	ListAccounts(filter []string) ([]Account, error)

	// FetchCosts retrieves cost data for the given parameters.
	FetchCosts(params CostParams) ([]CostRecord, error)

	// FetchCommitments retrieves commitment data (RI, SP) for the given parameters.
	FetchCommitments(params CommitmentParams) ([]CommitmentRecord, error)

	// DiscoverResources finds active resources for a service in a region.
	DiscoverResources(service, region string) ([]Resource, error)

	// GetActiveRegions returns the list of enabled regions for the provider.
	GetActiveRegions() ([]string, error)
}

// AccountMode describes whether the credentials belong to an organization
// or a standalone account.
type AccountMode struct {
	IsOrganization bool
	ManagementID   string
	Accounts       []Account
}

// Account represents a cloud provider account.
type Account struct {
	ID   string
	Name string
}

// CostParams defines the parameters for fetching cost data.
type CostParams struct {
	AccountID   string
	Start       time.Time
	End         time.Time
	Granularity string   // "MONTHLY"
	GroupBy     []string // e.g., ["SERVICE", "REGION"]
}

// CostRecord represents a single cost data point.
type CostRecord struct {
	Provider    string
	AccountID   string
	Service     string
	Region      string
	PeriodStart string
	PeriodEnd   string
	Granularity string
	Amount      float64
	Currency    string
}

// Resource represents a discovered cloud resource.
type Resource struct {
	Provider     string
	AccountID    string
	Service      string
	ResourceID   string
	ResourceType string
	Name         string
	Region       string
	Spec         string // JSON
	Tags         string // JSON
	State        string
}

// CommitmentRecord represents a cost commitment data point (RI, SP).
type CommitmentRecord struct {
	Provider           string
	AccountID          string
	CommitmentType     string // "savings_plan", "reserved_instance"
	PeriodStart        string
	PeriodEnd          string
	TotalCommitment    float64
	UsedCommitment     float64
	OnDemandEquivalent float64
	NetSavings         float64
	UtilizationPct     float64
	CoveragePct        float64
	Currency           string
}

// CommitmentParams defines parameters for fetching commitment data.
type CommitmentParams struct {
	AccountID string
	Start     time.Time
	End       time.Time
}
