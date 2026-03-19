package aws

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/helmcode/finops-cli/internal/provider"
	"github.com/helmcode/finops-cli/internal/provider/aws/discovery"
)

// STSAPI defines the STS operations used by the AWS provider.
type STSAPI interface {
	GetCallerIdentity(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error)
}

// OrganizationsAPI defines the Organizations operations used by the AWS provider.
type OrganizationsAPI interface {
	DescribeOrganization(ctx context.Context, params *organizations.DescribeOrganizationInput, optFns ...func(*organizations.Options)) (*organizations.DescribeOrganizationOutput, error)
	ListAccounts(ctx context.Context, params *organizations.ListAccountsInput, optFns ...func(*organizations.Options)) (*organizations.ListAccountsOutput, error)
}

// CostExplorerAPI defines the Cost Explorer operations used by the AWS provider.
type CostExplorerAPI interface {
	GetCostAndUsage(ctx context.Context, params *costexplorer.GetCostAndUsageInput, optFns ...func(*costexplorer.Options)) (*costexplorer.GetCostAndUsageOutput, error)
	GetSavingsPlansUtilization(ctx context.Context, params *costexplorer.GetSavingsPlansUtilizationInput, optFns ...func(*costexplorer.Options)) (*costexplorer.GetSavingsPlansUtilizationOutput, error)
	GetSavingsPlansCoverage(ctx context.Context, params *costexplorer.GetSavingsPlansCoverageInput, optFns ...func(*costexplorer.Options)) (*costexplorer.GetSavingsPlansCoverageOutput, error)
	GetReservationUtilization(ctx context.Context, params *costexplorer.GetReservationUtilizationInput, optFns ...func(*costexplorer.Options)) (*costexplorer.GetReservationUtilizationOutput, error)
	GetReservationCoverage(ctx context.Context, params *costexplorer.GetReservationCoverageInput, optFns ...func(*costexplorer.Options)) (*costexplorer.GetReservationCoverageOutput, error)
}

// EC2API defines the EC2 operations used by the AWS provider.
type EC2API interface {
	DescribeRegions(ctx context.Context, params *ec2.DescribeRegionsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeRegionsOutput, error)
	DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
	DescribeVolumes(ctx context.Context, params *ec2.DescribeVolumesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVolumesOutput, error)
	DescribeNatGateways(ctx context.Context, params *ec2.DescribeNatGatewaysInput, optFns ...func(*ec2.Options)) (*ec2.DescribeNatGatewaysOutput, error)
}

// RDSAPI defines the RDS operations used by the AWS provider.
type RDSAPI interface {
	DescribeDBInstances(ctx context.Context, params *rds.DescribeDBInstancesInput, optFns ...func(*rds.Options)) (*rds.DescribeDBInstancesOutput, error)
}

// S3API defines the S3 operations used by the AWS provider.
type S3API interface {
	ListBuckets(ctx context.Context, params *s3.ListBucketsInput, optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, error)
}

// LambdaAPI defines the Lambda operations used by the AWS provider.
type LambdaAPI interface {
	ListFunctions(ctx context.Context, params *lambda.ListFunctionsInput, optFns ...func(*lambda.Options)) (*lambda.ListFunctionsOutput, error)
}

// ECSAPI defines the ECS operations used by the AWS provider.
type ECSAPI interface {
	ListClusters(ctx context.Context, params *ecs.ListClustersInput, optFns ...func(*ecs.Options)) (*ecs.ListClustersOutput, error)
	DescribeClusters(ctx context.Context, params *ecs.DescribeClustersInput, optFns ...func(*ecs.Options)) (*ecs.DescribeClustersOutput, error)
}

// ElastiCacheAPI defines the ElastiCache operations used by the AWS provider.
type ElastiCacheAPI interface {
	DescribeCacheClusters(ctx context.Context, params *elasticache.DescribeCacheClustersInput, optFns ...func(*elasticache.Options)) (*elasticache.DescribeCacheClustersOutput, error)
}

// CloudFrontAPI defines the CloudFront operations used by the AWS provider.
type CloudFrontAPI interface {
	ListDistributions(ctx context.Context, params *cloudfront.ListDistributionsInput, optFns ...func(*cloudfront.Options)) (*cloudfront.ListDistributionsOutput, error)
}

// AWSProvider implements the provider.Provider interface for AWS.
type AWSProvider struct {
	cfg           aws.Config
	accountID     string
	stsClient     STSAPI
	orgClient     OrganizationsAPI
	ceClient      CostExplorerAPI
	ec2Client     EC2API
	rdsClient     RDSAPI
	s3Client      S3API
	lambdaClient  LambdaAPI
	ecsClient     ECSAPI
	ecacheClient  ElastiCacheAPI
	cfClient      CloudFrontAPI
	registry      *discovery.Registry
}

// Ensure AWSProvider implements provider.Provider at compile time.
var _ provider.Provider = (*AWSProvider)(nil)

// NewAWSProvider creates a new AWS provider by loading credentials from
// the default credential chain (env vars, ~/.aws/credentials, IAM roles).
func NewAWSProvider(ctx context.Context) (*AWSProvider, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	slog.Debug("AWS config loaded", "region", cfg.Region)

	p := &AWSProvider{
		cfg:          cfg,
		stsClient:    sts.NewFromConfig(cfg),
		orgClient:    organizations.NewFromConfig(cfg),
		ceClient:     costexplorer.NewFromConfig(cfg),
		ec2Client:    ec2.NewFromConfig(cfg),
		rdsClient:    rds.NewFromConfig(cfg),
		s3Client:     s3.NewFromConfig(cfg),
		lambdaClient: lambda.NewFromConfig(cfg),
		ecsClient:    ecs.NewFromConfig(cfg),
		ecacheClient: elasticache.NewFromConfig(cfg),
		cfClient:     cloudfront.NewFromConfig(cfg),
	}

	return p, nil
}

// NewAWSProviderWithClients creates an AWS provider with injected API clients (for testing).
func NewAWSProviderWithClients(
	stsClient STSAPI,
	orgClient OrganizationsAPI,
	ceClient CostExplorerAPI,
	ec2Client EC2API,
) *AWSProvider {
	return &AWSProvider{
		stsClient: stsClient,
		orgClient: orgClient,
		ceClient:  ceClient,
		ec2Client: ec2Client,
	}
}

// Name returns the provider identifier.
func (p *AWSProvider) Name() string {
	return "aws"
}

// EC2ClientForRegion creates an EC2 client configured for a specific region.
func (p *AWSProvider) EC2ClientForRegion(region string) EC2API {
	cfgCopy := p.cfg.Copy()
	cfgCopy.Region = region
	return ec2.NewFromConfig(cfgCopy)
}

// RDSClientForRegion creates an RDS client configured for a specific region.
func (p *AWSProvider) RDSClientForRegion(region string) RDSAPI {
	cfgCopy := p.cfg.Copy()
	cfgCopy.Region = region
	return rds.NewFromConfig(cfgCopy)
}

// LambdaClientForRegion creates a Lambda client configured for a specific region.
func (p *AWSProvider) LambdaClientForRegion(region string) LambdaAPI {
	cfgCopy := p.cfg.Copy()
	cfgCopy.Region = region
	return lambda.NewFromConfig(cfgCopy)
}

// ECSClientForRegion creates an ECS client configured for a specific region.
func (p *AWSProvider) ECSClientForRegion(region string) ECSAPI {
	cfgCopy := p.cfg.Copy()
	cfgCopy.Region = region
	return ecs.NewFromConfig(cfgCopy)
}

// ElastiCacheClientForRegion creates an ElastiCache client configured for a specific region.
func (p *AWSProvider) ElastiCacheClientForRegion(region string) ElastiCacheAPI {
	cfgCopy := p.cfg.Copy()
	cfgCopy.Region = region
	return elasticache.NewFromConfig(cfgCopy)
}
