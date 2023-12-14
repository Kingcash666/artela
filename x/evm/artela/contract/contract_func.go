package contract

import (
	"github.com/artela-network/artela-evm/vm"
	"github.com/artela-network/artela/x/evm/artela/types"
	evmtypes "github.com/artela-network/artela/x/evm/txs"
	"github.com/artela-network/aspect-core/djpm/contract"
	"github.com/artela-network/aspect-core/djpm/run"
	artelasdkType "github.com/artela-network/aspect-core/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/holiman/uint256"
	"github.com/pkg/errors"
)

func (anc *AspectNativeContract) entrypoint(ctx sdk.Context, msg *core.Message, method *abi.Method, aspectId common.Address, data []byte) (*evmtypes.MsgEthereumTxResponse, error) {
	lastHeight := ctx.BlockHeight()
	code, _ := anc.aspectService.GetAspectCode(lastHeight-1, aspectId)
	runner, newErr := run.NewRunner(aspectId.String(), code)
	if newErr != nil {
		return nil, newErr
	}
	defer runner.Return()

	transaction := &artelasdkType.EthTransaction{
		BlockNumber: ctx.BlockHeight(),
		From:        artelasdkType.ARTELA_ADDR,
		Input:       data,
		To:          aspectId.String(),
	}
	ret, runErr := runner.JoinPoint(artelasdkType.OPERATION_METHOD, msg.GasLimit, lastHeight, msg.To, transaction)
	vmError := ""
	retByte := make([]byte, 0)
	if runErr != nil {
		vmError = runErr.Error()
	} else {
		byteData := &artelasdkType.BytesData{}
		dataUnpackErr := ret.Data.UnmarshalTo(byteData)
		if dataUnpackErr != nil {
			vmError = dataUnpackErr.Error()
		}
		retByte, dataUnpackErr = method.Outputs.Pack(byteData.Data)
		if dataUnpackErr != nil {
			vmError = dataUnpackErr.Error()
		}
	}
	return &evmtypes.MsgEthereumTxResponse{
		GasUsed: ctx.GasMeter().GasConsumed(),
		VmError: vmError,
		Ret:     retByte,
		Logs:    nil,
	}, nil
}

func (anc *AspectNativeContract) contractsOf(ctx sdk.Context, method *abi.Method, aspectId common.Address) (*evmtypes.MsgEthereumTxResponse, error) {
	lastHeight := ctx.BlockHeight() - 1
	value, err := anc.aspectService.GetAspectOf(lastHeight, aspectId)
	if err != nil {
		return nil, err
	}
	addressAry := make([]common.Address, 0)
	for _, data := range value.Values() {
		contractAddr := common.HexToAddress(data.(string))
		addressAry = append(addressAry, contractAddr)
	}

	ret, err := method.Outputs.Pack(addressAry)
	if err != nil {
		return nil, err
	}
	return &evmtypes.MsgEthereumTxResponse{
		GasUsed: ctx.GasMeter().GasConsumed(),
		VmError: "",
		Ret:     ret,
		Logs:    nil,
	}, nil
}

func (k *AspectNativeContract) bind(ctx sdk.Context, aspectId common.Address, account common.Address, aspectVersion *uint256.Int, priority int8, isContract bool) (*evmtypes.MsgEthereumTxResponse, error) {
	// check aspect types
	txAspect, err := k.checkIsTransactionLevel(ctx, aspectId)
	if err != nil {
		return nil, err
	}

	txVerifier, err := k.checkIsTxVerifier(ctx, aspectId)
	if err != nil {
		return nil, err
	}

	// EoA can only bind with tx verifier
	if !txVerifier && !isContract {
		return nil, errors.New("unable to bind non-tx-verifier Aspect to an EoA account")
	}

	// bind tx processing aspect if account is a contract
	if txAspect && isContract {
		if err := k.aspectService.aspectStore.BindTxAspect(ctx, account, aspectId, aspectVersion, priority); err != nil {
			return nil, err
		}
	}

	// bind tx verifier aspect
	if txVerifier {
		if err := k.aspectService.aspectStore.BindVerificationAspect(ctx, account, aspectId, aspectVersion, priority, isContract); err != nil {
			return nil, err
		}
	}

	// save reverse index
	if err := k.aspectService.aspectStore.StoreAspectRefValue(ctx, account, aspectId); err != nil {
		return nil, err
	}

	return &evmtypes.MsgEthereumTxResponse{
		GasUsed: ctx.GasMeter().GasConsumed(),
		VmError: "",
		Ret:     nil,
		Logs:    nil,
	}, nil
}

func (k *AspectNativeContract) unbind(ctx sdk.Context, aspectId common.Address, contract common.Address, isContract bool) (*evmtypes.MsgEthereumTxResponse, error) {
	// contract=>aspect object
	// aspectId= [contract,contract]
	if !isContract {
		if err := k.aspectService.aspectStore.UnBindVerificationAspect(ctx, contract, aspectId); err != nil {
			return nil, err
		}
	}
	if err := k.aspectService.aspectStore.UnBindContractAspects(ctx, contract, aspectId); err != nil {
		return nil, err
	}
	if err := k.aspectService.aspectStore.UnbindAspectRefValue(ctx, contract, aspectId); err != nil {
		return nil, err
	}
	return &evmtypes.MsgEthereumTxResponse{
		GasUsed: ctx.GasMeter().GasConsumed(),
		VmError: "",
		Ret:     nil,
		Logs:    nil,
	}, nil
}

func (k *AspectNativeContract) changeVersion(ctx sdk.Context, aspectId common.Address, contract common.Address, version uint64) (*evmtypes.MsgEthereumTxResponse, error) {
	err := k.aspectService.aspectStore.ChangeBoundAspectVersion(ctx, contract, aspectId, version)
	if err != nil {
		return nil, err
	}

	return &evmtypes.MsgEthereumTxResponse{
		GasUsed: 100,
		VmError: "",
		Ret:     nil,
		Logs:    nil,
	}, nil
}

func (k *AspectNativeContract) version(ctx sdk.Context, method *abi.Method, aspectId common.Address) (*evmtypes.MsgEthereumTxResponse, error) {
	version := k.aspectService.aspectStore.GetAspectLastVersion(ctx, aspectId)

	ret, err := method.Outputs.Pack(version.Uint64())
	if err != nil {
		return nil, err
	}

	return &evmtypes.MsgEthereumTxResponse{
		GasUsed: 100,
		VmError: "",
		Ret:     ret,
		Logs:    nil,
	}, nil
}

func (k *AspectNativeContract) aspectsOf(ctx sdk.Context, method *abi.Method, contract common.Address, isContract bool) (*evmtypes.MsgEthereumTxResponse, error) {

	aspectInfo := make([]types.AspectInfo, 0)
	if isContract {
		aspects, err := k.aspectService.GetAspectForAddr(ctx.BlockHeight()-1, contract)
		if err != nil {
			return nil, err
		}
		for _, aspect := range aspects {
			info := types.AspectInfo{
				AspectId: common.HexToAddress(aspect.AspectId),
				Version:  aspect.Version,
				Priority: int8(aspect.Priority),
			}
			aspectInfo = append(aspectInfo, info)
		}
	} else {
		aspectAccounts, verErr := k.aspectService.GetAccountVerifiers(ctx.BlockHeight()-1, contract)
		if verErr != nil {
			return nil, verErr
		}
		for _, aspect := range aspectAccounts {
			info := types.AspectInfo{
				AspectId: common.HexToAddress(aspect.AspectId),
				Version:  aspect.Version,
				Priority: int8(aspect.Priority),
			}
			aspectInfo = append(aspectInfo, info)
		}
	}

	ret, pkErr := method.Outputs.Pack(aspectInfo)
	if pkErr != nil {
		return nil, pkErr
	}
	return &evmtypes.MsgEthereumTxResponse{
		GasUsed: ctx.GasMeter().GasConsumed(),
		VmError: "",
		Ret:     ret,
		Logs:    nil,
	}, nil
}

func (k *AspectNativeContract) checkContractOwner(ctx sdk.Context, to *common.Address, nonce uint64, sender common.Address) bool {
	msg, err := contract.ArtelaOwnerMsg(to, nonce, sender)
	if err != nil {
		return false
	}
	fromAccount := vm.AccountRef(msg.From)
	k.evm.CloseAspectCall()
	defer k.evm.AspectCall()
	ret, _, err := k.evm.Call(fromAccount, *msg.To, msg.Data, msg.GasLimit, msg.Value)
	if err != nil {
		return false
	}
	result, err := contract.UnpackIsOwnerResult(ret)
	if err != nil {
		return false
	}
	return result
}

func (k *AspectNativeContract) checkAspectOwner(ctx sdk.Context, aspectId common.Address, sender common.Address) (bool, error) {
	bHeight := ctx.BlockHeight() - 1
	code, _ := k.aspectService.GetAspectCode(bHeight, aspectId)
	if code == nil {
		return false, nil
	}
	runner, newErr := run.NewRunner(aspectId.String(), code)
	if newErr != nil {
		return false, newErr
	}
	defer runner.Return()

	binding, runErr := runner.IsOwner(bHeight, 0, &sender, sender.String())
	return binding, runErr
}

func (k *AspectNativeContract) CheckIsAspectOwnerByCode(ctx sdk.Context, aspectId common.Address, code []byte, sender common.Address) (bool, error) {
	runner, newErr := run.NewRunner(aspectId.String(), code)
	if newErr != nil {
		return false, newErr
	}
	defer runner.Return()

	binding, runErr := runner.IsOwner(ctx.BlockHeight(), 0, &sender, sender.String())
	return binding, runErr
}

func (k *AspectNativeContract) deploy(ctx sdk.Context, aspectId common.Address, code []byte, properties []types.Property) (*evmtypes.MsgEthereumTxResponse, error) {
	k.aspectService.aspectStore.StoreAspectCode(ctx, aspectId, code)
	k.aspectService.aspectStore.StoreAspectProperty(ctx, aspectId, properties)

	level, err := k.checkBlockLevel(aspectId, code)
	if err != nil {
		return nil, err
	}
	if level {
		storeErr := k.aspectService.aspectStore.StoreBlockLevelAspect(ctx, aspectId)
		if storeErr != nil {
			return nil, storeErr
		}
	} else {
		// for k.update,one aspect trans to tx level
		removeErr := k.aspectService.aspectStore.RemoveBlockLevelAspect(ctx, aspectId)
		if removeErr != nil {
			return nil, removeErr
		}
	}

	return &evmtypes.MsgEthereumTxResponse{
		GasUsed: ctx.GasMeter().GasConsumed(),
		VmError: "",
		Ret:     nil,
		Logs:    nil,
	}, nil
}

func (k *AspectNativeContract) checkBlockLevel(aspectId common.Address, code []byte) (bool, error) {
	runner, newErr := run.NewRunner(aspectId.String(), code)
	if newErr != nil {
		return false, newErr
	}
	defer runner.Return()

	binding, runErr := runner.IsBlockLevel()
	return binding, runErr
}

func (k *AspectNativeContract) checkIsTransactionLevel(ctx sdk.Context, aspectId common.Address) (bool, error) {
	code, _ := k.aspectService.GetAspectCode(ctx.BlockHeight()-1, aspectId)
	if code == nil {
		return false, nil
	}
	runner, newErr := run.NewRunner(aspectId.String(), code)
	if newErr != nil {
		return false, newErr
	}
	defer runner.Return()

	binding, runErr := runner.IsTransactionLevel()
	return binding, runErr
}

func (k *AspectNativeContract) checkIsTxVerifier(ctx sdk.Context, aspectId common.Address) (bool, error) {
	code, _ := k.aspectService.GetAspectCode(ctx.BlockHeight()-1, aspectId)
	if code == nil {
		return false, nil
	}
	runner, newErr := run.NewRunner(aspectId.String(), code)
	if newErr != nil {
		return false, newErr
	}
	defer runner.Return()

	binding, runErr := runner.IsTxVerifier()
	return binding, runErr
}
