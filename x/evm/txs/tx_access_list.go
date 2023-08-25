package txs

import (
	errorsmod "cosmossdk.io/errors"
	types3 "github.com/artela-network/artela/ethereum/types"
	types2 "github.com/artela-network/artela/x/evm/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"math/big"

	sdkmath "cosmossdk.io/math"
	"github.com/ethereum/go-ethereum/common"
	ethereum "github.com/ethereum/go-ethereum/core/types"
)

func newAccessListTx(tx *ethereum.Transaction) (*AccessListTx, error) {
	alTx := &AccessListTx{
		Nonce:    tx.Nonce(),
		Data:     tx.Data(),
		GasLimit: tx.Gas(),
	}

	// fill the to address
	v, r, s := tx.RawSignatureValues()
	if to := tx.To(); to != nil {
		alTx.To = to.Hex()
	}

	// fill the amount
	if tx.Value() != nil {
		amountInt, err := types3.SafeNewIntFromBigInt(tx.Value())
		if err != nil {
			return nil, err
		}
		alTx.Amount = &amountInt
	}

	// fill the gas price
	if tx.GasPrice() != nil {
		gasPriceInt, err := types3.SafeNewIntFromBigInt(tx.GasPrice())
		if err != nil {
			return nil, err
		}
		alTx.GasPrice = &gasPriceInt
	}

	// fill the access list
	if tx.AccessList() != nil {
		al := tx.AccessList()
		alTx.Accesses = NewAccessList(&al)
	}

	// fill the (v,r,s)
	alTx.SetSignatureValues(tx.ChainId(), v, r, s)
	return alTx, nil
}

// TxType returns the txs type
func (tx *AccessListTx) TxType() uint8 {
	return ethereum.AccessListTxType
}

// Copy returns an instance with the same field values
func (tx *AccessListTx) Copy() TxData {
	return &AccessListTx{
		ChainID:  tx.ChainID,
		Nonce:    tx.Nonce,
		GasPrice: tx.GasPrice,
		GasLimit: tx.GasLimit,
		To:       tx.To,
		Amount:   tx.Amount,
		Data:     common.CopyBytes(tx.Data),
		Accesses: tx.Accesses,
		V:        common.CopyBytes(tx.V),
		R:        common.CopyBytes(tx.R),
		S:        common.CopyBytes(tx.S),
	}
}

// Validate performs a stateless validation of the txs fields
func (tx AccessListTx) Validate() error {
	gasPrice := tx.GetGasPrice()
	if gasPrice == nil {
		return errorsmod.Wrap(types2.ErrInvalidGasPrice, "cannot be nil")
	}
	if !types3.IsValidInt256(gasPrice) {
		return errorsmod.Wrap(types2.ErrInvalidGasPrice, "out of bound")
	}

	if gasPrice.Sign() == -1 {
		return errorsmod.Wrapf(types2.ErrInvalidGasPrice, "gas price cannot be negative %s", gasPrice)
	}

	amount := tx.GetValue()
	// Amount can be 0
	if amount != nil && amount.Sign() == -1 {
		return errorsmod.Wrapf(types2.ErrInvalidAmount, "amount cannot be negative %s", amount)
	}
	if !types3.IsValidInt256(amount) {
		return errorsmod.Wrap(types2.ErrInvalidAmount, "out of bound")
	}

	if !types3.IsValidInt256(tx.Fee()) {
		return errorsmod.Wrap(types2.ErrInvalidGasFee, "out of bound")
	}

	if tx.To != "" {
		if err := types3.ValidateAddress(tx.To); err != nil {
			return errorsmod.Wrap(err, "invalid to address")
		}
	}

	chainID := tx.GetChainID()

	if chainID == nil {
		return errorsmod.Wrap(
			errortypes.ErrInvalidChainID,
			"chain ID must be present on AccessList txs",
		)
	}

	// TODO mark
	if !(chainID.Cmp(big.NewInt(9001)) == 0 || chainID.Cmp(big.NewInt(9000)) == 0) {
		return errorsmod.Wrapf(
			errortypes.ErrInvalidChainID,
			"chain ID must be 9000 or 9001 on Artela, got %s", chainID,
		)
	}

	return nil
}

// GetChainID returns the chain id field from the AccessListTx
func (tx *AccessListTx) GetChainID() *big.Int {
	if tx.ChainID == nil {
		return nil
	}

	return tx.ChainID.BigInt()
}

// GetAccessList returns the AccessList field
func (tx *AccessListTx) GetAccessList() ethereum.AccessList {
	if tx.Accesses == nil {
		return nil
	}
	return *tx.Accesses.ToEthAccessList()
}

// GetData returns the copy of the input data bytes
func (tx *AccessListTx) GetData() []byte {
	return common.CopyBytes(tx.Data)
}

// GetGas returns the gas limit
func (tx *AccessListTx) GetGas() uint64 {
	return tx.GasLimit
}

// GetGasPrice returns the gas price field
func (tx *AccessListTx) GetGasPrice() *big.Int {
	if tx.GasPrice == nil {
		return nil
	}
	return tx.GasPrice.BigInt()
}

// Fee returns gasprice * gaslimit
func (tx AccessListTx) Fee() *big.Int {
	return fee(tx.GetGasPrice(), tx.GetGas())
}

// Cost returns amount + gasprice * gaslimit
func (tx AccessListTx) Cost() *big.Int {
	return cost(tx.Fee(), tx.GetValue())
}

// EffectiveGasPrice is the same as GasPrice for AccessListTx
func (tx AccessListTx) EffectiveGasPrice(_ *big.Int) *big.Int {
	return tx.GetGasPrice()
}

// EffectiveFee is the same as Fee for AccessListTx
func (tx AccessListTx) EffectiveFee(_ *big.Int) *big.Int {
	return tx.Fee()
}

// EffectiveCost is the same as Cost for AccessListTx
func (tx AccessListTx) EffectiveCost(_ *big.Int) *big.Int {
	return tx.Cost()
}

// GetGasTipCap returns the gas price field
func (tx *AccessListTx) GetGasTipCap() *big.Int {
	return tx.GetGasPrice()
}

// GetGasFeeCap returns the gas price field
func (tx *AccessListTx) GetGasFeeCap() *big.Int {
	return tx.GetGasPrice()
}

// GetValue returns the txs amount
func (tx *AccessListTx) GetValue() *big.Int {
	if tx.Amount == nil {
		return nil
	}

	return tx.Amount.BigInt()
}

// GetNonce returns the account sequence for the txs
func (tx *AccessListTx) GetNonce() uint64 { return tx.Nonce }

// GetTo returns the pointer to the recipient address
func (tx *AccessListTx) GetTo() *common.Address {
	if tx.To == "" {
		return nil
	}
	to := common.HexToAddress(tx.To)
	return &to
}

// AsEthereumData returns an AccessListTx txs from the proto-formatted
func (tx *AccessListTx) AsEthereumData() ethereum.TxData {
	v, r, s := tx.GetRawSignatureValues()
	return &ethereum.AccessListTx{
		ChainID:    tx.GetChainID(),
		Nonce:      tx.GetNonce(),
		GasPrice:   tx.GetGasPrice(),
		Gas:        tx.GetGas(),
		To:         tx.GetTo(),
		Value:      tx.GetValue(),
		Data:       tx.GetData(),
		AccessList: tx.GetAccessList(),
		V:          v,
		R:          r,
		S:          s,
	}
}

// GetRawSignatureValues returns the V, R, S signature values of the txs
func (tx *AccessListTx) GetRawSignatureValues() (v, r, s *big.Int) {
	return rawSignatureValues(tx.V, tx.R, tx.S)
}

// SetSignatureValues sets the signature values to the txs
func (tx *AccessListTx) SetSignatureValues(chainID, v, r, s *big.Int) {
	if v != nil {
		tx.V = v.Bytes()
	}
	if r != nil {
		tx.R = r.Bytes()
	}
	if s != nil {
		tx.S = s.Bytes()
	}
	if chainID != nil {
		chainIDInt := sdkmath.NewIntFromBigInt(chainID)
		tx.ChainID = &chainIDInt
	}
}
