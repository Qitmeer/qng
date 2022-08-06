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

// TokenMetaData contains all meta data concerning the Token contract.
var TokenMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"_user\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"Claim\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"_from\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"Deposit\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"_user\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"endTime\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"Lock\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_user\",\"type\":\"address\"}],\"name\":\"canRelease\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_user\",\"type\":\"address\"}],\"name\":\"claim\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"endTime\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_user\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"lock\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"meerLockUsers\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lastReleaseTime\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"releaseAmount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"releasePerSec\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_end\",\"type\":\"uint256\"}],\"name\":\"setEndTime\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_start\",\"type\":\"uint256\"}],\"name\":\"setStartTime\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"startTime\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"stateMutability\":\"payable\",\"type\":\"receive\"}]",
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

// CanRelease is a free data retrieval call binding the contract method 0x6dfe789a.
//
// Solidity: function canRelease(address _user) view returns(uint256)
func (_Token *TokenCaller) CanRelease(opts *bind.CallOpts, _user common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Token.contract.Call(opts, &out, "canRelease", _user)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// CanRelease is a free data retrieval call binding the contract method 0x6dfe789a.
//
// Solidity: function canRelease(address _user) view returns(uint256)
func (_Token *TokenSession) CanRelease(_user common.Address) (*big.Int, error) {
	return _Token.Contract.CanRelease(&_Token.CallOpts, _user)
}

// CanRelease is a free data retrieval call binding the contract method 0x6dfe789a.
//
// Solidity: function canRelease(address _user) view returns(uint256)
func (_Token *TokenCallerSession) CanRelease(_user common.Address) (*big.Int, error) {
	return _Token.Contract.CanRelease(&_Token.CallOpts, _user)
}

// EndTime is a free data retrieval call binding the contract method 0x3197cbb6.
//
// Solidity: function endTime() view returns(uint256)
func (_Token *TokenCaller) EndTime(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Token.contract.Call(opts, &out, "endTime")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// EndTime is a free data retrieval call binding the contract method 0x3197cbb6.
//
// Solidity: function endTime() view returns(uint256)
func (_Token *TokenSession) EndTime() (*big.Int, error) {
	return _Token.Contract.EndTime(&_Token.CallOpts)
}

// EndTime is a free data retrieval call binding the contract method 0x3197cbb6.
//
// Solidity: function endTime() view returns(uint256)
func (_Token *TokenCallerSession) EndTime() (*big.Int, error) {
	return _Token.Contract.EndTime(&_Token.CallOpts)
}

// MeerLockUsers is a free data retrieval call binding the contract method 0xd113d4ed.
//
// Solidity: function meerLockUsers(address ) view returns(address addr, uint256 amount, uint256 lastReleaseTime, uint256 releaseAmount, uint256 releasePerSec)
func (_Token *TokenCaller) MeerLockUsers(opts *bind.CallOpts, arg0 common.Address) (struct {
	Addr            common.Address
	Amount          *big.Int
	LastReleaseTime *big.Int
	ReleaseAmount   *big.Int
	ReleasePerSec   *big.Int
}, error) {
	var out []interface{}
	err := _Token.contract.Call(opts, &out, "meerLockUsers", arg0)

	outstruct := new(struct {
		Addr            common.Address
		Amount          *big.Int
		LastReleaseTime *big.Int
		ReleaseAmount   *big.Int
		ReleasePerSec   *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Addr = *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	outstruct.Amount = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.LastReleaseTime = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.ReleaseAmount = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)
	outstruct.ReleasePerSec = *abi.ConvertType(out[4], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// MeerLockUsers is a free data retrieval call binding the contract method 0xd113d4ed.
//
// Solidity: function meerLockUsers(address ) view returns(address addr, uint256 amount, uint256 lastReleaseTime, uint256 releaseAmount, uint256 releasePerSec)
func (_Token *TokenSession) MeerLockUsers(arg0 common.Address) (struct {
	Addr            common.Address
	Amount          *big.Int
	LastReleaseTime *big.Int
	ReleaseAmount   *big.Int
	ReleasePerSec   *big.Int
}, error) {
	return _Token.Contract.MeerLockUsers(&_Token.CallOpts, arg0)
}

// MeerLockUsers is a free data retrieval call binding the contract method 0xd113d4ed.
//
// Solidity: function meerLockUsers(address ) view returns(address addr, uint256 amount, uint256 lastReleaseTime, uint256 releaseAmount, uint256 releasePerSec)
func (_Token *TokenCallerSession) MeerLockUsers(arg0 common.Address) (struct {
	Addr            common.Address
	Amount          *big.Int
	LastReleaseTime *big.Int
	ReleaseAmount   *big.Int
	ReleasePerSec   *big.Int
}, error) {
	return _Token.Contract.MeerLockUsers(&_Token.CallOpts, arg0)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Token *TokenCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Token.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Token *TokenSession) Owner() (common.Address, error) {
	return _Token.Contract.Owner(&_Token.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Token *TokenCallerSession) Owner() (common.Address, error) {
	return _Token.Contract.Owner(&_Token.CallOpts)
}

// StartTime is a free data retrieval call binding the contract method 0x78e97925.
//
// Solidity: function startTime() view returns(uint256)
func (_Token *TokenCaller) StartTime(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Token.contract.Call(opts, &out, "startTime")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// StartTime is a free data retrieval call binding the contract method 0x78e97925.
//
// Solidity: function startTime() view returns(uint256)
func (_Token *TokenSession) StartTime() (*big.Int, error) {
	return _Token.Contract.StartTime(&_Token.CallOpts)
}

// StartTime is a free data retrieval call binding the contract method 0x78e97925.
//
// Solidity: function startTime() view returns(uint256)
func (_Token *TokenCallerSession) StartTime() (*big.Int, error) {
	return _Token.Contract.StartTime(&_Token.CallOpts)
}

// Claim is a paid mutator transaction binding the contract method 0x1e83409a.
//
// Solidity: function claim(address _user) payable returns(uint256)
func (_Token *TokenTransactor) Claim(opts *bind.TransactOpts, _user common.Address) (*types.Transaction, error) {
	return _Token.contract.Transact(opts, "claim", _user)
}

// Claim is a paid mutator transaction binding the contract method 0x1e83409a.
//
// Solidity: function claim(address _user) payable returns(uint256)
func (_Token *TokenSession) Claim(_user common.Address) (*types.Transaction, error) {
	return _Token.Contract.Claim(&_Token.TransactOpts, _user)
}

// Claim is a paid mutator transaction binding the contract method 0x1e83409a.
//
// Solidity: function claim(address _user) payable returns(uint256)
func (_Token *TokenTransactorSession) Claim(_user common.Address) (*types.Transaction, error) {
	return _Token.Contract.Claim(&_Token.TransactOpts, _user)
}

// Lock is a paid mutator transaction binding the contract method 0x282d3fdf.
//
// Solidity: function lock(address _user, uint256 _value) returns()
func (_Token *TokenTransactor) Lock(opts *bind.TransactOpts, _user common.Address, _value *big.Int) (*types.Transaction, error) {
	return _Token.contract.Transact(opts, "lock", _user, _value)
}

// Lock is a paid mutator transaction binding the contract method 0x282d3fdf.
//
// Solidity: function lock(address _user, uint256 _value) returns()
func (_Token *TokenSession) Lock(_user common.Address, _value *big.Int) (*types.Transaction, error) {
	return _Token.Contract.Lock(&_Token.TransactOpts, _user, _value)
}

// Lock is a paid mutator transaction binding the contract method 0x282d3fdf.
//
// Solidity: function lock(address _user, uint256 _value) returns()
func (_Token *TokenTransactorSession) Lock(_user common.Address, _value *big.Int) (*types.Transaction, error) {
	return _Token.Contract.Lock(&_Token.TransactOpts, _user, _value)
}

// SetEndTime is a paid mutator transaction binding the contract method 0xccb98ffc.
//
// Solidity: function setEndTime(uint256 _end) returns()
func (_Token *TokenTransactor) SetEndTime(opts *bind.TransactOpts, _end *big.Int) (*types.Transaction, error) {
	return _Token.contract.Transact(opts, "setEndTime", _end)
}

// SetEndTime is a paid mutator transaction binding the contract method 0xccb98ffc.
//
// Solidity: function setEndTime(uint256 _end) returns()
func (_Token *TokenSession) SetEndTime(_end *big.Int) (*types.Transaction, error) {
	return _Token.Contract.SetEndTime(&_Token.TransactOpts, _end)
}

// SetEndTime is a paid mutator transaction binding the contract method 0xccb98ffc.
//
// Solidity: function setEndTime(uint256 _end) returns()
func (_Token *TokenTransactorSession) SetEndTime(_end *big.Int) (*types.Transaction, error) {
	return _Token.Contract.SetEndTime(&_Token.TransactOpts, _end)
}

// SetStartTime is a paid mutator transaction binding the contract method 0x3e0a322d.
//
// Solidity: function setStartTime(uint256 _start) returns()
func (_Token *TokenTransactor) SetStartTime(opts *bind.TransactOpts, _start *big.Int) (*types.Transaction, error) {
	return _Token.contract.Transact(opts, "setStartTime", _start)
}

// SetStartTime is a paid mutator transaction binding the contract method 0x3e0a322d.
//
// Solidity: function setStartTime(uint256 _start) returns()
func (_Token *TokenSession) SetStartTime(_start *big.Int) (*types.Transaction, error) {
	return _Token.Contract.SetStartTime(&_Token.TransactOpts, _start)
}

// SetStartTime is a paid mutator transaction binding the contract method 0x3e0a322d.
//
// Solidity: function setStartTime(uint256 _start) returns()
func (_Token *TokenTransactorSession) SetStartTime(_start *big.Int) (*types.Transaction, error) {
	return _Token.Contract.SetStartTime(&_Token.TransactOpts, _start)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_Token *TokenTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Token.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_Token *TokenSession) Receive() (*types.Transaction, error) {
	return _Token.Contract.Receive(&_Token.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_Token *TokenTransactorSession) Receive() (*types.Transaction, error) {
	return _Token.Contract.Receive(&_Token.TransactOpts)
}

// TokenClaimIterator is returned from FilterClaim and is used to iterate over the raw logs and unpacked data for Claim events raised by the Token contract.
type TokenClaimIterator struct {
	Event *TokenClaim // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *TokenClaimIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TokenClaim)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(TokenClaim)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *TokenClaimIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TokenClaimIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TokenClaim represents a Claim event raised by the Token contract.
type TokenClaim struct {
	User  common.Address
	Value *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterClaim is a free log retrieval operation binding the contract event 0x47cee97cb7acd717b3c0aa1435d004cd5b3c8c57d70dbceb4e4458bbd60e39d4.
//
// Solidity: event Claim(address indexed _user, uint256 _value)
func (_Token *TokenFilterer) FilterClaim(opts *bind.FilterOpts, _user []common.Address) (*TokenClaimIterator, error) {

	var _userRule []interface{}
	for _, _userItem := range _user {
		_userRule = append(_userRule, _userItem)
	}

	logs, sub, err := _Token.contract.FilterLogs(opts, "Claim", _userRule)
	if err != nil {
		return nil, err
	}
	return &TokenClaimIterator{contract: _Token.contract, event: "Claim", logs: logs, sub: sub}, nil
}

// WatchClaim is a free log subscription operation binding the contract event 0x47cee97cb7acd717b3c0aa1435d004cd5b3c8c57d70dbceb4e4458bbd60e39d4.
//
// Solidity: event Claim(address indexed _user, uint256 _value)
func (_Token *TokenFilterer) WatchClaim(opts *bind.WatchOpts, sink chan<- *TokenClaim, _user []common.Address) (event.Subscription, error) {

	var _userRule []interface{}
	for _, _userItem := range _user {
		_userRule = append(_userRule, _userItem)
	}

	logs, sub, err := _Token.contract.WatchLogs(opts, "Claim", _userRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TokenClaim)
				if err := _Token.contract.UnpackLog(event, "Claim", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseClaim is a log parse operation binding the contract event 0x47cee97cb7acd717b3c0aa1435d004cd5b3c8c57d70dbceb4e4458bbd60e39d4.
//
// Solidity: event Claim(address indexed _user, uint256 _value)
func (_Token *TokenFilterer) ParseClaim(log types.Log) (*TokenClaim, error) {
	event := new(TokenClaim)
	if err := _Token.contract.UnpackLog(event, "Claim", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TokenDepositIterator is returned from FilterDeposit and is used to iterate over the raw logs and unpacked data for Deposit events raised by the Token contract.
type TokenDepositIterator struct {
	Event *TokenDeposit // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *TokenDepositIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TokenDeposit)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(TokenDeposit)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *TokenDepositIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TokenDepositIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TokenDeposit represents a Deposit event raised by the Token contract.
type TokenDeposit struct {
	From  common.Address
	Value *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterDeposit is a free log retrieval operation binding the contract event 0xe1fffcc4923d04b559f4d29a8bfc6cda04eb5b0d3c460751c2402c5c5cc9109c.
//
// Solidity: event Deposit(address indexed _from, uint256 _value)
func (_Token *TokenFilterer) FilterDeposit(opts *bind.FilterOpts, _from []common.Address) (*TokenDepositIterator, error) {

	var _fromRule []interface{}
	for _, _fromItem := range _from {
		_fromRule = append(_fromRule, _fromItem)
	}

	logs, sub, err := _Token.contract.FilterLogs(opts, "Deposit", _fromRule)
	if err != nil {
		return nil, err
	}
	return &TokenDepositIterator{contract: _Token.contract, event: "Deposit", logs: logs, sub: sub}, nil
}

// WatchDeposit is a free log subscription operation binding the contract event 0xe1fffcc4923d04b559f4d29a8bfc6cda04eb5b0d3c460751c2402c5c5cc9109c.
//
// Solidity: event Deposit(address indexed _from, uint256 _value)
func (_Token *TokenFilterer) WatchDeposit(opts *bind.WatchOpts, sink chan<- *TokenDeposit, _from []common.Address) (event.Subscription, error) {

	var _fromRule []interface{}
	for _, _fromItem := range _from {
		_fromRule = append(_fromRule, _fromItem)
	}

	logs, sub, err := _Token.contract.WatchLogs(opts, "Deposit", _fromRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TokenDeposit)
				if err := _Token.contract.UnpackLog(event, "Deposit", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseDeposit is a log parse operation binding the contract event 0xe1fffcc4923d04b559f4d29a8bfc6cda04eb5b0d3c460751c2402c5c5cc9109c.
//
// Solidity: event Deposit(address indexed _from, uint256 _value)
func (_Token *TokenFilterer) ParseDeposit(log types.Log) (*TokenDeposit, error) {
	event := new(TokenDeposit)
	if err := _Token.contract.UnpackLog(event, "Deposit", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TokenLockIterator is returned from FilterLock and is used to iterate over the raw logs and unpacked data for Lock events raised by the Token contract.
type TokenLockIterator struct {
	Event *TokenLock // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *TokenLockIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TokenLock)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(TokenLock)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *TokenLockIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TokenLockIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TokenLock represents a Lock event raised by the Token contract.
type TokenLock struct {
	User    common.Address
	EndTime *big.Int
	Value   *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterLock is a free log retrieval operation binding the contract event 0x49eaf4942f1237055eb4cfa5f31c9dfe50d5b4ade01e021f7de8be2fbbde557b.
//
// Solidity: event Lock(address indexed _user, uint256 endTime, uint256 _value)
func (_Token *TokenFilterer) FilterLock(opts *bind.FilterOpts, _user []common.Address) (*TokenLockIterator, error) {

	var _userRule []interface{}
	for _, _userItem := range _user {
		_userRule = append(_userRule, _userItem)
	}

	logs, sub, err := _Token.contract.FilterLogs(opts, "Lock", _userRule)
	if err != nil {
		return nil, err
	}
	return &TokenLockIterator{contract: _Token.contract, event: "Lock", logs: logs, sub: sub}, nil
}

// WatchLock is a free log subscription operation binding the contract event 0x49eaf4942f1237055eb4cfa5f31c9dfe50d5b4ade01e021f7de8be2fbbde557b.
//
// Solidity: event Lock(address indexed _user, uint256 endTime, uint256 _value)
func (_Token *TokenFilterer) WatchLock(opts *bind.WatchOpts, sink chan<- *TokenLock, _user []common.Address) (event.Subscription, error) {

	var _userRule []interface{}
	for _, _userItem := range _user {
		_userRule = append(_userRule, _userItem)
	}

	logs, sub, err := _Token.contract.WatchLogs(opts, "Lock", _userRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TokenLock)
				if err := _Token.contract.UnpackLog(event, "Lock", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseLock is a log parse operation binding the contract event 0x49eaf4942f1237055eb4cfa5f31c9dfe50d5b4ade01e021f7de8be2fbbde557b.
//
// Solidity: event Lock(address indexed _user, uint256 endTime, uint256 _value)
func (_Token *TokenFilterer) ParseLock(log types.Log) (*TokenLock, error) {
	event := new(TokenLock)
	if err := _Token.contract.UnpackLog(event, "Lock", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
