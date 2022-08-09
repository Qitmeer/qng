// SPDX-License-Identifier: MIT
pragma solidity ^0.8.3;

import "./tool.sol";
import "./blake2b.sol";

contract MeerRelease is BLAKE2b{
    // var
    uint256 public startTime; // release start time index 0
    uint256 public endTime; // release end time index 1

    // all lockUsers mapping Amounts
    mapping(bytes => uint256) public meerLockAmounts; // index 2

    struct MeerLockUser {
        uint256 amount;
        address addr;
        uint256 lastReleaseTime;
        uint256 releaseAmount;
        uint256 releasePerSec;
    }

    // all lockUsers
    mapping(address =>MeerLockUser) public meerLockUsers;

    // events
    event Deposit(address indexed _from, uint256 _value);
    event Claim(address indexed _user, uint256 _value);
    event Lock(address indexed _user,uint256 endTime, uint256 _value);


    function toBytes(bytes32 _data) public pure returns (bytes memory) {
        return abi.encodePacked(_data);
    }

    // contract can receive meer
    receive() external payable {
        emit Deposit(msg.sender, msg.value);
    }

    // lock user amount
    function lock(address _user ,uint256 _value) internal {
        require(endTime > startTime && startTime >0 && endTime > 0,"time not set");
        if(meerLockUsers[_user].amount <= 0){
            // first lock
            meerLockUsers[_user].addr = _user;
            meerLockUsers[_user].releaseAmount = 0;
            meerLockUsers[_user].lastReleaseTime = startTime;
        }
        meerLockUsers[_user].amount += _value;
        meerLockUsers[_user].releasePerSec = meerLockUsers[_user].amount / (endTime - meerLockUsers[_user].lastReleaseTime);
        emit Lock(_user,endTime, _value);
    }

    // query release amount
    function canRelease(address _user) public view returns(uint256){
        uint256 leftAmount = meerLockUsers[_user].amount - meerLockUsers[_user].releaseAmount;
        if(leftAmount <= 0) {
            return 0;
        }
        if(block.timestamp <= meerLockUsers[_user].lastReleaseTime){
            return 0;
        }
        uint256 canReleaseAmount = ( block.timestamp - meerLockUsers[_user].lastReleaseTime ) * meerLockUsers[_user].releasePerSec;
        if(canReleaseAmount > leftAmount){
            canReleaseAmount = leftAmount;
        }
        return canReleaseAmount;
    }

    // claim meer
    function claim(bytes memory _pubkey) external payable returns(uint256){
        address _user = CheckBitcoinSigs.accountFromPubkey(_pubkey);
        bytes32 h = blake2b_256(_pubkey);
        bytes memory _qngHash160 =  CheckBitcoinSigs.p2wpkhFromPubkey(toBytes(h));
        if(meerLockUsers[_user].amount <= 0 && meerLockAmounts[_qngHash160] > 0){
            // first lock
            lock(_user,meerLockAmounts[_qngHash160]);
        }
        uint256 amount = canRelease(_user);
        if (amount <= 0){
            return 0;
        }

        meerLockUsers[_user].releaseAmount += amount;
        meerLockUsers[_user].lastReleaseTime = block.timestamp;
        payable(_user).transfer(amount);
        emit Claim(_user, amount);
        return amount;
    }
}