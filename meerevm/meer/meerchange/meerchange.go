// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package meerchange

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

// MeerchangeMetaData contains all meta data concerning the Meerchange contract.
var MeerchangeMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"txid\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint32\",\"name\":\"idx\",\"type\":\"uint32\"}],\"name\":\"Export\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"txid\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint32\",\"name\":\"idx\",\"type\":\"uint32\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"fee\",\"type\":\"uint64\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"sig\",\"type\":\"string\"}],\"name\":\"Export4337\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"txid\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"idx\",\"type\":\"uint32\"}],\"name\":\"export\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"txid\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"idx\",\"type\":\"uint32\"},{\"internalType\":\"uint64\",\"name\":\"fee\",\"type\":\"uint64\"},{\"internalType\":\"string\",\"name\":\"sig\",\"type\":\"string\"}],\"name\":\"export4337\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getExport\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getImport\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b506105b5806100206000396000f3fe608060405234801561001057600080fd5b506004361061004c5760003560e01c80634ccccead146100515780639801767c1461006d578063ac55902914610089578063bf2ec2e6146100a7575b600080fd5b61006b60048036038101906100669190610296565b6100c5565b005b6100876004803603810190610082919061037b565b61014e565b005b6100916101e0565b60405161009e9190610412565b60405180910390f35b6100af6101fd565b6040516100bc9190610412565b60405180910390f35b60008081819054906101000a900467ffffffffffffffff16809291906100ea9061045c565b91906101000a81548167ffffffffffffffff021916908367ffffffffffffffff160217905550507fa752968fe9a336e1e2de83b6e2f3bc1dd7d05dd3359dd92d9b92993209fffe3982826040516101429291906104aa565b60405180910390a15050565b60008081819054906101000a900467ffffffffffffffff16809291906101739061045c565b91906101000a81548167ffffffffffffffff021916908367ffffffffffffffff160217905550507fd0c588a715ffec02e91f35aa0907d659e0f19d01b5eb5a4229e2d43f448edc6385858585856040516101d1959493929190610531565b60405180910390a15050505050565b60008060089054906101000a900467ffffffffffffffff16905090565b60008060009054906101000a900467ffffffffffffffff16905090565b600080fd5b600080fd5b6000819050919050565b61023781610224565b811461024257600080fd5b50565b6000813590506102548161022e565b92915050565b600063ffffffff82169050919050565b6102738161025a565b811461027e57600080fd5b50565b6000813590506102908161026a565b92915050565b600080604083850312156102ad576102ac61021a565b5b60006102bb85828601610245565b92505060206102cc85828601610281565b9150509250929050565b600067ffffffffffffffff82169050919050565b6102f3816102d6565b81146102fe57600080fd5b50565b600081359050610310816102ea565b92915050565b600080fd5b600080fd5b600080fd5b60008083601f84011261033b5761033a610316565b5b8235905067ffffffffffffffff8111156103585761035761031b565b5b60208301915083600182028301111561037457610373610320565b5b9250929050565b6000806000806000608086880312156103975761039661021a565b5b60006103a588828901610245565b95505060206103b688828901610281565b94505060406103c788828901610301565b935050606086013567ffffffffffffffff8111156103e8576103e761021f565b5b6103f488828901610325565b92509250509295509295909350565b61040c816102d6565b82525050565b60006020820190506104276000830184610403565b92915050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b6000610467826102d6565b915067ffffffffffffffff82036104815761048061042d565b5b600182019050919050565b61049581610224565b82525050565b6104a48161025a565b82525050565b60006040820190506104bf600083018561048c565b6104cc602083018461049b565b9392505050565b600082825260208201905092915050565b82818337600083830152505050565b6000601f19601f8301169050919050565b600061051083856104d3565b935061051d8385846104e4565b610526836104f3565b840190509392505050565b6000608082019050610546600083018861048c565b610553602083018761049b565b6105606040830186610403565b8181036060830152610573818486610504565b9050969550505050505056fea26469706673582212209197861206fb8a023d18a77d58d0ee47b50a3ef8b0175a8744e7599dfe5c23a264736f6c634300080d0033",
}

// MeerchangeABI is the input ABI used to generate the binding from.
// Deprecated: Use MeerchangeMetaData.ABI instead.
var MeerchangeABI = MeerchangeMetaData.ABI

// MeerchangeBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use MeerchangeMetaData.Bin instead.
var MeerchangeBin = MeerchangeMetaData.Bin

// DeployMeerchange deploys a new Ethereum contract, binding an instance of Meerchange to it.
func DeployMeerchange(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Meerchange, error) {
	parsed, err := MeerchangeMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(MeerchangeBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Meerchange{MeerchangeCaller: MeerchangeCaller{contract: contract}, MeerchangeTransactor: MeerchangeTransactor{contract: contract}, MeerchangeFilterer: MeerchangeFilterer{contract: contract}}, nil
}

// Meerchange is an auto generated Go binding around an Ethereum contract.
type Meerchange struct {
	MeerchangeCaller     // Read-only binding to the contract
	MeerchangeTransactor // Write-only binding to the contract
	MeerchangeFilterer   // Log filterer for contract events
}

// MeerchangeCaller is an auto generated read-only Go binding around an Ethereum contract.
type MeerchangeCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MeerchangeTransactor is an auto generated write-only Go binding around an Ethereum contract.
type MeerchangeTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MeerchangeFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type MeerchangeFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MeerchangeSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type MeerchangeSession struct {
	Contract     *Meerchange       // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// MeerchangeCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type MeerchangeCallerSession struct {
	Contract *MeerchangeCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts     // Call options to use throughout this session
}

// MeerchangeTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type MeerchangeTransactorSession struct {
	Contract     *MeerchangeTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// MeerchangeRaw is an auto generated low-level Go binding around an Ethereum contract.
type MeerchangeRaw struct {
	Contract *Meerchange // Generic contract binding to access the raw methods on
}

// MeerchangeCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type MeerchangeCallerRaw struct {
	Contract *MeerchangeCaller // Generic read-only contract binding to access the raw methods on
}

// MeerchangeTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type MeerchangeTransactorRaw struct {
	Contract *MeerchangeTransactor // Generic write-only contract binding to access the raw methods on
}

// NewMeerchange creates a new instance of Meerchange, bound to a specific deployed contract.
func NewMeerchange(address common.Address, backend bind.ContractBackend) (*Meerchange, error) {
	contract, err := bindMeerchange(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Meerchange{MeerchangeCaller: MeerchangeCaller{contract: contract}, MeerchangeTransactor: MeerchangeTransactor{contract: contract}, MeerchangeFilterer: MeerchangeFilterer{contract: contract}}, nil
}

// NewMeerchangeCaller creates a new read-only instance of Meerchange, bound to a specific deployed contract.
func NewMeerchangeCaller(address common.Address, caller bind.ContractCaller) (*MeerchangeCaller, error) {
	contract, err := bindMeerchange(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &MeerchangeCaller{contract: contract}, nil
}

// NewMeerchangeTransactor creates a new write-only instance of Meerchange, bound to a specific deployed contract.
func NewMeerchangeTransactor(address common.Address, transactor bind.ContractTransactor) (*MeerchangeTransactor, error) {
	contract, err := bindMeerchange(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &MeerchangeTransactor{contract: contract}, nil
}

// NewMeerchangeFilterer creates a new log filterer instance of Meerchange, bound to a specific deployed contract.
func NewMeerchangeFilterer(address common.Address, filterer bind.ContractFilterer) (*MeerchangeFilterer, error) {
	contract, err := bindMeerchange(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &MeerchangeFilterer{contract: contract}, nil
}

// bindMeerchange binds a generic wrapper to an already deployed contract.
func bindMeerchange(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(MeerchangeABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Meerchange *MeerchangeRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Meerchange.Contract.MeerchangeCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Meerchange *MeerchangeRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Meerchange.Contract.MeerchangeTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Meerchange *MeerchangeRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Meerchange.Contract.MeerchangeTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Meerchange *MeerchangeCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Meerchange.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Meerchange *MeerchangeTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Meerchange.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Meerchange *MeerchangeTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Meerchange.Contract.contract.Transact(opts, method, params...)
}

// GetExport is a free data retrieval call binding the contract method 0xbf2ec2e6.
//
// Solidity: function getExport() view returns(uint64)
func (_Meerchange *MeerchangeCaller) GetExport(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _Meerchange.contract.Call(opts, &out, "getExport")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// GetExport is a free data retrieval call binding the contract method 0xbf2ec2e6.
//
// Solidity: function getExport() view returns(uint64)
func (_Meerchange *MeerchangeSession) GetExport() (uint64, error) {
	return _Meerchange.Contract.GetExport(&_Meerchange.CallOpts)
}

// GetExport is a free data retrieval call binding the contract method 0xbf2ec2e6.
//
// Solidity: function getExport() view returns(uint64)
func (_Meerchange *MeerchangeCallerSession) GetExport() (uint64, error) {
	return _Meerchange.Contract.GetExport(&_Meerchange.CallOpts)
}

// GetImport is a free data retrieval call binding the contract method 0xac559029.
//
// Solidity: function getImport() view returns(uint64)
func (_Meerchange *MeerchangeCaller) GetImport(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _Meerchange.contract.Call(opts, &out, "getImport")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// GetImport is a free data retrieval call binding the contract method 0xac559029.
//
// Solidity: function getImport() view returns(uint64)
func (_Meerchange *MeerchangeSession) GetImport() (uint64, error) {
	return _Meerchange.Contract.GetImport(&_Meerchange.CallOpts)
}

// GetImport is a free data retrieval call binding the contract method 0xac559029.
//
// Solidity: function getImport() view returns(uint64)
func (_Meerchange *MeerchangeCallerSession) GetImport() (uint64, error) {
	return _Meerchange.Contract.GetImport(&_Meerchange.CallOpts)
}

// Export is a paid mutator transaction binding the contract method 0x4ccccead.
//
// Solidity: function export(bytes32 txid, uint32 idx) returns()
func (_Meerchange *MeerchangeTransactor) Export(opts *bind.TransactOpts, txid [32]byte, idx uint32) (*types.Transaction, error) {
	return _Meerchange.contract.Transact(opts, "export", txid, idx)
}

// Export is a paid mutator transaction binding the contract method 0x4ccccead.
//
// Solidity: function export(bytes32 txid, uint32 idx) returns()
func (_Meerchange *MeerchangeSession) Export(txid [32]byte, idx uint32) (*types.Transaction, error) {
	return _Meerchange.Contract.Export(&_Meerchange.TransactOpts, txid, idx)
}

// Export is a paid mutator transaction binding the contract method 0x4ccccead.
//
// Solidity: function export(bytes32 txid, uint32 idx) returns()
func (_Meerchange *MeerchangeTransactorSession) Export(txid [32]byte, idx uint32) (*types.Transaction, error) {
	return _Meerchange.Contract.Export(&_Meerchange.TransactOpts, txid, idx)
}

// Export4337 is a paid mutator transaction binding the contract method 0x9801767c.
//
// Solidity: function export4337(bytes32 txid, uint32 idx, uint64 fee, string sig) returns()
func (_Meerchange *MeerchangeTransactor) Export4337(opts *bind.TransactOpts, txid [32]byte, idx uint32, fee uint64, sig string) (*types.Transaction, error) {
	return _Meerchange.contract.Transact(opts, "export4337", txid, idx, fee, sig)
}

// Export4337 is a paid mutator transaction binding the contract method 0x9801767c.
//
// Solidity: function export4337(bytes32 txid, uint32 idx, uint64 fee, string sig) returns()
func (_Meerchange *MeerchangeSession) Export4337(txid [32]byte, idx uint32, fee uint64, sig string) (*types.Transaction, error) {
	return _Meerchange.Contract.Export4337(&_Meerchange.TransactOpts, txid, idx, fee, sig)
}

// Export4337 is a paid mutator transaction binding the contract method 0x9801767c.
//
// Solidity: function export4337(bytes32 txid, uint32 idx, uint64 fee, string sig) returns()
func (_Meerchange *MeerchangeTransactorSession) Export4337(txid [32]byte, idx uint32, fee uint64, sig string) (*types.Transaction, error) {
	return _Meerchange.Contract.Export4337(&_Meerchange.TransactOpts, txid, idx, fee, sig)
}

// MeerchangeExportIterator is returned from FilterExport and is used to iterate over the raw logs and unpacked data for Export events raised by the Meerchange contract.
type MeerchangeExportIterator struct {
	Event *MeerchangeExport // Event containing the contract specifics and raw log

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
func (it *MeerchangeExportIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MeerchangeExport)
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
		it.Event = new(MeerchangeExport)
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
func (it *MeerchangeExportIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MeerchangeExportIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MeerchangeExport represents a Export event raised by the Meerchange contract.
type MeerchangeExport struct {
	Txid [32]byte
	Idx  uint32
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterExport is a free log retrieval operation binding the contract event 0xa752968fe9a336e1e2de83b6e2f3bc1dd7d05dd3359dd92d9b92993209fffe39.
//
// Solidity: event Export(bytes32 txid, uint32 idx)
func (_Meerchange *MeerchangeFilterer) FilterExport(opts *bind.FilterOpts) (*MeerchangeExportIterator, error) {

	logs, sub, err := _Meerchange.contract.FilterLogs(opts, "Export")
	if err != nil {
		return nil, err
	}
	return &MeerchangeExportIterator{contract: _Meerchange.contract, event: "Export", logs: logs, sub: sub}, nil
}

// WatchExport is a free log subscription operation binding the contract event 0xa752968fe9a336e1e2de83b6e2f3bc1dd7d05dd3359dd92d9b92993209fffe39.
//
// Solidity: event Export(bytes32 txid, uint32 idx)
func (_Meerchange *MeerchangeFilterer) WatchExport(opts *bind.WatchOpts, sink chan<- *MeerchangeExport) (event.Subscription, error) {

	logs, sub, err := _Meerchange.contract.WatchLogs(opts, "Export")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MeerchangeExport)
				if err := _Meerchange.contract.UnpackLog(event, "Export", log); err != nil {
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

// ParseExport is a log parse operation binding the contract event 0xa752968fe9a336e1e2de83b6e2f3bc1dd7d05dd3359dd92d9b92993209fffe39.
//
// Solidity: event Export(bytes32 txid, uint32 idx)
func (_Meerchange *MeerchangeFilterer) ParseExport(log types.Log) (*MeerchangeExport, error) {
	event := new(MeerchangeExport)
	if err := _Meerchange.contract.UnpackLog(event, "Export", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// MeerchangeExport4337Iterator is returned from FilterExport4337 and is used to iterate over the raw logs and unpacked data for Export4337 events raised by the Meerchange contract.
type MeerchangeExport4337Iterator struct {
	Event *MeerchangeExport4337 // Event containing the contract specifics and raw log

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
func (it *MeerchangeExport4337Iterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MeerchangeExport4337)
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
		it.Event = new(MeerchangeExport4337)
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
func (it *MeerchangeExport4337Iterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MeerchangeExport4337Iterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MeerchangeExport4337 represents a Export4337 event raised by the Meerchange contract.
type MeerchangeExport4337 struct {
	Txid [32]byte
	Idx  uint32
	Fee  uint64
	Sig  string
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterExport4337 is a free log retrieval operation binding the contract event 0xd0c588a715ffec02e91f35aa0907d659e0f19d01b5eb5a4229e2d43f448edc63.
//
// Solidity: event Export4337(bytes32 txid, uint32 idx, uint64 fee, string sig)
func (_Meerchange *MeerchangeFilterer) FilterExport4337(opts *bind.FilterOpts) (*MeerchangeExport4337Iterator, error) {

	logs, sub, err := _Meerchange.contract.FilterLogs(opts, "Export4337")
	if err != nil {
		return nil, err
	}
	return &MeerchangeExport4337Iterator{contract: _Meerchange.contract, event: "Export4337", logs: logs, sub: sub}, nil
}

// WatchExport4337 is a free log subscription operation binding the contract event 0xd0c588a715ffec02e91f35aa0907d659e0f19d01b5eb5a4229e2d43f448edc63.
//
// Solidity: event Export4337(bytes32 txid, uint32 idx, uint64 fee, string sig)
func (_Meerchange *MeerchangeFilterer) WatchExport4337(opts *bind.WatchOpts, sink chan<- *MeerchangeExport4337) (event.Subscription, error) {

	logs, sub, err := _Meerchange.contract.WatchLogs(opts, "Export4337")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MeerchangeExport4337)
				if err := _Meerchange.contract.UnpackLog(event, "Export4337", log); err != nil {
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

// ParseExport4337 is a log parse operation binding the contract event 0xd0c588a715ffec02e91f35aa0907d659e0f19d01b5eb5a4229e2d43f448edc63.
//
// Solidity: event Export4337(bytes32 txid, uint32 idx, uint64 fee, string sig)
func (_Meerchange *MeerchangeFilterer) ParseExport4337(log types.Log) (*MeerchangeExport4337, error) {
	event := new(MeerchangeExport4337)
	if err := _Meerchange.contract.UnpackLog(event, "Export4337", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
