package types_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"

	"github.com/initia-labs/initia/x/mstaking/types"
)

func TestNewStakeAuthorization(t *testing.T) {
	tests := []struct {
		name              string
		allowedValidators []string
		deniedValidators  []string
		authzType         types.AuthorizationType
		amount           sdk.Coins
		expectError      bool
	}{
		{
			name:              "valid authorization with allow list",
			allowedValidators: []string{"val1", "val2"},
			deniedValidators:  nil,
			authzType:         types.AuthorizationType_AUTHORIZATION_TYPE_DELEGATE,
			amount:           sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(1000))),
			expectError:      false,
		},
		{
			name:              "valid authorization with deny list",
			allowedValidators: nil,
			deniedValidators:  []string{"val1", "val2"},
			authzType:         types.AuthorizationType_AUTHORIZATION_TYPE_UNDELEGATE,
			amount:           sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(1000))),
			expectError:      false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			authorization, err := types.NewStakeAuthorization(
				tc.allowedValidators,
				tc.deniedValidators,
				tc.authzType,
				tc.amount,
			)

			if tc.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, authorization)

			if tc.allowedValidators != nil {
				require.NotNil(t, authorization.GetAllowList())
				require.Equal(t, tc.allowedValidators, authorization.GetAllowList().Address)
			}

			if tc.deniedValidators != nil {
				require.NotNil(t, authorization.GetDenyList())
				require.Equal(t, tc.deniedValidators, authorization.GetDenyList().Address)
			}

			require.Equal(t, tc.authzType, authorization.AuthorizationType)
			if tc.amount != nil {
				require.Equal(t, tc.amount, authorization.MaxTokens)
			}
		})
	}
}

func TestStakeAuthorization_ValidateBasic(t *testing.T) {
	tests := []struct {
		name        string
		auth        types.StakeAuthorization
		expectError bool
	}{
		{
			name: "valid authorization",
			auth: types.StakeAuthorization{
				MaxTokens:         sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(1000))),
				AuthorizationType: types.AuthorizationType_AUTHORIZATION_TYPE_DELEGATE,
			},
			expectError: false,
		},
		{
			name: "invalid - negative coins",
			auth: types.StakeAuthorization{
				MaxTokens:         sdk.Coins{sdk.Coin{Denom: "stake", Amount: sdk.NewInt(-1000)}},
				AuthorizationType: types.AuthorizationType_AUTHORIZATION_TYPE_DELEGATE,
			},
			expectError: true,
		},
		{
			name: "invalid - unspecified authorization type",
			auth: types.StakeAuthorization{
				MaxTokens:         sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(1000))),
				AuthorizationType: types.AuthorizationType_AUTHORIZATION_TYPE_UNSPECIFIED,
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.auth.ValidateBasic()
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestStakeAuthorization_Accept(t *testing.T) {
	ctx := sdk.Context{}.WithGasMeter(sdk.NewInfiniteGasMeter())
	
	tests := []struct {
		name        string
		auth        types.StakeAuthorization
		msg         sdk.Msg
		expectError bool
	}{
		{
			name: "valid delegate with allow list",
			auth: types.StakeAuthorization{
				Validators: &types.StakeAuthorization_AllowList{
					AllowList: &types.StakeAuthorization_Validators{
						Address: []string{"val1"},
					},
				},
				MaxTokens:         sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(1000))),
				AuthorizationType: types.AuthorizationType_AUTHORIZATION_TYPE_DELEGATE,
			},
			msg: &types.MsgDelegate{
				DelegatorAddress: "delegator",
				ValidatorAddress: "val1",
				Amount:          sdk.NewCoin("stake", sdk.NewInt(500)),
			},
			expectError: false,
		},
		{
			name: "invalid - validator not in allow list",
			auth: types.StakeAuthorization{
				Validators: &types.StakeAuthorization_AllowList{
					AllowList: &types.StakeAuthorization_Validators{
						Address: []string{"val1"},
					},
				},
				MaxTokens:         sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(1000))),
				AuthorizationType: types.AuthorizationType_AUTHORIZATION_TYPE_DELEGATE,
			},
			msg: &types.MsgDelegate{
				DelegatorAddress: "delegator",
				ValidatorAddress: "val2",
				Amount:          sdk.NewCoin("stake", sdk.NewInt(500)),
			},
			expectError: true,
		},
		{
			name: "invalid - validator in deny list",
			auth: types.StakeAuthorization{
				Validators: &types.StakeAuthorization_DenyList{
					DenyList: &types.StakeAuthorization_Validators{
						Address: []string{"val1"},
					},
				},
				MaxTokens:         sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(1000))),
				AuthorizationType: types.AuthorizationType_AUTHORIZATION_TYPE_DELEGATE,
			},
			msg: &types.MsgDelegate{
				DelegatorAddress: "delegator",
				ValidatorAddress: "val1",
				Amount:          sdk.NewCoin("stake", sdk.NewInt(500)),
			},
			expectError: true,
		},
		{
			name: "invalid - exceeds max tokens",
			auth: types.StakeAuthorization{
				Validators: &types.StakeAuthorization_AllowList{
					AllowList: &types.StakeAuthorization_Validators{
						Address: []string{"val1"},
					},
				},
				MaxTokens:         sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(400))),
				AuthorizationType: types.AuthorizationType_AUTHORIZATION_TYPE_DELEGATE,
			},
			msg: &types.MsgDelegate{
				DelegatorAddress: "delegator",
				ValidatorAddress: "val1",
				Amount:          sdk.NewCoin("stake", sdk.NewInt(500)),
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := tc.auth.Accept(context.Background(), tc.msg)
			if tc.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.True(t, resp.Accept)
			
			// Check if authorization should be deleted (when max tokens are used up)
			if tc.auth.MaxTokens != nil {
				msgCoins := sdk.NewCoins(tc.msg.(*types.MsgDelegate).Amount)
				if tc.auth.MaxTokens.IsEqual(msgCoins) {
					require.True(t, resp.Delete)
				}
			}
		})
	}
}

func TestStakeAuthorization_MsgTypeURL(t *testing.T) {
	tests := []struct {
		name        string
		authzType   types.AuthorizationType
		expectError bool
	}{
		{
			name:        "delegate authorization",
			authzType:   types.AuthorizationType_AUTHORIZATION_TYPE_DELEGATE,
			expectError: false,
		},
		{
			name:        "undelegate authorization",
			authzType:   types.AuthorizationType_AUTHORIZATION_TYPE_UNDELEGATE,
			expectError: false,
		},
		{
			name:        "redelegate authorization",
			authzType:   types.AuthorizationType_AUTHORIZATION_TYPE_REDELEGATE,
			expectError: false,
		},
		{
			name:        "cancel unbonding delegation authorization",
			authzType:   types.AuthorizationType_AUTHORIZATION_TYPE_CANCEL_UNBONDING_DELEGATION,
			expectError: false,
		},
		{
			name:        "unspecified authorization type",
			authzType:   types.AuthorizationType_AUTHORIZATION_TYPE_UNSPECIFIED,
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			auth := types.StakeAuthorization{
				AuthorizationType: tc.authzType,
			}

			if tc.expectError {
				require.Panics(t, func() { auth.MsgTypeURL() })
				return
			}

			url := auth.MsgTypeURL()
			require.NotEmpty(t, url)
		})
	}
}
