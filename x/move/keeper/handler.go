package keeper

import (
	"errors"
	"math"
	"strings"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/initia-labs/initia/x/move/ante"
	"github.com/initia-labs/initia/x/move/types"
	vmapi "github.com/initia-labs/initiavm/api"
	vmtypes "github.com/initia-labs/initiavm/types"
)

func isSimulationOrCheckTx(
	ctx sdk.Context,
) bool {
	if ctx.IsCheckTx() || ctx.IsReCheckTx() {
		return true
	}

	simulate := ctx.Value(ante.SimulationFlagContextKey)
	if simulate == nil {
		return false
	}

	return simulate.(bool)
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
	ctx sdk.Context,
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
	ctx sdk.Context,
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
	ctx sdk.Context,
	senders []vmtypes.AccountAddress,
	moduleAddr vmtypes.AccountAddress,
	moduleName string,
	functionName string,
	typeArgs []vmtypes.TypeTag,
	args [][]byte,
) error {
	vm := k.moveVM
	gasMeter := ctx.GasMeter()
	gasForRuntime := gasMeter.Limit() - gasMeter.GasConsumedToLimit()

	isSimulationOrCheckTx := isSimulationOrCheckTx(ctx)
	if isSimulationOrCheckTx {
		vm = k.buildSimulationVM()
		defer vm.Destroy()

		gasForRuntime = k.config.ContractSimulationGasLimit
	} else if gasMeter.Limit() == 0 {
		// infinite gas meter
		gasForRuntime = math.MaxUint64
	}

	// delegate gas metering to move vm
	ctx = ctx.WithGasMeter(sdk.NewInfiniteGasMeter())
	kvStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.PrefixKeyVMStore)

	// normalize type tags
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
	signers := make([][]byte, len(senders))
	for i, signer := range senders {
		signers[i] = signer[:]
		sendersStr[i] = signer.String()
	}

	api := NewApi(k, ctx)
	env := types.NewEnv(
		ctx,
		types.NextAccountNumber(ctx, k.authKeeper),
		k.IncreaseExecutionCounter(ctx),
	)

	execRes, err := vm.ExecuteEntryFunction(
		kvStore,
		api,
		env,
		gasForRuntime,
		signers,
		payload,
	)

	// Mark loader cache loads new published modules.
	if !isSimulationOrCheckTx {
		k.abciListener.SetNewPublishedModulesLoaded(execRes.NewPublishedModulesLoaded)
	}

	// consume gas first and check error
	gasMeter.ConsumeGas(execRes.GasUsed, "move runtime")
	if err != nil {
		return err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeExecute,
		sdk.NewAttribute(types.AttributeKeySender, strings.Join(sendersStr, ",")),
		sdk.NewAttribute(types.AttributeKeyModuleAddr, moduleAddr.String()),
		sdk.NewAttribute(types.AttributeKeyModuleName, moduleName),
		sdk.NewAttribute(types.AttributeKeyFunctionName, functionName),
	))

	return k.handleExecuteResponse(ctx, gasMeter, execRes)
}

func (k Keeper) ExecuteScript(
	ctx sdk.Context,
	sender sdk.AccAddress,
	byteCodes []byte,
	typeArgs []vmtypes.TypeTag,
	args [][]byte,
) error {
	return k.ExecuteScriptWithMultiSenders(
		ctx,
		[]sdk.AccAddress{sender},
		byteCodes,
		typeArgs,
		args,
	)
}

func (k Keeper) ExecuteScriptWithMultiSenders(
	ctx sdk.Context,
	senders []sdk.AccAddress,
	byteCodes []byte,
	typeArgs []vmtypes.TypeTag,
	args [][]byte,
) error {
	vm := k.moveVM
	gasMeter := ctx.GasMeter()
	gasForRuntime := gasMeter.Limit() - gasMeter.GasConsumedToLimit()

	isSimulationOrCheckTx := isSimulationOrCheckTx(ctx)
	if isSimulationOrCheckTx {
		vm = k.buildSimulationVM()
		defer vm.Destroy()

		gasForRuntime = k.config.ContractSimulationGasLimit
	} else if gasMeter.Limit() == 0 {
		// infinite gas meter
		gasForRuntime = math.MaxUint64
	}

	// delegate gas metering to move vm
	ctx = ctx.WithGasMeter(sdk.NewInfiniteGasMeter())
	kvStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.PrefixKeyVMStore)

	// normalize type tags
	payload, err := types.BuildExecuteScriptPayload(
		byteCodes,
		typeArgs,
		args,
	)
	if err != nil {
		return err
	}

	sendersStr := make([]string, len(senders))
	signers := make([][]byte, len(senders))
	for i, signer := range senders {
		signers[i] = signer
		sendersStr[i] = signer.String()
	}

	api := NewApi(k, ctx)
	env := types.NewEnv(
		ctx,
		types.NextAccountNumber(ctx, k.authKeeper),
		k.IncreaseExecutionCounter(ctx),
	)

	execRes, err := vm.ExecuteScript(
		kvStore,
		api,
		env,
		gasForRuntime,
		signers,
		payload,
	)

	// Mark loader cache loads new published modules.
	if !isSimulationOrCheckTx {
		k.abciListener.SetNewPublishedModulesLoaded(execRes.NewPublishedModulesLoaded)
	}

	// consume gas first and check error
	gasMeter.ConsumeGas(execRes.GasUsed, "move runtime")
	if err != nil {
		return err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeScript,
		sdk.NewAttribute(types.AttributeKeySender, strings.Join(sendersStr, ",")),
	))

	return k.handleExecuteResponse(ctx, gasMeter, execRes)
}

func (k Keeper) handleExecuteResponse(
	ctx sdk.Context,
	gasMeter sdk.GasMeter,
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

		var accI authtypes.AccountI
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
func (k Keeper) DispatchMessages(ctx sdk.Context, messages []vmtypes.CosmosMessage) error {
	for _, message := range messages {
		msg, err := types.ConvertToSDKMessage(ctx, NewMoveBankKeeper(&k), NewNftKeeper(&k), message)
		if err != nil {
			return err
		}

		// validate msg
		if err := msg.ValidateBasic(); err != nil {
			return err
		}

		// find the handler
		handler := k.msgRouter.Handler(msg)
		if handler == nil {
			return types.ErrNotSupportedCosmosMessage
		}

		//  and execute it
		res, err := handler(ctx, msg)
		if err != nil {
			return err
		}

		// emit events
		ctx.EventManager().EmitEvents(res.GetEvents())
	}

	return nil
}

// DistributeContractSharedRevenue distribute a portion of gas fee to contract creator account
func (k Keeper) DistributeContractSharedRevenue(ctx sdk.Context, gasUsages []vmtypes.GasUsage) error {
	value := ctx.Value(ante.GasPricesContextKey)
	if value == nil {
		return nil
	}

	gasPrices := value.(sdk.DecCoins)
	revenueGasPrices := gasPrices.MulDec(k.ContractSharedRevenueRatio(ctx))
	if revenueGasPrices.IsZero() {
		return nil
	}

	for _, gasUsage := range gasUsages {

		// ignore 0x1 gas usage
		if vmtypes.StdAddress.Equals(gasUsage.ModuleId.Address) {
			continue
		}

		revenue, _ := revenueGasPrices.MulDec(sdk.NewDec(int64(gasUsage.GasUsed))).TruncateDecimal()
		if revenue.IsZero() {
			continue
		}

		creatorAddr := types.ConvertVMAddressToSDKAddress(gasUsage.ModuleId.Address)
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, k.feeCollector, creatorAddr, revenue); err != nil {
			return err
		}

		ctx.EventManager().EmitEvent(sdk.NewEvent(
			types.EventTypeContractSharedRevenue,
			sdk.NewAttribute(types.AttributeKeyCreator, gasUsage.ModuleId.Address.String()),
			sdk.NewAttribute(types.AttributeKeyRevenue, revenue.String()),
		))
	}

	return nil
}
