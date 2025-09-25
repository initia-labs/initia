package reward

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	rewardv1 "github.com/initia-labs/initia/api/initia/reward/v1"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service:              rewardv1.Query_ServiceDesc.ServiceName,
			RpcCommandOptions:    []*autocliv1.RpcCommandOptions{},
			EnhanceCustomCommand: true,
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              rewardv1.Msg_ServiceDesc.ServiceName,
			RpcCommandOptions:    []*autocliv1.RpcCommandOptions{},
			EnhanceCustomCommand: true,
		},
	}
}
