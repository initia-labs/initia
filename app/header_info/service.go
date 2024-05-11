package header_info

import (
	"context"

	"cosmossdk.io/core/header"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// headerInfoService implements the header.Service interface
type headerInfoService struct{}

func NewHeaderInfoService() header.Service {
	return headerInfoService{}
}

func (hs headerInfoService) GetHeaderInfo(ctx context.Context) header.Info {
	return sdk.UnwrapSDKContext(ctx).HeaderInfo()
}
