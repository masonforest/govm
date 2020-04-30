package vm

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/ioutil"
	"math/big"
	"os"
	"reflect"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	ErrImpureInitcode       = errors.New("initCode is impure")
	ExecutionManagerAddress = common.HexToAddress(os.Getenv("EXECUTION_MANAGER_ADDRESS"))
	PurityCheckerAddress    = common.HexToAddress(os.Getenv("PURITY_CHECKER_ADDRESS"))
	WORD_SIZE               = 32
)

type ovmOperation func(*EVM, ContractRef, *Contract, []byte) ([]byte, error)
type methodId [4]byte

var funcs = map[string]ovmOperation{
	"SSTORE": sStore,
	"SLOAD":  sLoad,
	// "CREATE":  create,
	// "CALL":  call,
	"CREATE2": create2,
}
var methodIds map[[4]byte]ovmOperation
var executionMangerBytecode []byte

type ContractJSON struct {
	Bytecode string
}

func init() {
	executionMangerBytecode = readExecutionManagerBytecode()
	methodIds = make(map[[4]byte]ovmOperation, len(funcs))
	for methodName, f := range funcs {
		methodIds[OvmMethodId(methodName)] = f
	}
}

func readExecutionManagerBytecode() []byte {
	rawExecutionManger, _ := ioutil.ReadFile("../ExecutionManager.json")
	var executionManger ContractJSON
	json.Unmarshal(rawExecutionManger, &executionManger)
	bytecode, _ := hex.DecodeString(executionManger.Bytecode)
	return bytecode
}

func OvmMethodId(methodName string) [4]byte {
	var methodId [4]byte
	var fullMethodName = "ovm" + methodName + "()"
	copy(methodId[:], crypto.Keccak256([]byte(fullMethodName)))
	return methodId
}

func isOvmOperation(contract *Contract, input []byte) bool {
	if contract.Address() != ExecutionManagerAddress {
		return false
	} else {
	}
	if len(input) < 4 {
		return false
	}
	for methodId := range methodIds {
		if bytes.Equal(input[0:4], methodId[:]) {
			return true
		} else {
		}
	}
	return false
}

func runOvmOperation(input []byte, evm *EVM, caller ContractRef, contract *Contract) (ret []byte, err error) {
	var methodId [4]byte
	copy(methodId[:], input[:4])
	evm.StateDB.SetCode(contract.Address(), executionMangerBytecode)
	ret, err = methodIds[methodId](evm, caller, contract, input)
	evm.StateDB.SetCode(contract.Address(), []byte{})
	return ret, err
}

func create(evm *EVM, caller ContractRef, contract *Contract, input []byte) (ret []byte, err error) {
	initCode := input[4:]
	gas := contract.Gas
	if evm.chainRules.IsEIP150 {
		gas -= gas / 64
	}
	contract.UseGas(gas)
	activeContractHash := evm.StateDB.GetState(ExecutionManagerAddress, common.BigToHash(big.NewInt(9)))
	activeContract := common.BytesToAddress(activeContractHash.Bytes())
	activeContractRef := &Contract{self: AccountRef(activeContract)}

	_, address, _, _ := evm.Create(activeContractRef, initCode, contract.Gas, big.NewInt(0))

	emitActiveContract(evm, contract, caller.Address())
	emitCreatedContract(evm, contract, caller.Address(), address, [32]byte{1})
	return address.Bytes(), nil
}

func create2(evm *EVM, caller ContractRef, contract *Contract, input []byte) (ret []byte, err error) {
	initCode := input[4:]
	gas := contract.Gas
	if evm.chainRules.IsEIP150 {
		gas -= gas / 64
	}
	contract.UseGas(gas)

	_, address, _, _ := evm.Create2(caller, initCode, contract.Gas, big.NewInt(0), big.NewInt(0))
	return address.Bytes(), nil
}

func call(evm *EVM, caller ContractRef, contract *Contract, input []byte) (ret []byte, err error) {

	to := common.BytesToAddress(input[0:20])
	args := input[20:]
	ret, _, err = evm.Call(contract, to, args, contract.Gas, big.NewInt(0))
	return ret, err
}

func sLoad(evm *EVM, caller ContractRef, contract *Contract, input []byte) (ret []byte, err error) {
	key := common.BytesToHash(input[4:36])
	val := evm.StateDB.GetState(caller.Address(), key)
	return val.Bytes(), nil
}
func sStore(evm *EVM, caller ContractRef, contract *Contract, input []byte) (ret []byte, err error) {
	key := common.BytesToHash(input[4:36])
	val := common.BytesToHash(input[36:68])
	evm.StateDB.SetState(caller.Address(), key, val)
	return []byte{}, nil
}

func isPure(evm *EVM, caller ContractRef, gas uint64, code []byte) bool {
	return true
}

func emitActiveContract(
	evm *EVM,
	contract *Contract,
	contractAddress common.Address,
) {
	typ, _ := abi.NewType("(address)", "", []abi.ArgumentMarshaling{})
	data, _ := typ.Pack(reflect.ValueOf(contractAddress))
	emitEvent(evm, contract, "ActiveContract(address)", data)
}

func emitCreatedContract(
	evm *EVM,
	contract *Contract,
	ovmContractAddress common.Address,
	codeContractAddress common.Address,
	codeContractHash [32]byte,
) {
	typ, _ := abi.NewType("tuple", "", []abi.ArgumentMarshaling{
		{Name: "a", Type: "address"},
		{Name: "b", Type: "address"},
		{Name: "c", Type: "bytes32"},
	})
	data, _ := typ.Pack(reflect.ValueOf(struct {
		A common.Address
		B common.Address
		C [32]byte
	}{
		ovmContractAddress,
		codeContractAddress,
		codeContractHash,
	}))

	emitEvent(evm, contract, "CreatedContract(address,address,bytes32)", data)
}

func emitEvent(evm *EVM, contract *Contract, topic string, data []byte) {
	topics := []common.Hash{common.BytesToHash(crypto.Keccak256([]byte(topic)))}
	evm.StateDB.AddLog(&types.Log{
		Address:     contract.Address(),
		Topics:      topics,
		Data:        data,
		BlockNumber: evm.BlockNumber.Uint64(),
	})
}
