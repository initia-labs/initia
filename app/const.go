package app

import (
	movetypes "github.com/initia-labs/initia/x/move/types"
)

const (
	// FeeDeductionGasAmount is a estimated gas amount of fee payment
	FeeDeductionGasAmount = 180_000

	// AccountAddressPrefix is the prefix of bech32 encoded address
	AccountAddressPrefix = "init"

	// AppName is the application name
	AppName = "initia"

	// EnvPrefix is environment variable prefix for the app
	EnvPrefix = "INITIA"

	// CoinType is the Cosmos Chain's coin type as defined in SLIP44 (https://github.com/satoshilabs/slips/blob/master/slip-0044.md)
	CoinType = 118

	// BondDenom staking denom for genesis boot
	BondDenom = movetypes.DefaultBaseDenom

	authzMsgExec                         = "/cosmos.authz.v1beta1.MsgExec"
	authzMsgGrant                        = "/cosmos.authz.v1beta1.MsgGrant"
	authzMsgRevoke                       = "/cosmos.authz.v1beta1.MsgRevoke"
	bankMsgSend                          = "/cosmos.bank.v1beta1.MsgSend"
	bankMsgMultiSend                     = "/cosmos.bank.v1beta1.MsgMultiSend"
	distrMsgSetWithdrawAddr              = "/cosmos.distribution.v1beta1.MsgSetWithdrawAddress"
	distrMsgWithdrawValidatorCommission  = "/cosmos.distribution.v1beta1.MsgWithdrawValidatorCommission"
	distrMsgFundCommunityPool            = "/cosmos.distribution.v1beta1.MsgFundCommunityPool"
	distrMsgWithdrawDelegatorReward      = "/cosmos.distribution.v1beta1.MsgWithdrawDelegatorReward"
	feegrantMsgGrantAllowance            = "/cosmos.feegrant.v1beta1.MsgGrantAllowance"
	feegrantMsgRevokeAllowance           = "/cosmos.feegrant.v1beta1.MsgRevokeAllowance"
	govMsgSubmitProposal                 = "/cosmos.gov.v1.MsgSubmitProposal"
	govMsgDeposit                        = "/cosmos.gov.v1.MsgDeposit"
	govMsgVote                           = "/cosmos.gov.v1.MsgVote"
	govMsgVoteWeighted                   = "/cosmos.gov.v1.MsgVoteWeighted"
	groupCreateGroup                     = "/cosmos.group.v1.MsgCreateGroup"
	groupUpdateGroupMember               = "/cosmos.group.v1.MsgUpdateGroupMember"
	groupUpdateGroupAdmin                = "/cosmos.group.v1.MsgUpdateGroupAdmin"
	groupUpdateGroupMetadata             = "/cosmos.group.v1.MsgUpdateGroupMetadata"
	groupCreateGroupPolicy               = "/cosmos.group.v1.MsgCreateGroupPolicy"
	groupUpdateGroupPolicyAdmin          = "/cosmos.group.v1.MsgUpdateGroupPolicyAdmin"
	groupUpdateGroupPolicyDecisionPolicy = "/cosmos.group.v1.MsgUpdateGroupPolicyDecisionPolicy"
	groupSubmitProposal                  = "/cosmos.group.v1.MsgSubmitProposal"
	groupWithdrawProposal                = "/cosmos.group.v1.MsgWithdrawProposal"
	groupVote                            = "/cosmos.group.v1.MsgVote"
	groupExec                            = "/cosmos.group.v1.MsgExec"
	groupLeaveGroup                      = "/cosmos.group.v1.MsgLeaveGroup"
	transferMsgTransfer                  = "/ibc.applications.transfer.v1.MsgTransfer"
	nftTransferMsgTransfer               = "/ibc.applications.nft_transfer.v1.MsgNftTransfer"
	sftTransferMsgTransfer               = "/ibc.applications.sft_transfer.v1.MsgSftTransfer"
	stakingMsgEditValidator              = "/initia.mstaking.v1.MsgEditValidator"
	stakingMsgDelegate                   = "/initia.mstaking.v1.MsgDelegate"
	stakingMsgUndelegate                 = "/initia.mstaking.v1.MsgUndelegate"
	stakingMsgBeginRedelegate            = "/initia.mstaking.v1.MsgBeginRedelegate"
	stakingMsgCreateValidator            = "/initia.mstaking.v1.MsgCreateValidator"
	moveMsgPublishModuleBundle           = "/initia.move.v1.MsgPublish"
	moveMsgExecuteEntryFunction          = "/initia.move.v1.MsgExecute"
	moveMsgExecuteScript                 = "/initia.move.v1.MsgScript"

	// UpgradeName gov proposal name
	UpgradeName = "0.0.0"
)
