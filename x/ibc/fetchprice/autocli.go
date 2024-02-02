package fetchprice

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	fetchpricev1 "github.com/initia-labs/initia/api/ibc/applications/fetchprice/v1"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: fetchpricev1.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "Params",
					Use:       "params",
					Short:     "Query the parameters of the fetchprice process",
					Long:      "Query the parameters of the fetchprice process",
				},
			},
		},
	}
}
