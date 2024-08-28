// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package release

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

// MeerMappingBurnDetail is an auto generated low-level Go binding around an user-defined struct.
type MeerMappingBurnDetail struct {
	Amount *big.Int
	Time   *big.Int
	Order  *big.Int
	Height *big.Int
}

// TokenMetaData contains all meta data concerning the Token contract.
var TokenMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"meerMappingAmounts\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"Amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"Time\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"Order\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"Height\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"name\":\"meerMappingCount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"_qngHash160\",\"type\":\"bytes\"}],\"name\":\"queryAmount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"_qngHash160\",\"type\":\"bytes\"}],\"name\":\"queryBurnDetails\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"Amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"Time\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"Order\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"Height\",\"type\":\"uint256\"}],\"internalType\":\"structMeerMapping.BurnDetail[]\",\"name\":\"\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// TokenABI is the input ABI used to generate the binding from.
// Deprecated: Use TokenMetaData.ABI instead.
var TokenABI = TokenMetaData.ABI

// Token is an auto generated Go binding around an Ethereum contract.
type Token struct {
	TokenCaller     // Read-only binding to the contract
	TokenTransactor // Write-only binding to the contract
	TokenFilterer   // Log filterer for contract events
}

// TokenCaller is an auto generated read-only Go binding around an Ethereum contract.
type TokenCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TokenTransactor is an auto generated write-only Go binding around an Ethereum contract.
type TokenTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TokenFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type TokenFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TokenSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type TokenSession struct {
	Contract     *Token            // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// TokenCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type TokenCallerSession struct {
	Contract *TokenCaller  // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// TokenTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type TokenTransactorSession struct {
	Contract     *TokenTransactor  // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// TokenRaw is an auto generated low-level Go binding around an Ethereum contract.
type TokenRaw struct {
	Contract *Token // Generic contract binding to access the raw methods on
}

// TokenCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type TokenCallerRaw struct {
	Contract *TokenCaller // Generic read-only contract binding to access the raw methods on
}

// TokenTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type TokenTransactorRaw struct {
	Contract *TokenTransactor // Generic write-only contract binding to access the raw methods on
}

// NewToken creates a new instance of Token, bound to a specific deployed contract.
func NewToken(address common.Address, backend bind.ContractBackend) (*Token, error) {
	contract, err := bindToken(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Token{TokenCaller: TokenCaller{contract: contract}, TokenTransactor: TokenTransactor{contract: contract}, TokenFilterer: TokenFilterer{contract: contract}}, nil
}

// NewTokenCaller creates a new read-only instance of Token, bound to a specific deployed contract.
func NewTokenCaller(address common.Address, caller bind.ContractCaller) (*TokenCaller, error) {
	contract, err := bindToken(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &TokenCaller{contract: contract}, nil
}

// NewTokenTransactor creates a new write-only instance of Token, bound to a specific deployed contract.
func NewTokenTransactor(address common.Address, transactor bind.ContractTransactor) (*TokenTransactor, error) {
	contract, err := bindToken(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &TokenTransactor{contract: contract}, nil
}

// NewTokenFilterer creates a new log filterer instance of Token, bound to a specific deployed contract.
func NewTokenFilterer(address common.Address, filterer bind.ContractFilterer) (*TokenFilterer, error) {
	contract, err := bindToken(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &TokenFilterer{contract: contract}, nil
}

// bindToken binds a generic wrapper to an already deployed contract.
func bindToken(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(TokenABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Token *TokenRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Token.Contract.TokenCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Token *TokenRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Token.Contract.TokenTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Token *TokenRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Token.Contract.TokenTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Token *TokenCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Token.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Token *TokenTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Token.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Token *TokenTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Token.Contract.contract.Transact(opts, method, params...)
}

// MeerMappingAmounts is a free data retrieval call binding the contract method 0xdac86377.
//
// Solidity: function meerMappingAmounts(bytes , uint256 ) view returns(uint256 Amount, uint256 Time, uint256 Order, uint256 Height)
func (_Token *TokenCaller) MeerMappingAmounts(opts *bind.CallOpts, arg0 []byte, arg1 *big.Int) (struct {
	Amount *big.Int
	Time   *big.Int
	Order  *big.Int
	Height *big.Int
}, error) {
	var out []interface{}
	err := _Token.contract.Call(opts, &out, "meerMappingAmounts", arg0, arg1)

	outstruct := new(struct {
		Amount *big.Int
		Time   *big.Int
		Order  *big.Int
		Height *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Amount = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.Time = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.Order = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.Height = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// MeerMappingAmounts is a free data retrieval call binding the contract method 0xdac86377.
//
// Solidity: function meerMappingAmounts(bytes , uint256 ) view returns(uint256 Amount, uint256 Time, uint256 Order, uint256 Height)
func (_Token *TokenSession) MeerMappingAmounts(arg0 []byte, arg1 *big.Int) (struct {
	Amount *big.Int
	Time   *big.Int
	Order  *big.Int
	Height *big.Int
}, error) {
	return _Token.Contract.MeerMappingAmounts(&_Token.CallOpts, arg0, arg1)
}

// MeerMappingAmounts is a free data retrieval call binding the contract method 0xdac86377.
//
// Solidity: function meerMappingAmounts(bytes , uint256 ) view returns(uint256 Amount, uint256 Time, uint256 Order, uint256 Height)
func (_Token *TokenCallerSession) MeerMappingAmounts(arg0 []byte, arg1 *big.Int) (struct {
	Amount *big.Int
	Time   *big.Int
	Order  *big.Int
	Height *big.Int
}, error) {
	return _Token.Contract.MeerMappingAmounts(&_Token.CallOpts, arg0, arg1)
}

// MeerMappingCount is a free data retrieval call binding the contract method 0xf96eb242.
//
// Solidity: function meerMappingCount(bytes ) view returns(uint256)
func (_Token *TokenCaller) MeerMappingCount(opts *bind.CallOpts, arg0 []byte) (*big.Int, error) {
	var out []interface{}
	err := _Token.contract.Call(opts, &out, "meerMappingCount", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MeerMappingCount is a free data retrieval call binding the contract method 0xf96eb242.
//
// Solidity: function meerMappingCount(bytes ) view returns(uint256)
func (_Token *TokenSession) MeerMappingCount(arg0 []byte) (*big.Int, error) {
	return _Token.Contract.MeerMappingCount(&_Token.CallOpts, arg0)
}

// MeerMappingCount is a free data retrieval call binding the contract method 0xf96eb242.
//
// Solidity: function meerMappingCount(bytes ) view returns(uint256)
func (_Token *TokenCallerSession) MeerMappingCount(arg0 []byte) (*big.Int, error) {
	return _Token.Contract.MeerMappingCount(&_Token.CallOpts, arg0)
}

// QueryAmount is a free data retrieval call binding the contract method 0xe944dec8.
//
// Solidity: function queryAmount(bytes _qngHash160) view returns(uint256)
func (_Token *TokenCaller) QueryAmount(opts *bind.CallOpts, _qngHash160 []byte) (*big.Int, error) {
	var out []interface{}
	err := _Token.contract.Call(opts, &out, "queryAmount", _qngHash160)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// QueryAmount is a free data retrieval call binding the contract method 0xe944dec8.
//
// Solidity: function queryAmount(bytes _qngHash160) view returns(uint256)
func (_Token *TokenSession) QueryAmount(_qngHash160 []byte) (*big.Int, error) {
	return _Token.Contract.QueryAmount(&_Token.CallOpts, _qngHash160)
}

// QueryAmount is a free data retrieval call binding the contract method 0xe944dec8.
//
// Solidity: function queryAmount(bytes _qngHash160) view returns(uint256)
func (_Token *TokenCallerSession) QueryAmount(_qngHash160 []byte) (*big.Int, error) {
	return _Token.Contract.QueryAmount(&_Token.CallOpts, _qngHash160)
}

// QueryBurnDetails is a free data retrieval call binding the contract method 0xcc53ccf3.
//
// Solidity: function queryBurnDetails(bytes _qngHash160) view returns((uint256,uint256,uint256,uint256)[])
func (_Token *TokenCaller) QueryBurnDetails(opts *bind.CallOpts, _qngHash160 []byte) ([]MeerMappingBurnDetail, error) {
	var out []interface{}
	err := _Token.contract.Call(opts, &out, "queryBurnDetails", _qngHash160)

	if err != nil {
		return *new([]MeerMappingBurnDetail), err
	}

	out0 := *abi.ConvertType(out[0], new([]MeerMappingBurnDetail)).(*[]MeerMappingBurnDetail)

	return out0, err

}

// QueryBurnDetails is a free data retrieval call binding the contract method 0xcc53ccf3.
//
// Solidity: function queryBurnDetails(bytes _qngHash160) view returns((uint256,uint256,uint256,uint256)[])
func (_Token *TokenSession) QueryBurnDetails(_qngHash160 []byte) ([]MeerMappingBurnDetail, error) {
	return _Token.Contract.QueryBurnDetails(&_Token.CallOpts, _qngHash160)
}

// QueryBurnDetails is a free data retrieval call binding the contract method 0xcc53ccf3.
//
// Solidity: function queryBurnDetails(bytes _qngHash160) view returns((uint256,uint256,uint256,uint256)[])
func (_Token *TokenCallerSession) QueryBurnDetails(_qngHash160 []byte) ([]MeerMappingBurnDetail, error) {
	return _Token.Contract.QueryBurnDetails(&_Token.CallOpts, _qngHash160)
}
