package testcommon

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"log"
	"math/big"
	"strings"

	"github.com/Qitmeer/qng/test/swap/factory"
	"github.com/Qitmeer/qng/test/swap/router"
	"github.com/Qitmeer/qng/testutils"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func CreateLegacyTx(node *testutils.MockNode, fromPkByte []byte, to *common.Address, nonce uint64, gas uint64, val *big.Int, d []byte) (string, error) {
	privateKey := crypto.ToECDSAUnsafe(fromPkByte)
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return "", errors.New("private key error")
	}
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	log.Println("from address", fromAddress.String())
	var err error
	if nonce <= 0 {
		nonce, err = node.GetEvmClient().PendingNonceAt(context.Background(), fromAddress)
		if err != nil {
			return "", err
		}
	}
	gasLimit := uint64(GAS_LIMIT) // in units
	if gas > 0 {
		gasLimit = gas
	}
	gasPrice, err := node.GetEvmClient().SuggestGasPrice(context.Background())
	if err != nil {
		return "", err
	}
	data := &types.LegacyTx{
		To:       to,
		Nonce:    nonce,
		Gas:      gasLimit,
		GasPrice: gasPrice,
		Value:    val,
		Data:     d,
	}
	tx := types.NewTx(data)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(CHAIN_ID), privateKey)
	if err != nil {
		return "", err
	}
	err = node.GetEvmClient().SendTransaction(context.Background(), signedTx)
	if err != nil {
		return "", err
	}
	return signedTx.Hash().Hex(), nil
}
func CreateErc20(node *testutils.MockNode) (string, error) {
	return CreateLegacyTx(node, node.GetBuilder().Get(0), nil, 0, 0, big.NewInt(0), common.FromHex(ERC20Code))
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
func CreateWETH(node *testutils.MockNode) (string, error) {
	return CreateLegacyTx(node, node.GetBuilder().Get(0), nil, 0, 0, big.NewInt(0), common.FromHex(WETH))
}

func CreateFactory(node *testutils.MockNode, _feeToSetter common.Address) (string, error) {
	parsed, _ := abi.JSON(strings.NewReader(factory.TokenMetaData.ABI))
	// constructor params
	initP, _ := parsed.Pack("", _feeToSetter)
	return CreateLegacyTx(node, node.GetBuilder().Get(0), nil, 0, 0, big.NewInt(0), append(common.FromHex(FACTORY), initP...))
}

func CreateRouter(node *testutils.MockNode, factory, weth common.Address) (string, error) {
	parsed, _ := abi.JSON(strings.NewReader(router.TokenMetaData.ABI))
	initP, _ := parsed.Pack("", factory, weth)
	return CreateLegacyTx(node, node.GetBuilder().Get(0), nil, 0, 0, big.NewInt(0), append(common.FromHex(ROUTER), initP...))
}
