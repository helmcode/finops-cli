package discovery

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/lambda"

	"github.com/helmcode/finops-cli/internal/provider"
)

// LambdaAPI defines the Lambda operations needed for discovery.
type LambdaAPI interface {
	ListFunctions(ctx context.Context, params *lambda.ListFunctionsInput, optFns ...func(*lambda.Options)) (*lambda.ListFunctionsOutput, error)
}

// LambdaClientFactory creates Lambda clients for specific regions.
type LambdaClientFactory func(region string) LambdaAPI

// LambdaDiscoverer discovers Lambda functions.
type LambdaDiscoverer struct {
	clientFactory LambdaClientFactory
}

// NewLambdaDiscoverer creates a new Lambda resource discoverer.
func NewLambdaDiscoverer(factory LambdaClientFactory) *LambdaDiscoverer {
	return &LambdaDiscoverer{clientFactory: factory}
}

func (d *LambdaDiscoverer) ServiceName() string {
	return "AWS Lambda"
}

func (d *LambdaDiscoverer) Discover(ctx context.Context, accountID, region string) ([]provider.Resource, error) {
	client := d.clientFactory(region)

	output, err := client.ListFunctions(ctx, &lambda.ListFunctionsInput{})
	if err != nil {
		return nil, fmt.Errorf("listing Lambda functions: %w", err)
	}

	var resources []provider.Resource
	for _, fn := range output.Functions {
		spec := map[string]interface{}{
			"runtime":    string(fn.Runtime),
			"memory_mb":  fn.MemorySize,
			"timeout_s":  fn.Timeout,
			"handler":    safeStr(fn.Handler),
			"code_size":  fn.CodeSize,
		}
		specJSON, _ := json.Marshal(spec)

		resources = append(resources, provider.Resource{
			Provider:     "aws",
			AccountID:    accountID,
			Service:      "AWS Lambda",
			ResourceID:   safeStr(fn.FunctionArn),
			ResourceType: "lambda:function",
			Name:         safeStr(fn.FunctionName),
			Region:       region,
			Spec:         string(specJSON),
			Tags:         "{}",
			State:        string(fn.State),
		})
	}

	return resources, nil
}
