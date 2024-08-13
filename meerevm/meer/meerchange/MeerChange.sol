//SPDX-License-Identifier: Unlicense
pragma solidity ^0.8.0;

contract MeerChange {
    uint64 private exportCount;
    uint64 private importCount;

    event Export(
        bytes32 txid,
        uint32 idx
    ); 

    function export(bytes32 txid,uint32 idx) public {
        exportCount++;
        emit Export(txid,idx);
    }

    function getExport() public view returns (uint64) {
        return exportCount;
    }

    function getImport() public view returns (uint64) {
        return importCount;
    }
}
