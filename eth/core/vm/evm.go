// Copyright (C) 2022, Berachain Foundation. All rights reserved.
// See the file LICENSE for licensing terms.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
// AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
// FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
// DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
// SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
// CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
// OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package vm

import (
	"math/big"

	"github.com/berachain/stargazer/eth/params"
	"github.com/berachain/stargazer/lib/common"
)

type StargazerEVM interface {
	Reset(txCtx TxContext, sdb GethStateDB)
	Create(caller ContractRef, code []byte,
		gas uint64, value *big.Int,
	) (ret []byte, contractAddr common.Address, leftOverGas uint64, err error)
	Call(caller ContractRef, addr common.Address, input []byte,
		gas uint64, value *big.Int,
	) (ret []byte, leftOverGas uint64, err error)

	StateDB() StargazerStateDB
}

// Compile-time assertion to ensure `StargazerEVM` implements `VMInterface`.
var _ StargazerEVM = (*stargazerEVM)(nil)

// `StargazerEVM` is the wrapper for the Go-Ethereum EVM.
type stargazerEVM struct {
	*GethEVM
}

// `NewStargazerEVM` creates and returns a new `StargazerEVM`.
func NewStargazerEVM(
	blockCtx BlockContext,
	txCtx TxContext,
	stateDB StargazerStateDB,
	chainConfig *params.EthChainConfig,
	config Config,
	pctr PrecompileController,
) StargazerEVM {
	return &stargazerEVM{
		GethEVM: NewGethEVMWithPrecompiles(
			blockCtx, txCtx, stateDB, chainConfig, config, pctr,
		),
	}
}

func (evm *stargazerEVM) StateDB() StargazerStateDB {
	return evm.GethEVM.StateDB.(StargazerStateDB)
}