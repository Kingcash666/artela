package txs

import (
	"math/big"
	"os"

	"github.com/artela-network/artela-evm/tracers/logger"

	"github.com/artela-network/artela-evm/vm"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/params"
)

const (
	TracerAccessList = "access_list"

	TracerJSON = "json"

	TracerStruct = "struct"

	TracerMarkdown = "markdown"
)

var _ vm.EVMLogger = &NoOpTracer{}

// TxTraceResult is the result of a single txs trace during a block trace.
type TxTraceResult struct {
	Result interface{} `json:"result,omitempty"` // Trace results produced by the tracer
	Error  string      `json:"error,omitempty"`  // Trace failure produced by the tracer
}

// NewTracer creates a new Logger tracer to collect execution traces from an
// EVM txs.
func NewTracer(tracer string, msg *core.Message, cfg *params.ChainConfig, height int64) vm.EVMLogger {
	// TODO: enable additional log configuration
	logCfg := &logger.Config{
		Debug: true,
	}

	switch tracer {
	case TracerAccessList:
		preCompiles := vm.ActivePrecompiles(cfg.Rules(big.NewInt(height), cfg.MergeNetsplitBlock != nil, *cfg.PragueTime))
		return logger.NewAccessListTracer(msg.AccessList, msg.From, *msg.To, preCompiles)
	case TracerJSON:
		return logger.NewJSONLogger(logCfg, os.Stderr)
	case TracerMarkdown:
		return logger.NewMarkdownLogger(logCfg, os.Stdout) // TODO: Stderr ?
	case TracerStruct:
		return logger.NewStructLogger(logCfg)
	default:
		return NewNoOpTracer()
	}
}

// ===============================================================
//          		        NoOp Tracer
// ===============================================================

// NoOpTracer is an empty implementation of vm.Tracer interface
type NoOpTracer struct{}

// NewNoOpTracer creates a no-op vm.Tracer
func NewNoOpTracer() *NoOpTracer {
	return &NoOpTracer{}
}

// CaptureStart implements vm.Tracer interface
//
//nolint:revive // allow unused parameters to indicate expected signature
func (dt NoOpTracer) CaptureStart(env *vm.EVM,
	from common.Address,
	to common.Address,
	create bool,
	input []byte,
	gas uint64,
	value *big.Int) {
}

// CaptureState implements vm.Tracer interface
//
//nolint:revive // allow unused parameters to indicate expected signature
func (dt NoOpTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
}

// CaptureFault implements vm.Tracer interface
//
//nolint:revive // allow unused parameters to indicate expected signature
func (dt NoOpTracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {
}

// CaptureEnd implements vm.Tracer interface
//
//nolint:revive // allow unused parameters to indicate expected signature
func (dt NoOpTracer) CaptureEnd(output []byte, gasUsed uint64, err error) {}

// CaptureEnter implements vm.Tracer interface
//
//nolint:revive // allow unused parameters to indicate expected signature
func (dt NoOpTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
}

// CaptureExit implements vm.Tracer interface
//
//nolint:revive // allow unused parameters to indicate expected signature
func (dt NoOpTracer) CaptureExit(output []byte, gasUsed uint64, err error) {}

// CaptureTxStart implements vm.Tracer interface
//
//nolint:revive // allow unused parameters to indicate expected signature
func (dt NoOpTracer) CaptureTxStart(gasLimit uint64) {}

// CaptureTxEnd implements vm.Tracer interface
//
//nolint:revive // allow unused parameters to indicate expected signature
func (dt NoOpTracer) CaptureTxEnd(restGas uint64) {}
