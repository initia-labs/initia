package types_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	"github.com/initia-labs/initia/v1/x/distribution/types"
)

func TestParams_ValidateBasic(t *testing.T) {
	toDec := math.LegacyMustNewDecFromStr

	type fields struct {
		CommunityTax        math.LegacyDec
		WithdrawAddrEnabled bool
		RewardWeights       []types.RewardWeight
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{"success", fields{toDec("0.1"), false, []types.RewardWeight{}}, false},
		{"negative community tax", fields{toDec("-0.1"), false, []types.RewardWeight{}}, true},
		{"total sum greater than 1", fields{toDec("1.1"), false, []types.RewardWeight{}}, true},
		{"valid reward weight", fields{toDec("0.1"), true, []types.RewardWeight{
			{
				Denom:  "foo",
				Weight: math.LegacyOneDec(),
			},
		}}, false},
		{"invalid reward denom", fields{toDec("0.1"), true, []types.RewardWeight{
			{
				Denom:  "foo!",
				Weight: math.LegacyOneDec(),
			},
		}}, true},
		{"invalid reward weight", fields{toDec("0.1"), true, []types.RewardWeight{
			{
				Denom:  "foo",
				Weight: math.LegacyOneDec().Neg(),
			},
		}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := types.Params{
				CommunityTax:        tt.fields.CommunityTax,
				WithdrawAddrEnabled: tt.fields.WithdrawAddrEnabled,
				RewardWeights:       tt.fields.RewardWeights,
			}
			if err := p.ValidateBasic(); (err != nil) != tt.wantErr {
				t.Errorf("ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultParams(t *testing.T) {
	require.NoError(t, types.DefaultParams().ValidateBasic())
}
