package tests

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/runtime"
	"github.com/ethereum/go-ethereum/params"
)

var KEY = common.FromHex("0102030000000000000000000000000000000000000000000000000000000000")
var VALUE1 = common.FromHex("0405060000000000000000000000000000000000000000000000000000000000")
var VALUE2 = common.FromHex("0708090000000000000000000000000000000000000000000000000000000000")
var chainConfig params.ChainConfig

func init() {
	chainConfig = params.ChainConfig{
		ChainID:             big.NewInt(1),
		HomesteadBlock:      new(big.Int),
		ByzantiumBlock:      new(big.Int),
		ConstantinopleBlock: new(big.Int),
		DAOForkBlock:        new(big.Int),
		DAOForkSupport:      false,
		EIP150Block:         new(big.Int),
		EIP155Block:         new(big.Int),
		EIP158Block:         new(big.Int),
	}
	vm.ExecutionManagerAddress = common.HexToAddress("0x0000000000000000000000000000000000000000")
}

func TestSloadAndStore(t *testing.T) {
	state := newState()
	storeCode := ovmMethodId("SSTORE")
	storeCode = append(storeCode, KEY...)
	storeCode = append(storeCode, VALUE1...)
	loadCode := ovmMethodId("SLOAD")
	loadCode = append(loadCode, KEY...)
	call(t, state, vm.ExecutionManagerAddress, storeCode)
	returnValue, _ := call(t, state, vm.ExecutionManagerAddress, loadCode)
	if !bytes.Equal(VALUE1, returnValue) {
		t.Errorf("Expected %020x; got %020x", VALUE1, returnValue)
	}
}

func newState() *state.StateDB {
	db := state.NewDatabase(rawdb.NewMemoryDatabase())
	state, _ := state.New(common.Hash{}, db, nil)
	return state
}
func call(t *testing.T, state *state.StateDB, address common.Address, callData []byte) ([]byte, error) {
	returnValue, _, err := runtime.Call(address, callData, &runtime.Config{
		State:       state,
		ChainConfig: &chainConfig,
	})

	return returnValue, err
}

func int64ToBytes(n int64) []byte {
	if bytes.Equal(big.NewInt(n).Bytes(), []byte{}) {
		return []byte{0}
	} else {
		return big.NewInt(n).Bytes()
	}
}
func pushN(n int64) byte {
	return byte(int(vm.PUSH1) + byteLength(n) - 1)
}
func byteLength(n int64) int {
	if bytes.Equal(big.NewInt(n).Bytes(), []byte{}) {
		return 1
	} else {
		return len(big.NewInt(n).Bytes())
	}
}

func mockPurityChecker(pure bool) []byte {
	var pureByte byte

	if pure {
		pureByte = 1
	} else {
		pureByte = 0
	}

	return []byte{
		byte(vm.PUSH1),
		pureByte,
		byte(vm.PUSH1),
		0,
		byte(vm.MSTORE8),
		byte(vm.PUSH1),
		1,
		byte(vm.PUSH1),
		0,
		byte(vm.RETURN),
	}
}

func ovmMethodId(methodName string) []byte {
	fixedBytes := vm.OvmMethodId(methodName)
	return fixedBytes[:]
}
