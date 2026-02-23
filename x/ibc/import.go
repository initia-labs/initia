package ibc

import (
	_ "github.com/initia-labs/OPinit/api/cosmos/ics23/v1"
	_ "github.com/initia-labs/OPinit/api/ibc/applications/fee/v1"
	_ "github.com/initia-labs/OPinit/api/ibc/applications/interchain_accounts/controller/v1"
	_ "github.com/initia-labs/OPinit/api/ibc/applications/interchain_accounts/genesis/v1"
	_ "github.com/initia-labs/OPinit/api/ibc/applications/interchain_accounts/host/v1"
	_ "github.com/initia-labs/OPinit/api/ibc/applications/interchain_accounts/v1"
	_ "github.com/initia-labs/OPinit/api/ibc/applications/transfer/v1"
	_ "github.com/initia-labs/OPinit/api/ibc/core/channel/v1"
	_ "github.com/initia-labs/OPinit/api/ibc/core/client/v1"
	_ "github.com/initia-labs/OPinit/api/ibc/core/commitment/v1"
	_ "github.com/initia-labs/OPinit/api/ibc/core/connection/v1"
	_ "github.com/initia-labs/OPinit/api/ibc/core/types/v1"
	_ "github.com/initia-labs/OPinit/api/ibc/lightclients/tendermint/v1"

	_ "github.com/initia-labs/initia/api/ibc/applications/nft_transfer/v1"
	_ "github.com/initia-labs/initia/api/ibc/applications/perm/v1"
)
