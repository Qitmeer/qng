//SPDX-License-Identifier: Unlicense
pragma solidity ^0.8.0;

contract MeerChange {
    uint64 private exportCount;
    uint64 private importCount;

    event Export(
        bytes32 txid,
        uint32 idx
    ); 

    event Export4337(
        bytes32 txid,
        uint32 idx,
        uint64 fee,
        string sig
    ); 

    function export(bytes32 txid,uint32 idx) public {
        exportCount++;
        emit Export(txid,idx);
    }

    function export4337(bytes32 txid,uint32 idx,uint64 fee,string calldata sig) public {
        exportCount++;
        emit Export4337(txid,idx,fee,sig);
    }

    function getExport() public view returns (uint64) {
        return exportCount;
    }

    function getImport() public view returns (uint64) {
        return importCount;
    }
}
