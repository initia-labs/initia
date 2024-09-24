package v1

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"
	corestoretypes "cosmossdk.io/core/store"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/x/gov/types"
	customtypes "github.com/initia-labs/initia/x/gov/types"
)

const (
	ModuleName = "gov"
)

func ConvertLegacyProposalToProposal(proposal customtypes.LegacyProposal) customtypes.Proposal {
	return customtypes.Proposal{
		Id:                     proposal.Id,
		Messages:               proposal.Messages,
		EmergencyStartTime:     proposal.EmergencyStartTime,
		EmergencyNextTallyTime: proposal.EmergencyNextTallyTime,
		Metadata:               proposal.Metadata,
		Title:                  proposal.Title,
		Summary:                proposal.Summary,
		Proposer:               proposal.Proposer,
		Expedited:              proposal.Expedited,
		Emergency:              proposal.Emergency,
		FailedReason:           proposal.FailedReason,
		Status:                 proposal.Status,
		SubmitTime:             proposal.SubmitTime,
		DepositEndTime:         proposal.DepositEndTime,
		TotalDeposit:           proposal.TotalDeposit,
		VotingStartTime:        proposal.VotingStartTime,
		VotingEndTime:          proposal.VotingEndTime,

		// Convert the final tally result
		FinalTallyResult: customtypes.TallyResult{
			TallyHeight:       0,
			TotalStakingPower: "0",
			TotalVestingPower: "0",
			V1TallyResult:     proposal.FinalTallyResult,
		},
	}
}

func MigrateStore(
	ctx context.Context,
	proposals collections.Map[uint64, customtypes.Proposal],
	storeService corestoretypes.KVStoreService, cdc codec.BinaryCodec,
) error {
	sb := collections.NewSchemaBuilder(storeService)
	legacyProposals := collections.NewMap(sb, types.ProposalsKeyPrefix, "proposals", collections.Uint64Key, codec.CollValue[customtypes.LegacyProposal](cdc))
	_, err := sb.Build()
	if err != nil {
		return err
	}

	fmt.Println("SIBONG")
	return legacyProposals.Walk(ctx, nil, func(pid uint64, lp customtypes.LegacyProposal) (bool, error) {
		p := ConvertLegacyProposalToProposal(lp)
		return false, proposals.Set(ctx, pid, p)
	})
}
