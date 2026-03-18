package aws

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/service/organizations"
	orgtypes "github.com/aws/aws-sdk-go-v2/service/organizations/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/smithy-go"

	"github.com/helmcode/finops-cli/internal/provider"
)

// DetectAccountMode determines if the current credentials belong to an
// AWS Organization or a standalone account.
func (p *AWSProvider) DetectAccountMode() (provider.AccountMode, error) {
	ctx := context.Background()

	// Step 1: Get caller identity
	identity, err := p.stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return provider.AccountMode{}, fmt.Errorf("getting caller identity: %w", err)
	}
	p.accountID = *identity.Account
	slog.Info("identified AWS account", "account_id", p.accountID)

	// Step 2: Try to describe the organization
	orgOutput, err := p.orgClient.DescribeOrganization(ctx, &organizations.DescribeOrganizationInput{})
	if err != nil {
		if isAccessDeniedOrOrgNotFound(err) {
			slog.Info("single account mode (no organization access)")
			return provider.AccountMode{
				IsOrganization: false,
				ManagementID:   p.accountID,
				Accounts: []provider.Account{
					{ID: p.accountID, Name: "current"},
				},
			}, nil
		}
		return provider.AccountMode{}, fmt.Errorf("describing organization: %w", err)
	}

	managementID := ""
	if orgOutput.Organization != nil && orgOutput.Organization.MasterAccountId != nil {
		managementID = *orgOutput.Organization.MasterAccountId
	}

	slog.Info("AWS Organization detected", "management_account", managementID)

	return provider.AccountMode{
		IsOrganization: true,
		ManagementID:   managementID,
	}, nil
}

// ListAccounts returns the list of accounts to scan. If the credentials are
// for an organization, it lists all member accounts. Otherwise, it returns
// only the current account. An optional filter restricts to specific account IDs.
func (p *AWSProvider) ListAccounts(filter []string) ([]provider.Account, error) {
	ctx := context.Background()

	// If we don't have the account ID yet, detect it
	if p.accountID == "" {
		identity, err := p.stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
		if err != nil {
			return nil, fmt.Errorf("getting caller identity: %w", err)
		}
		p.accountID = *identity.Account
	}

	// Try listing organization accounts
	accounts, err := p.listOrgAccounts(ctx)
	if err != nil {
		if isAccessDeniedOrOrgNotFound(err) {
			slog.Debug("cannot list organization accounts, using single account mode")
			return []provider.Account{
				{ID: p.accountID, Name: "current"},
			}, nil
		}
		return nil, fmt.Errorf("listing organization accounts: %w", err)
	}

	// Apply filter if provided
	if len(filter) > 0 {
		accounts = filterAccounts(accounts, filter)
	}

	return accounts, nil
}

func (p *AWSProvider) listOrgAccounts(ctx context.Context) ([]provider.Account, error) {
	var accounts []provider.Account
	var nextToken *string

	for {
		output, err := p.orgClient.ListAccounts(ctx, &organizations.ListAccountsInput{
			NextToken: nextToken,
		})
		if err != nil {
			return nil, err
		}

		for _, acct := range output.Accounts {
			if acct.Status == orgtypes.AccountStatusActive {
				name := ""
				if acct.Name != nil {
					name = *acct.Name
				}
				accounts = append(accounts, provider.Account{
					ID:   *acct.Id,
					Name: name,
				})
			}
		}

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	slog.Info("found organization accounts", "count", len(accounts))
	return accounts, nil
}

func filterAccounts(accounts []provider.Account, filter []string) []provider.Account {
	filterSet := make(map[string]bool, len(filter))
	for _, id := range filter {
		filterSet[id] = true
	}

	var filtered []provider.Account
	for _, acct := range accounts {
		if filterSet[acct.ID] {
			filtered = append(filtered, acct)
		}
	}
	return filtered
}

func isAccessDeniedOrOrgNotFound(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "AccessDeniedException", "AWSOrganizationsNotInUseException":
			return true
		}
	}
	return false
}
