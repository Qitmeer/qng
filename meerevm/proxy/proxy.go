package proxy

import (
	"context"
	"errors"
	"fmt"
	"github.com/Qitmeer/qng/log"
	qparams "github.com/Qitmeer/qng/params"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"math/big"
	"time"
)

// from: https://github.com/Arachnid/deterministic-deployment-proxy
const (
	DeterministicDeploymentProxyAddr     = "4e59b44847b379578588920ca78fbf26c0b4956c"
	DeterministicDeploymentProxyTx       = "0xf8a58085174876e800830186a08080b853604580600e600039806000f350fe7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe03601600081602082378035828234f58015156039578182fd5b8082525050506014600cf31ba02222222222222222222222222222222222222222222222222222222222222222a02222222222222222222222222222222222222222222222222222222222222222"
	DeterministicDeploymentProxyDeployer = "0x3fab184622dc19b6109349b94811493bf2a45362"
	DeterministicDeploymentProxyGasPrice = 100e9
	DeterministicDeploymentProxyGasLimit = 100000
	DeterministicDeploymentProxyFee      = DeterministicDeploymentProxyGasPrice * DeterministicDeploymentProxyGasLimit
)

type DeterministicDeploymentProxy struct {
	ctx      context.Context
	rpc      *ethclient.Client
	bytecode []byte
}

func (ddp DeterministicDeploymentProxy) GetAddress() *common.Address {
	addr := common.HexToAddress(DeterministicDeploymentProxyAddr)
	return &addr
}

func (ddp DeterministicDeploymentProxy) GetContractAddress(owner common.Address, bytecode []byte, version int64) (common.Address, error) {
	if ddp.GetAddress().Cmp(common.Address{}) == 0 {
		return common.Address{}, errors.New("No support DeterministicDeploymentProxy")
	}
	msg := ethereum.CallMsg{
		From: owner,
		To:   ddp.GetAddress(),
		Data: ddp.GetContractDeployData(bytecode, version),
	}
	addrBytes, err := ddp.rpc.CallContract(ddp.ctx, msg, nil)
	if err != nil {
		return common.Address{}, err
	}
	return common.BytesToAddress(addrBytes), nil
}

func (ddp DeterministicDeploymentProxy) DeployContract(owner common.Address, bytecode []byte, version int64, value *big.Int, gas uint64) (common.Hash, error) {
	if ddp.GetAddress().Cmp(common.Address{}) == 0 {
		return common.Hash{}, errors.New("No support DeterministicDeploymentProxy")
	}
	arg := map[string]interface{}{
		"from":  owner,
		"to":    ddp.GetAddress(),
		"input": hexutil.Bytes(ddp.GetContractDeployData(bytecode, version)),
	}
	if gas != 0 {
		arg["gas"] = hexutil.Uint64(gas)
	}
	if value != nil {
		arg["value"] = (*hexutil.Big)(value)
	}
	var hashBytes hexutil.Bytes
	err := ddp.rpc.Client().CallContext(ddp.ctx, &hashBytes, "eth_sendTransaction", arg)
	if err != nil {
		return common.Hash{}, err
	}
	return common.BytesToHash(hashBytes), nil
}

func (ddp DeterministicDeploymentProxy) GetContractDeployData(bytecode []byte, version int64) []byte {
	salt := common.BigToHash(big.NewInt(version))
	data := []byte{}
	data = append(salt.Bytes(), bytecode...)
	return data
}

func (ddp DeterministicDeploymentProxy) GetCode() ([]byte, error) {
	bytecode, err := ddp.rpc.CodeAt(ddp.ctx, *ddp.GetAddress(), nil)
	if err != nil {
		return nil, err
	}
	ddp.bytecode = bytecode
	return bytecode, nil
}

func (ddp DeterministicDeploymentProxy) CheckDeploy() (bool, []byte) {
	bytecode, err := ddp.GetCode()
	if err != nil {
		log.Warn(err.Error())
		return false, nil
	}
	if len(bytecode) <= 0 {
		return false, nil
	}
	return true, bytecode
}

func (ddp DeterministicDeploymentProxy) IsDeployed() bool {
	ret, _ := ddp.CheckDeploy()
	return ret
}

func (ddp DeterministicDeploymentProxy) Deploy(owner common.Address) error {
	if ddp.IsDeployed() {
		log.Info("It has already been deployed, so ignore this operation")
		return nil
	}
	arg := map[string]interface{}{
		"from":  owner,
		"to":    common.HexToAddress(DeterministicDeploymentProxyDeployer),
		"value": (*hexutil.Big)(big.NewInt(DeterministicDeploymentProxyFee)),
	}

	var hashBytes hexutil.Bytes
	err := ddp.rpc.Client().CallContext(ddp.ctx, &hashBytes, "eth_sendTransaction", arg)
	if err != nil {
		return err
	}
	txHash := common.BytesToHash(hashBytes)
	log.Info("Send fee to deterministic deployment proxy deployer", "tx", txHash.String())

	err = ddp.waitTx(txHash)
	if err != nil {
		return err
	}
	hashBytes = hexutil.Bytes{}
	err = ddp.rpc.Client().CallContext(ddp.ctx, &hashBytes, "eth_sendRawTransaction", DeterministicDeploymentProxyTx)
	if err != nil {
		return err
	}
	return ddp.waitTx(common.BytesToHash(hashBytes))
}

func (ddp DeterministicDeploymentProxy) waitTx(txh common.Hash) error {
	ctx, cancel := context.WithTimeout(ddp.ctx, qparams.ActiveNetParams.TargetTimePerBlock*300)
	defer cancel()
	for {
		select {
		case <-time.After(time.Second):
			tx, isPending, err := ddp.rpc.TransactionByHash(ddp.ctx, txh)
			if err != nil {
				return err
			}
			if isPending {
				log.Info("Deterministic deployment proxy waiting...", "tx", txh.String())
				continue
			}
			if tx == nil {
				return fmt.Errorf("Proxy Confirmation tx failed:%s", txh.String())
			}
			log.Info("Deterministic deployment proxy finished", "tx", txh.String())
			return nil
		case <-ctx.Done():
			return fmt.Errorf("Proxy Confirmation tx failed:%s", txh.String())
		}
	}
}

func NewDeterministicDeploymentProxy(ctx context.Context, rpc *ethclient.Client) *DeterministicDeploymentProxy {
	return &DeterministicDeploymentProxy{
		ctx: ctx,
		rpc: rpc,
	}
}
