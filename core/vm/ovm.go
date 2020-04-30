package vm

import (
	"errors"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	ErrImpureInitcode       = errors.New("initCode is impure")
	ExecutionManagerAddress = common.HexToAddress(os.Getenv("EXECUTION_MANAGER_ADDRESS"))
	PurityCheckerAddress    = common.HexToAddress(os.Getenv("PURITY_CHECKER_ADDRESS"))
	WORD_SIZE               = 32
)

func OvmMethodId(methodName string) [4]byte {
	var methodId [4]byte
	var fullMethodName = "ovm" + methodName + "()"
	copy(methodId[:], crypto.Keccak256([]byte(fullMethodName)))
	return methodId
}
