package perm

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	permv1 "github.com/initia-labs/initia/api/ibc/applications/perm/v1"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: permv1.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "PermissionedRelayer",
					Use:       "permissioned-relayer",
					Alias:     []string{"relayer"},
					Short:     "Query the permissioned relayer of the IBC connection",
					Long:      "Query the permissioned relayer of the IBC connection",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "port_id"},
						{ProtoField: "channel_id"},
					},
				},
				{
					RpcMethod: "PermissionedRelayers",
					Use:       "permissioned-relayers",
					Alias:     []string{"relayers"},
					Short:     "Query the permissioned relayers",
					Long:      "Query the permissioned relayers",
				},
			},
		},
	}
}
