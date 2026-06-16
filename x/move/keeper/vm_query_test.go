package keeper_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	storetypes "cosmossdk.io/store/types"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	govtypes "github.com/initia-labs/initia/x/gov/types"
	"github.com/initia-labs/initia/x/move/types"

	vmtypes "github.com/initia-labs/movevm/types"
)

func Test_VMQuery_Stargate(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	// not existing
	_, err := input.MoveKeeper.HandleVMQuery(ctx, &vmtypes.QueryRequest{
		Stargate: &vmtypes.StargateQuery{
			Path: "/initia.gov.v1.Query/Proposals",
			Data: nil,
		},
	})
	require.ErrorContains(t, err, types.ErrNotSupportedStargateQuery.Error())

	proposal := govtypes.Proposal{
		Id:      1,
		Title:   "title",
		Summary: "summary",
		Status:  govtypesv1.ProposalStatus_PROPOSAL_STATUS_DEPOSIT_PERIOD,
	}

	// set Proposal
	err = input.GovKeeper.SetProposal(ctx, proposal)
	require.NoError(t, err)

	// query proposal
	resBz, err := input.MoveKeeper.HandleVMQuery(ctx, &vmtypes.QueryRequest{
		Stargate: &vmtypes.StargateQuery{
			Path: "/initia.gov.v1.Query/Proposal",
			Data: []byte(`{"proposal_id": "1"}`),
		},
	})
	require.NoError(t, err)

	// expected proposal res json bytes
	expectedResBz, err := input.EncodingConfig.Codec.MarshalJSON(&govtypes.QueryProposalResponse{
		Proposal: &proposal,
	})
	require.NoError(t, err)
	require.Equal(t, expectedResBz, resBz)
}

func Test_VMQuery_Custom(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	// not existing
	_, err := input.MoveKeeper.HandleVMQuery(ctx, &vmtypes.QueryRequest{
		Custom: &vmtypes.CustomQuery{
			Name: "aaa",
			Data: nil,
		},
	})
	require.ErrorContains(t, err, types.ErrNotSupportedCustomQuery.Error())

	reqBz, err := json.Marshal(types.ToSDKAddressRequest{
		VMAddr: vmtypes.StdAddress.String(),
	})
	require.NoError(t, err)

	resBz, err := input.MoveKeeper.HandleVMQuery(ctx, &vmtypes.QueryRequest{
		Custom: &vmtypes.CustomQuery{
			Name: "to_sdk_address",
			Data: reqBz,
		},
	})
	require.NoError(t, err)

	var res types.ToSDKAddressResponse
	err = json.Unmarshal(resBz, &res)
	require.NoError(t, err)
	require.Equal(t, types.StdAddr[12:].String(), res.SDKAddr)
}

func TestQueryStargateDataRace(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	k := input.MoveKeeper

	// Seed distinct proposals id=1..N so a wrong id is unambiguous.
	const N = 16
	submitTime := time.Unix(1, 0).UTC()
	for i := uint64(1); i <= N; i++ {
		require.NoError(t, input.GovKeeper.SetProposal(ctx, govtypes.Proposal{
			Id:         i,
			Title:      fmt.Sprintf("proposal-%d", i),
			Status:     govtypesv1.ProposalStatus_PROPOSAL_STATUS_DEPOSIT_PERIOD,
			SubmitTime: &submitTime,
			Emergency:  true,
		}))
	}

	const workers = 128
	var wg sync.WaitGroup
	var mismatches int64
	var errs int64
	mu := sync.Mutex{}
	samples := []string{}

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					atomic.AddInt64(&errs, 1)
				}
			}()
			cctx, _ := sdk.UnwrapSDKContext(ctx).CacheContext()
			cctx = cctx.WithGasMeter(storetypes.NewInfiniteGasMeter())

			reqID := uint64((w % N) + 1)
			res, err := k.HandleVMQuery(cctx, &vmtypes.QueryRequest{
				Stargate: &vmtypes.StargateQuery{
					Path: "/initia.gov.v1.Query/Proposal",
					Data: []byte(fmt.Sprintf(`{"proposal_id":"%d"}`, reqID)),
				},
			})
			if err != nil {
				atomic.AddInt64(&errs, 1)
				return
			}
			// Parse returned proposal id and compare with requested.
			var resp struct {
				Proposal struct {
					Id string `json:"id"`
				} `json:"proposal"`
			}
			if json.Unmarshal(res, &resp) == nil {
				got := resp.Proposal.Id
				want := fmt.Sprintf("%d", reqID)
				if got != "" && got != want {
					atomic.AddInt64(&mismatches, 1)
					mu.Lock()
					if len(samples) < 20 {
						samples = append(samples, fmt.Sprintf("asked=%s got=%s", want, got))
					}
					mu.Unlock()
				}
			}
		}(w)
	}
	wg.Wait()

	t.Logf("workers=%d mismatches=%d errors=%d", workers, mismatches, errs)
	for _, s := range samples {
		t.Logf("CORRUPTION: %s", s)
	}
	require.Zero(t, errs)
	require.Zero(t, mismatches)
}

func TestQueryStargateVMSessionRace(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	k := input.MoveKeeper

	const N = 16
	submitTime := time.Unix(1, 0).UTC()
	for i := uint64(1); i <= N; i++ {
		require.NoError(t, input.GovKeeper.SetProposal(ctx, govtypes.Proposal{
			Id:         i,
			Title:      fmt.Sprintf("proposal-%d", i),
			Summary:    fmt.Sprintf("summary-%d", i),
			Status:     govtypesv1.ProposalStatus_PROPOSAL_STATUS_DEPOSIT_PERIOD,
			SubmitTime: &submitTime,
			Emergency:  true,
		}))
	}

	const workers = 128
	var wg sync.WaitGroup
	var mismatches int64
	var errs int64
	var ok int64
	mu := sync.Mutex{}
	samples := []string{}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					atomic.AddInt64(&errs, 1)
				}
			}()
			reqID := uint64((w % N) + 1)
			// own cache context -> only shared mutable state is the keeper proto buffer
			cctx, _ := sdkCtx.CacheContext()
			cctx = cctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
			out, _, err := k.ExecuteViewFunctionJSON(
				cctx,
				vmtypes.StdAddress,
				"query",
				"get_proposal",
				[]vmtypes.TypeTag{},
				[]string{fmt.Sprintf(`"%d"`, reqID)},
			)
			if err != nil {
				atomic.AddInt64(&errs, 1)
				mu.Lock()
				if len(samples) < 3 {
					samples = append(samples, "ERR: "+err.Error())
				}
				mu.Unlock()
				return
			}
			// out.Ret is a JSON array: [id, title, summary, status]
			want := fmt.Sprintf(`"%d"`, reqID)
			wantTitle := fmt.Sprintf("proposal-%d", reqID)
			ret := out.Ret
			// detect corruption: returned title belongs to a different proposal
			if !strings.Contains(ret, wantTitle) {
				atomic.AddInt64(&mismatches, 1)
				mu.Lock()
				if len(samples) < 20 {
					trimmed := ret
					if len(trimmed) > 90 {
						trimmed = trimmed[:90]
					}
					samples = append(samples, fmt.Sprintf("asked id=%s want title=%s | got=%s", want, wantTitle, trimmed))
				}
				mu.Unlock()
			} else {
				atomic.AddInt64(&ok, 1)
			}
		}(w)
	}
	wg.Wait()

	t.Logf("workers=%d ok=%d mismatches=%d errors=%d", workers, ok, mismatches, errs)
	for _, s := range samples {
		t.Logf("VM-SESSION CORRUPTION: %s", s)
	}
	require.EqualValues(t, workers, ok)
	require.Zero(t, errs)
	require.Zero(t, mismatches)
}
