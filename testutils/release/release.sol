// SPDX-License-Identifier: MIT
pragma solidity ^0.8.3;

contract MeerRelease {
    // var
    uint256 public startTime; // release start time
    uint256 public endTime; // release end time
    address public owner; // contract owner

    struct MeerLockUser {
        address addr;
        uint256 amount;
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

    constructor() {
    }

    modifier onlyOwner() {
        require(msg.sender == owner, 'not owner!!!');
        _;
    }

    // contract can receive meer
    receive() external payable {
        emit Deposit(msg.sender, msg.value);
    }
    // set release start time
    function setStartTime(uint256 _start) external onlyOwner {
        startTime = _start;
    }
    // set release end time
    function setEndTime(uint256 _end) external onlyOwner {
        endTime = _end;
    }

    // lock user amount
    function lock(address _user ,uint256 _value) external onlyOwner {
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
    function claim(address _user) external payable returns(uint256){
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