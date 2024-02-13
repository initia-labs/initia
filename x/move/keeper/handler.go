package keeper

import (
	"context"
	"errors"
	"math"
	"strings"

	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/cosmos/gogoproto/proto"
	"github.com/initia-labs/initia/x/move/ante"
	"github.com/initia-labs/initia/x/move/types"
	vmapi "github.com/initia-labs/initiavm/api"
	vmtypes "github.com/initia-labs/initiavm/types"
)

func isSimulation(
	ctx context.Context,
) bool {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// only executed when ExecMode is
	// * simulate
	// * finalize
	return sdkCtx.ExecMode() == sdk.ExecModeSimulate
}

// extract module address and module name from the compiled module bytes
func (k Keeper) extractModuleIdentifier(moduleBundle vmtypes.ModuleBundle) ([]string, error) {
	modules := make([]string, len(moduleBundle.Codes))
	for i, moduleBz := range moduleBundle.Codes {
		moduleAddr, moduleName, err := vmapi.ReadModuleInfo(moduleBz.Code)
		if err != nil {
			return nil, err
		}

		modules[i] = vmtypes.NewModuleId(moduleAddr, moduleName).String()
	}

	return modules, nil
}

func (k Keeper) PublishModuleBundle(
	ctx context.Context,
	sender vmtypes.AccountAddress,
	moduleBundle vmtypes.ModuleBundle,
	upgradePolicy types.UpgradePolicy,
) error {

	// build execute args
	moduleIds, err := k.extractModuleIdentifier(moduleBundle)
	if err != nil {
		return err
	}

	moduleIdBz, err := vmtypes.SerializeStringVector(moduleIds)
	if err != nil {
		return err
	}

	moduleCodeArr := make([][]byte, len(moduleBundle.Codes))
	for i, moduleCode := range moduleBundle.Codes {
		moduleCodeArr[i] = moduleCode.Code[:]
	}

	codeBz, err := vmtypes.SerializeBytesVector(moduleCodeArr)
	if err != nil {
		return err
	}

	err = k.ExecuteEntryFunction(
		ctx,
		sender,
		vmtypes.StdAddress,
		types.MoveModuleNameCode,
		types.FunctionNameCodePublish,
		[]vmtypes.TypeTag{},
		[][]byte{
			moduleIdBz,
			codeBz,
			{upgradePolicy.ToVmUpgradePolicy()},
		},
	)
	if err != nil {
		return err
	}

	return nil
}

func (k Keeper) ExecuteEntryFunction(
	ctx context.Context,
	sender vmtypes.AccountAddress,
	moduleAddr vmtypes.AccountAddress,
	moduleName string,
	functionName string,
	typeArgs []vmtypes.TypeTag,
	args [][]byte,
) error {
	return k.ExecuteEntryFunctionWithMultiSenders(
		ctx,
		[]vmtypes.AccountAddress{sender},
		moduleAddr,
		moduleName,
		functionName,
		typeArgs,
		args,
	)
}

func (k Keeper) ExecuteEntryFunctionWithMultiSenders(
	ctx context.Context,
	senders []vmtypes.AccountAddress,
	moduleAddr vmtypes.AccountAddress,
	moduleName string,
	functionName string,
	typeArgs []vmtypes.TypeTag,
	args [][]byte,
) error {
	// prepare payload
	payload, err := types.BuildExecuteEntryFunctionPayload(
		moduleAddr,
		moduleName,
		functionName,
		typeArgs,
		args,
	)
	if err != nil {
		return err
	}

	sendersStr := make([]string, len(senders))
	for i, sender := range senders {
		sendersStr[i] = sender.String()
	}

	ac := types.NextAccountNumber(ctx, k.authKeeper)
	ec, err := k.ExecutionCounter.Next(ctx)
	if err != nil {
		return err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	gasMeter := sdkCtx.GasMeter()
	gasForRuntime := gasMeter.Limit() - gasMeter.GasConsumedToLimit()

	if isSimulation(ctx) {
		gasForRuntime = k.config.ContractSimulationGasLimit
	} else if gasMeter.Limit() == 0 {
		// infinite gas meter
		gasForRuntime = math.MaxUint64
	}

	// delegate gas metering to move vm
	sdkCtx = sdkCtx.WithGasMeter(storetypes.NewInfiniteGasMeter())

	// run vm
	execRes, err := k.moveVM.ExecuteEntryFunction(
		types.NewVMStore(sdkCtx, k.VMStore),
		NewApi(k, sdkCtx),
		types.NewEnv(sdkCtx, ac, ec),
		gasForRuntime,
		senders,
		payload,
	)

	// consume gas first and check error
	gasMeter.ConsumeGas(execRes.GasUsed, "move runtime")
	if err != nil {
		return err
	}

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeExecute,
		sdk.NewAttribute(types.AttributeKeySender, strings.Join(sendersStr, ",")),
		sdk.NewAttribute(types.AttributeKeyModuleAddr, moduleAddr.String()),
		sdk.NewAttribute(types.AttributeKeyModuleName, moduleName),
		sdk.NewAttribute(types.AttributeKeyFunctionName, functionName),
	))

	// we still need infinite gas meter for CSR, so pass new context
	return k.handleExecuteResponse(sdkCtx, gasMeter, execRes)
}

func (k Keeper) ExecuteScript(
	ctx context.Context,
	sender vmtypes.AccountAddress,
	byteCodes []byte,
	typeArgs []vmtypes.TypeTag,
	args [][]byte,
) error {
	return k.ExecuteScriptWithMultiSenders(
		ctx,
		[]vmtypes.AccountAddress{sender},
		byteCodes,
		typeArgs,
		args,
	)
}

func (k Keeper) ExecuteScriptWithMultiSenders(
	ctx context.Context,
	senders []vmtypes.AccountAddress,
	byteCodes []byte,
	typeArgs []vmtypes.TypeTag,
	args [][]byte,
) error {
	// prepare payload
	payload, err := types.BuildExecuteScriptPayload(
		byteCodes,
		typeArgs,
		args,
	)
	if err != nil {
		return err
	}

	sendersStr := make([]string, len(senders))
	for i, sender := range senders {
		sendersStr[i] = sender.String()
	}

	ac := types.NextAccountNumber(ctx, k.authKeeper)
	ec, err := k.ExecutionCounter.Next(ctx)
	if err != nil {
		return err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	gasMeter := sdkCtx.GasMeter()
	gasForRuntime := gasMeter.Limit() - gasMeter.GasConsumedToLimit()

	if isSimulation(ctx) {
		gasForRuntime = k.config.ContractSimulationGasLimit
	} else if gasMeter.Limit() == 0 {
		// infinite gas meter
		gasForRuntime = math.MaxUint64
	}

	// delegate gas metering to move vm
	sdkCtx = sdkCtx.WithGasMeter(storetypes.NewInfiniteGasMeter())

	// run vm
	execRes, err := k.moveVM.ExecuteScript(
		types.NewVMStore(sdkCtx, k.VMStore),
		NewApi(k, sdkCtx),
		types.NewEnv(sdkCtx, ac, ec),
		gasForRuntime,
		senders,
		payload,
	)

	// consume gas first and check error
	gasMeter.ConsumeGas(execRes.GasUsed, "move runtime")
	if err != nil {
		return err
	}

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeScript,
		sdk.NewAttribute(types.AttributeKeySender, strings.Join(sendersStr, ",")),
	))

	// we still need infinite gas meter for CSR, so pass new context
	return k.handleExecuteResponse(sdkCtx, gasMeter, execRes)
}

func (k Keeper) handleExecuteResponse(
	ctx sdk.Context,
	gasMeter storetypes.GasMeter,
	execRes vmtypes.ExecutionResult,
) error {
	// Emit contract events
	for _, event := range execRes.Events {
		typeTag, err := vmapi.StringifyTypeTag(event.TypeTag)
		if err != nil {
			return err
		}

		ctx.EventManager().EmitEvent(sdk.NewEvent(types.EventTypeMove,
			sdk.NewAttribute(types.AttributeKeyTypeTag, typeTag),
			sdk.NewAttribute(types.AttributeKeyData, event.EventData),
		))
	}

	// Create cosmos accounts
	for _, acc := range execRes.NewAccounts {
		addr := types.ConvertVMAddressToSDKAddress(acc.Address)

		var accI sdk.AccountI
		switch acc.AccountType {
		case vmtypes.AccountType_Base:
			accI = authtypes.NewBaseAccountWithAddress(addr)
		case vmtypes.AccountType_Object:
			accI = types.NewObjectAccountWithAddress(addr)
		case vmtypes.AccountType_Table:
			accI = types.NewTableAccountWithAddress(addr)
		default:
			return errors.New("unsupported account type")
		}
		if err := accI.SetAccountNumber(acc.AccountNumber); err != nil {
			return err
		}

		k.authKeeper.SetAccount(ctx, accI)
		k.authKeeper.NextAccountNumber(ctx) // increase global account number
	}

	// CSR: distribute fee coins to contract creator
	err := k.DistributeContractSharedRevenue(ctx, execRes.GasUsages)
	if err != nil {
		return err
	}

	// restore gas meter to original
	// NOTE: this line should be here to avoid charging any extra gas for CSR
	ctx = ctx.WithGasMeter(gasMeter)

	// apply staking delta
	if err := k.ApplyStakingDeltas(ctx, execRes.StakingDeltas); err != nil {
		return err
	}

	// dispatch returned cosmos messages
	err = k.DispatchMessages(ctx, execRes.CosmosMessages)
	if err != nil {
		return err
	}

	return nil
}

// DispatchMessages run the given cosmos messages and emit events
func (k Keeper) DispatchMessages(ctx context.Context, messages []vmtypes.CosmosMessage) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	for _, message := range messages {
		var msg proto.Message
		var err error
		if stargateMsg, ok := message.(*vmtypes.CosmosMessage__Stargate); ok {
			msg, err = k.HandleVMStargateMsg(ctx, &stargateMsg.Value)
		} else {
			msg, err = types.ConvertToSDKMessage(ctx, NewMoveBankKeeper(&k), NewNftKeeper(&k), message, k.ac, k.vc)
			if err != nil {
				return err
			}
		}

		// validate msg
		if msg, ok := msg.(sdk.HasValidateBasic); ok {
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
		}

		// find the handler
		handler := k.msgRouter.Handler(msg)
		if handler == nil {
			return types.ErrNotSupportedCosmosMessage
		}

		//  and execute it
		res, err := handler(sdkCtx, msg)
		if err != nil {
			return err
		}

		// emit events
		sdkCtx.EventManager().EmitEvents(res.GetEvents())
	}

	return nil
}

// DistributeContractSharedRevenue distribute a portion of gas fee to contract creator account
func (k Keeper) DistributeContractSharedRevenue(ctx context.Context, gasUsages []vmtypes.GasUsage) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	value := ctx.Value(ante.GasPricesContextKey)
	if value == nil {
		return nil
	}

	gasPrices := value.(sdk.DecCoins)
	revenueRatio, err := k.ContractSharedRevenueRatio(ctx)
	if err != nil {
		return err
	}

	revenueGasPrices := gasPrices.MulDec(revenueRatio)
	if revenueGasPrices.IsZero() {
		return nil
	}

	for _, gasUsage := range gasUsages {

		// ignore 0x1 gas usage
		if vmtypes.StdAddress.Equals(gasUsage.ModuleId.Address) {
			continue
		}

		revenue, _ := revenueGasPrices.MulDec(sdkmath.LegacyNewDec(int64(gasUsage.GasUsed))).TruncateDecimal()
		if revenue.IsZero() {
			continue
		}

		creatorAddr := types.ConvertVMAddressToSDKAddress(gasUsage.ModuleId.Address)
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, k.feeCollector, creatorAddr, revenue); err != nil {
			return err
		}

		sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
			types.EventTypeContractSharedRevenue,
			sdk.NewAttribute(types.AttributeKeyCreator, gasUsage.ModuleId.Address.String()),
			sdk.NewAttribute(types.AttributeKeyRevenue, revenue.String()),
		))
	}

	return nil
}
