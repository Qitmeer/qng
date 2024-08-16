package testcommon

import (
	"math/big"

	"github.com/Qitmeer/qng/testutils"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func CreateErc20(node *testutils.MockNode) (string, error) {
	return testutils.CreateLegacyTx(node, node.GetBuilder().Get(0), nil, 0, 0, big.NewInt(0), common.FromHex(ERC20Code), GAS_LIMIT, CHAIN_ID)
}
func AuthTrans(privatekeybyte []byte) (*bind.TransactOpts, error) {
	privateKey := crypto.ToECDSAUnsafe(privatekeybyte)
	authCaller, err := bind.NewKeyedTransactorWithChainID(privateKey, CHAIN_ID)
	if err != nil {
		return nil, err
	}
	authCaller.GasLimit = uint64(GAS_LIMIT)
	return authCaller, nil
}
