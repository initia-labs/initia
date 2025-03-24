package types

import (
	"fmt"
	"strings"
	"time"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"

	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
)

// NewProposal creates a new Proposal instance
func NewProposal(messages []sdk.Msg, id uint64, submitTime, depositEndTime time.Time, metadata, title, summary string, proposer sdk.AccAddress, expedited bool) (Proposal, error) {
	msgs, err := sdktx.SetMsgs(messages)
	if err != nil {
		return Proposal{}, err
	}

	p := Proposal{
		Id:               id,
		Messages:         msgs,
		Metadata:         metadata,
		Status:           v1.StatusDepositPeriod,
		FinalTallyResult: EmptyTallyResult(),
		SubmitTime:       &submitTime,
		DepositEndTime:   &depositEndTime,
		Title:            title,
		Summary:          summary,
		Proposer:         proposer.String(),
		Expedited:        expedited,
	}

	return p, nil
}

// GetMessages returns the proposal messages
func (p Proposal) GetMsgs() ([]sdk.Msg, error) {
	return sdktx.GetMsgs(p.Messages, "sdk.MsgProposal")
}

// GetMinDepositFromParams returns min expedited deposit from the gov params if
// the proposal is expedited. Otherwise, returns the regular min deposit from
// gov params.
func (p Proposal) GetMinDepositFromParams(params Params) sdk.Coins {
	if p.Expedited {
		return params.ExpeditedMinDeposit
	}
	return params.MinDeposit
}

// UnpackInterfaces implements UnpackInterfacesMessage.UnpackInterfaces
func (p Proposal) UnpackInterfaces(unpacker types.AnyUnpacker) error {
	return sdktx.UnpackInterfaces(unpacker, p.Messages)
}

func (p Proposal) ToV1() v1.Proposal {
	return v1.Proposal{
		Id:               p.Id,
		Messages:         p.Messages,
		Status:           p.Status,
		FinalTallyResult: p.FinalTallyResult.V1TallyResult,
		SubmitTime:       p.SubmitTime,
		DepositEndTime:   p.DepositEndTime,
		TotalDeposit:     p.TotalDeposit,
		VotingStartTime:  p.VotingStartTime,
		VotingEndTime:    p.VotingEndTime,
		Metadata:         p.Metadata,
		Title:            p.Title,
		Summary:          p.Summary,
		Proposer:         p.Proposer,
		Expedited:        p.Expedited,
		FailedReason:     p.FailedReason,
	}
}

// Proposals is an array of proposal
type Proposals []*Proposal

var _ types.UnpackInterfacesMessage = Proposals{}

// String implements stringer interface
func (p Proposals) String() string {
	out := "ID - (Status) [Type] Title\n"
	for _, prop := range p {
		out += fmt.Sprintf("%d - %s\n",
			prop.Id, prop.Status)
	}
	return strings.TrimSpace(out)
}

// UnpackInterfaces implements UnpackInterfacesMessage.UnpackInterfaces
func (p Proposals) UnpackInterfaces(unpacker types.AnyUnpacker) error {
	for _, x := range p {
		err := x.UnpackInterfaces(unpacker)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p Proposals) ToV1() []*v1.Proposal {
	return ProposalsToV1(p)
}

func ProposalsToV1(proposals []*Proposal) []*v1.Proposal {
	v1Proposals := make([]*v1.Proposal, 0, len(proposals))
	for _, proposal := range proposals {
		v1Proposal := proposal.ToV1()
		v1Proposals = append(v1Proposals, &v1Proposal)
	}
	return v1Proposals
}

type (
	// ProposalQueue defines a queue for proposal ids
	ProposalQueue []uint64
)

// ProposalStatusFromString turns a string into a ProposalStatus
func ProposalStatusFromString(str string) (v1.ProposalStatus, error) {
	num, ok := v1.ProposalStatus_value[str]
	if !ok {
		return v1.StatusNil, fmt.Errorf("'%s' is not a valid proposal status", str)
	}
	return v1.ProposalStatus(num), nil
}

// ValidProposalStatus returns true if the proposal status is valid and false
// otherwise.
func ValidProposalStatus(status v1.ProposalStatus) bool {
	if status == v1.StatusDepositPeriod ||
		status == v1.StatusVotingPeriod ||
		status == v1.StatusPassed ||
		status == v1.StatusRejected ||
		status == v1.StatusFailed {
		return true
	}
	return false
}

// EmptyTallyResult returns an empty TallyResult
func EmptyTallyResult() TallyResult {
	v1TallyResult := v1.EmptyTallyResult()
	return TallyResult{
		TallyHeight:       0,
		TotalStakingPower: math.ZeroInt().String(),
		TotalVestingPower: math.ZeroInt().String(),
		V1TallyResult:     &v1TallyResult,
	}
}
