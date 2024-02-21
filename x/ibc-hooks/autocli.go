package ibc_hooks

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	ibchooksv1 "github.com/initia-labs/initia/api/initia/ibchooks/v1"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: ibchooksv1.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "Params",
					Use:       "params",
					Short:     "Query the parameters of the hook process",
					Long:      "Query the parameters of the hook process",
				},
				{
					RpcMethod: "ACL",
					Use:       "acl",
					Short:     "Query the ACL of the address",
					Long:      "Query the ACL of the address",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "address"},
					},
				},
				{
					RpcMethod: "ACLs",
					Use:       "acls",
					Short:     "Query the ACLs",
					Long:      "Query the ACLs",
				},
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              ibchooksv1.Msg_ServiceDesc.ServiceName,
			RpcCommandOptions:    []*autocliv1.RpcCommandOptions{},
			EnhanceCustomCommand: false,
		},
	}
}
