package perm

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	permv1 "github.com/initia-labs/initia/v1/api/ibc/applications/perm/v1"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: permv1.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "ChannelState",
					Use:       "channel-state",
					Alias:     []string{"channel"},
					Short:     "Query the permissioned relayers of the IBC connection",
					Long:      "Query the permissioned relayers of the IBC connection",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "port_id"},
						{ProtoField: "channel_id"},
					},
				},
				{
					RpcMethod: "ChannelStates",
					Use:       "channel-states",
					Alias:     []string{"channels"},
					Short:     "Query the permissioned relayers of all IBC connections",
					Long:      "Query the permissioned relayers of all IBC connections",
				},
			},
		},
	}
}
