package aws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/organizations"
	orgtypes "github.com/aws/aws-sdk-go-v2/service/organizations/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/smithy-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock STS client
type mockSTS struct {
	accountID string
	err       error
}

func (m *mockSTS) GetCallerIdentity(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &sts.GetCallerIdentityOutput{
		Account: &m.accountID,
	}, nil
}

// Mock Organizations client
type mockOrganizations struct {
	org      *organizations.DescribeOrganizationOutput
	accounts *organizations.ListAccountsOutput
	descErr  error
	listErr  error
}

func (m *mockOrganizations) DescribeOrganization(ctx context.Context, params *organizations.DescribeOrganizationInput, optFns ...func(*organizations.Options)) (*organizations.DescribeOrganizationOutput, error) {
	if m.descErr != nil {
		return nil, m.descErr
	}
	return m.org, nil
}

func (m *mockOrganizations) ListAccounts(ctx context.Context, params *organizations.ListAccountsInput, optFns ...func(*organizations.Options)) (*organizations.ListAccountsOutput, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.accounts, nil
}

func TestDetectAccountMode_SingleAccount(t *testing.T) {
	p := &AWSProvider{
		stsClient: &mockSTS{accountID: "123456789012"},
		orgClient: &mockOrganizations{
			descErr: &smithy.GenericAPIError{Code: "AWSOrganizationsNotInUseException", Message: "not in use"},
		},
	}

	mode, err := p.DetectAccountMode()
	require.NoError(t, err)
	assert.False(t, mode.IsOrganization)
	assert.Equal(t, "123456789012", mode.ManagementID)
	assert.Len(t, mode.Accounts, 1)
	assert.Equal(t, "123456789012", mode.Accounts[0].ID)
}

func TestDetectAccountMode_Organization(t *testing.T) {
	mgmtID := "111111111111"
	p := &AWSProvider{
		stsClient: &mockSTS{accountID: "123456789012"},
		orgClient: &mockOrganizations{
			org: &organizations.DescribeOrganizationOutput{
				Organization: &orgtypes.Organization{
					MasterAccountId: &mgmtID,
				},
			},
		},
	}

	mode, err := p.DetectAccountMode()
	require.NoError(t, err)
	assert.True(t, mode.IsOrganization)
	assert.Equal(t, "111111111111", mode.ManagementID)
}

func TestListAccounts_SingleAccount(t *testing.T) {
	p := &AWSProvider{
		stsClient: &mockSTS{accountID: "123456789012"},
		orgClient: &mockOrganizations{
			listErr: &smithy.GenericAPIError{Code: "AccessDeniedException", Message: "denied"},
		},
	}

	accounts, err := p.ListAccounts(nil)
	require.NoError(t, err)
	require.Len(t, accounts, 1)
	assert.Equal(t, "123456789012", accounts[0].ID)
}

func TestListAccounts_Organization(t *testing.T) {
	acctID1 := "111111111111"
	acctName1 := "prod"
	acctID2 := "222222222222"
	acctName2 := "dev"
	acctID3 := "333333333333"
	acctName3 := "suspended"

	p := &AWSProvider{
		accountID: "111111111111",
		stsClient: &mockSTS{accountID: "111111111111"},
		orgClient: &mockOrganizations{
			accounts: &organizations.ListAccountsOutput{
				Accounts: []orgtypes.Account{
					{Id: &acctID1, Name: &acctName1, Status: orgtypes.AccountStatusActive},
					{Id: &acctID2, Name: &acctName2, Status: orgtypes.AccountStatusActive},
					{Id: &acctID3, Name: &acctName3, Status: orgtypes.AccountStatusSuspended},
				},
			},
		},
	}

	accounts, err := p.ListAccounts(nil)
	require.NoError(t, err)
	assert.Len(t, accounts, 2) // suspended account excluded
}

func TestListAccounts_WithFilter(t *testing.T) {
	acctID1 := "111111111111"
	acctName1 := "prod"
	acctID2 := "222222222222"
	acctName2 := "dev"

	p := &AWSProvider{
		accountID: "111111111111",
		stsClient: &mockSTS{accountID: "111111111111"},
		orgClient: &mockOrganizations{
			accounts: &organizations.ListAccountsOutput{
				Accounts: []orgtypes.Account{
					{Id: &acctID1, Name: &acctName1, Status: orgtypes.AccountStatusActive},
					{Id: &acctID2, Name: &acctName2, Status: orgtypes.AccountStatusActive},
				},
			},
		},
	}

	accounts, err := p.ListAccounts([]string{"222222222222"})
	require.NoError(t, err)
	require.Len(t, accounts, 1)
	assert.Equal(t, "222222222222", accounts[0].ID)
}

func TestIsAccessDeniedOrOrgNotFound(t *testing.T) {
	tests := []struct {
		code     string
		expected bool
	}{
		{"AccessDeniedException", true},
		{"AWSOrganizationsNotInUseException", true},
		{"InternalServerError", false},
	}

	for _, tc := range tests {
		err := &smithy.GenericAPIError{Code: tc.code, Message: "test"}
		assert.Equal(t, tc.expected, isAccessDeniedOrOrgNotFound(err), "code: %s", tc.code)
	}
}
