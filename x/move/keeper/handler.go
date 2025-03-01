package keeper

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"unsafe"

	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/cosmos/gogoproto/proto"
	"github.com/initia-labs/initia/x/move/ante"
	"github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/movevm/types"
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

////////////////////////////////////////
// Publish Functions

func (k Keeper) PublishModuleBundle(
	ctx context.Context,
	sender vmtypes.AccountAddress,
	moduleBundle vmtypes.ModuleBundle,
	upgradePolicy types.UpgradePolicy,
) error {
	moduleCodes := make([]string, len(moduleBundle.Codes))
	for i, moduleCode := range moduleBundle.Codes {
		// bytes -> hex string
		moduleCodes[i] = hex.EncodeToString(moduleCode.Code[:])
	}

	moduleCodesBz, err := json.Marshal(&moduleCodes)
	if err != nil {
		return err
	}

	upgradePolicyBz, err := json.Marshal(upgradePolicy.ToVmUpgradePolicy())
	if err != nil {
		return err
	}

	err = k.ExecuteEntryFunctionJSON(
		ctx,
		sender,
		vmtypes.StdAddress,
		types.MoveModuleNameCode,
		types.FunctionNameCodePublishV2,
		[]vmtypes.TypeTag{},
		[]string{
			// use unsafe method for fast conversion
			unsafe.String(unsafe.SliceData(moduleCodesBz), len(moduleCodesBz)),
			unsafe.String(unsafe.SliceData(upgradePolicyBz), len(upgradePolicyBz)),
		},
	)
	if err != nil {
		return err
	}

	return nil
}

////////////////////////////////////////
// Execute Functions

// Deprecated: use ExecuteEntryFunctionJSON instead
func (k Keeper) ExecuteEntryFunction(
	ctx context.Context,
	sender vmtypes.AccountAddress,
	moduleAddr vmtypes.AccountAddress,
	moduleName string,
	functionName string,
	typeArgs []vmtypes.TypeTag,
	args [][]byte,
) error {
	return k.executeEntryFunction(
		ctx,
		[]vmtypes.AccountAddress{sender},
		moduleAddr,
		moduleName,
		functionName,
		typeArgs,
		args,
		false,
	)
}

func (k Keeper) ExecuteEntryFunctionJSON(
	ctx context.Context,
	sender vmtypes.AccountAddress,
	moduleAddr vmtypes.AccountAddress,
	moduleName string,
	functionName string,
	typeArgs []vmtypes.TypeTag,
	jsonArgs []string,
) error {
	args := make([][]byte, len(jsonArgs))
	for i, jsonArg := range jsonArgs {
		// use unsafe method for fast conversion
		args[i] = unsafe.Slice(unsafe.StringData(jsonArg), len(jsonArg))
	}

	return k.executeEntryFunction(
		ctx,
		[]vmtypes.AccountAddress{sender},
		moduleAddr,
		moduleName,
		functionName,
		typeArgs,
		args,
		true,
	)
}

func (k Keeper) executeEntryFunction(
	ctx context.Context,
	senders []vmtypes.AccountAddress,
	moduleAddr vmtypes.AccountAddress,
	moduleName string,
	functionName string,
	typeArgs []vmtypes.TypeTag,
	args [][]byte,
	isJSON bool,
) error {
	payload, err := types.BuildExecuteEntryFunctionPayload(
		moduleAddr,
		moduleName,
		functionName,
		typeArgs,
		args,
		isJSON,
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
	gasForRuntime := k.computeGasForRuntime(ctx, gasMeter)

	// delegate gas metering to move vm
	sdkCtx = sdkCtx.WithGasMeter(storetypes.NewInfiniteGasMeter())

	gasBalance := gasForRuntime
	execRes, err := k.initiaMoveVM.ExecuteEntryFunction(
		&gasBalance,
		types.NewVMStore(sdkCtx, k.VMStore),
		NewApi(k, sdkCtx),
		types.NewEnv(sdkCtx, ac, ec),
		senders,
		payload,
	)

	// consume gas first and check error
	gasUsed := gasForRuntime - gasBalance
	gasMeter.ConsumeGas(gasUsed, "move runtime")
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

////////////////////////////////////////
// Script Functions

// Deprecated: use ExecuteScriptJSON instead
func (k Keeper) ExecuteScript(
	ctx context.Context,
	sender vmtypes.AccountAddress,
	byteCodes []byte,
	typeArgs []vmtypes.TypeTag,
	args [][]byte,
) error {
	return k.executeScript(
		ctx,
		[]vmtypes.AccountAddress{sender},
		byteCodes,
		typeArgs,
		args,
		false,
	)
}

func (k Keeper) ExecuteScriptJSON(
	ctx context.Context,
	sender vmtypes.AccountAddress,
	byteCodes []byte,
	typeArgs []vmtypes.TypeTag,
	jsonArgs []string,
) error {
	args := make([][]byte, len(jsonArgs))
	for i, jsonArg := range jsonArgs {
		// use unsafe method for fast conversion
		args[i] = unsafe.Slice(unsafe.StringData(jsonArg), len(jsonArg))
	}

	return k.executeScript(
		ctx,
		[]vmtypes.AccountAddress{sender},
		byteCodes,
		typeArgs,
		args,
		true,
	)
}

func (k Keeper) executeScript(
	ctx context.Context,
	senders []vmtypes.AccountAddress,
	byteCodes []byte,
	typeArgs []vmtypes.TypeTag,
	args [][]byte,
	isJSON bool,
) error {
	// prepare payload
	payload, err := types.BuildExecuteScriptPayload(
		byteCodes,
		typeArgs,
		args,
		isJSON,
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
	gasForRuntime := k.computeGasForRuntime(ctx, gasMeter)

	// delegate gas metering to move vm
	sdkCtx = sdkCtx.WithGasMeter(storetypes.NewInfiniteGasMeter())

	gasBalance := gasForRuntime
	execRes, err := k.initiaMoveVM.ExecuteScript(
		&gasBalance,
		types.NewVMStore(sdkCtx, k.VMStore),
		NewApi(k, sdkCtx),
		types.NewEnv(sdkCtx, ac, ec),
		senders,
		payload,
	)

	// consume gas first and check error
	gasUsed := gasForRuntime - gasBalance
	gasMeter.ConsumeGas(gasUsed, "move runtime")
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

////////////////////////////////////////
// Response Handler

func (k Keeper) handleExecuteResponse(
	ctx sdk.Context,
	gasMeter storetypes.GasMeter,
	execRes vmtypes.ExecutionResult,
) error {
	// Emit contract events
	for _, event := range execRes.Events {
		typeTag := event.TypeTag
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

		// increase global account number if the given account is not exists
		if !k.authKeeper.HasAccount(ctx, addr) {
			k.authKeeper.NextAccountNumber(ctx)
		}

		// write or overwrite account
		k.authKeeper.SetAccount(ctx, accI)
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
		err := k.dispatchMessage(sdkCtx, message)
		if err != nil {
			return err
		}
	}

	return nil
}

func (k Keeper) dispatchMessage(parentCtx sdk.Context, message vmtypes.CosmosMessage) (err error) {
	ctx, commit := parentCtx.CacheContext()
	defer func() {
		if r := recover(); r != nil {
			switch r.(type) {
			case storetypes.ErrorOutOfGas:
				// propagate out of gas error
				panic(r)
			default:
				err = fmt.Errorf("panic: %v", r)
			}
		}

		success := err == nil

		// create submsg event
		event := sdk.NewEvent(
			types.EventTypeSubmsg,
			sdk.NewAttribute(types.AttributeKeySuccess, fmt.Sprintf("%v", success)),
		)

		if !success {
			// return error if failed and not allowed to fail
			if !message.AllowFailure {
				return
			}

			// emit failed reason event if failed and allowed to fail
			event = event.AppendAttributes(sdk.NewAttribute(types.AttributeKeyReason, err.Error()))
		} else {
			// commit if success
			commit()
		}

		// reset error because it's allowed to fail
		err = nil

		// emit submessage event
		parentCtx.EventManager().EmitEvent(event)

		// if callback exists, execute it with parent context because it's already committed
		if message.Callback != nil {
			err = k.ExecuteEntryFunctionJSON(
				parentCtx,
				message.Sender,
				message.Callback.ModuleAddress,
				message.Callback.ModuleName,
				message.Callback.FunctionName,
				[]vmtypes.TypeTag{},
				[]string{
					fmt.Sprintf("\"%d\"", message.Callback.Id),
					fmt.Sprintf("%v", success),
				},
			)
		}
	}()

	// validate basic & signer check is done in HandleVMStargateMsg
	var msg proto.Message
	msg, err = k.HandleVMStargateMsg(ctx, &message)
	if err != nil {
		return
	}

	// find the handler
	handler := k.msgRouter.Handler(msg)
	if handler == nil {
		err = types.ErrNotSupportedCosmosMessage
		return
	}

	//  and execute it
	res, err := handler(ctx, msg)
	if err != nil {
		return
	}

	// emit events
	ctx.EventManager().EmitEvents(res.GetEvents())

	return
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

////////////////////////////////////////
// View Functions

// Deprecated: use ExecuteViewFunctionJSON instead
func (k Keeper) ExecuteViewFunction(
	ctx context.Context,
	moduleAddr vmtypes.AccountAddress,
	moduleName string,
	functionName string,
	typeArgs []vmtypes.TypeTag,
	args [][]byte,
) (vmtypes.ViewOutput, uint64, error) {
	return k.executeViewFunction(
		ctx,
		moduleAddr,
		moduleName,
		functionName,
		typeArgs,
		args,
		false,
	)
}

func (k Keeper) ExecuteViewFunctionJSON(
	ctx context.Context,
	moduleAddr vmtypes.AccountAddress,
	moduleName string,
	functionName string,
	typeArgs []vmtypes.TypeTag,
	jsonArgs []string,
) (vmtypes.ViewOutput, uint64, error) {
	args := make([][]byte, len(jsonArgs))
	for i, jsonArg := range jsonArgs {
		// use unsafe method for fast conversion
		args[i] = unsafe.Slice(unsafe.StringData(jsonArg), len(jsonArg))
	}

	return k.executeViewFunction(
		ctx,
		moduleAddr,
		moduleName,
		functionName,
		typeArgs,
		args,
		true,
	)
}

func (k Keeper) executeViewFunction(
	ctx context.Context,
	moduleAddr vmtypes.AccountAddress,
	moduleName string,
	functionName string,
	typeArgs []vmtypes.TypeTag,
	args [][]byte,
	isJSON bool,
) (vmtypes.ViewOutput, uint64, error) {
	payload, err := types.BuildExecuteViewFunctionPayload(
		moduleAddr,
		moduleName,
		functionName,
		typeArgs,
		args,
		isJSON,
	)
	if err != nil {
		return vmtypes.ViewOutput{}, 0, err
	}

	ac := types.NextAccountNumber(ctx, k.authKeeper)
	ec, err := k.ExecutionCounter.Next(ctx)
	if err != nil {
		return vmtypes.ViewOutput{}, 0, err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	gasMeter := sdkCtx.GasMeter()
	gasForRuntime := k.computeGasForRuntime(ctx, gasMeter)

	// delegate gas metering to move vm
	sdkCtx = sdkCtx.WithGasMeter(storetypes.NewInfiniteGasMeter())

	gasBalance := gasForRuntime
	viewRes, err := k.initiaMoveVM.ExecuteViewFunction(
		&gasBalance,
		types.NewVMStore(sdkCtx, k.VMStore),
		NewApi(k, sdkCtx),
		types.NewEnv(sdkCtx, ac, ec),
		payload,
	)

	// consume gas first and check error
	gasUsed := gasForRuntime - gasBalance
	gasMeter.ConsumeGas(gasUsed, "view; move runtime")
	if err != nil {
		return vmtypes.ViewOutput{}, gasUsed, err
	}

	return viewRes, gasUsed, nil
}

func (k Keeper) computeGasForRuntime(ctx context.Context, gasMeter storetypes.GasMeter) uint64 {
	gasForRuntime := gasMeter.Limit() - gasMeter.GasConsumedToLimit()
	if isSimulation(ctx) {
		gasForRuntime = k.config.ContractSimulationGasLimit
	}

	// gasUnitScale is multiplied in moveVM to scale the gas limit, so we need to divide it here
	// if gasForRuntime is too large, it will overflow when multiplied in moveVM.
	const gasUintScale = 100
	if gasForRuntime > math.MaxUint64/gasUintScale {
		return math.MaxUint64 / gasUintScale
	}

	return gasForRuntime
}
