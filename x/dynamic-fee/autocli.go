package dynamicfee

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	dynamicfeev1 "github.com/initia-labs/initia/api/initia/dynamicfee/v1"
)

func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: dynamicfeev1.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "Params",
					Use:       "params",
					Short:     "Query the current dynamic fee parameters",
				},
			},
		},
	}
}
