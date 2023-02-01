# MeerEVM

The MeerEVM is the Qtimeer's implementation of the Ethereum Virtual Machine (EVM), which supports Ethereum Smart Contract and most Ethereum client functionality. 

## QitSubnet
### How to open Qit subnet
```
~ ./qng -A=./ --qit
or
~ ./qng -A=./ --qit --qitenv="--http"
```

You can use RPC `./cli.sh qitinfo` to view the operation.

### How to package transaction submission blocks for signers ?
1. Import account into QNG node (Note: There are many ways to operate wallet accounts. Here, just one of them is listed for convenience)
```
~ ./qng --testnet -A=./ account import
or
~ ./qng --testnet -A=./ account new
```
2. Configure QNG startup parameters
```
~ ./qng -A=./ --testnet --qit --qitenv="--mine --miner.etherbase=[YourAddress] --unlock=[YourAddress] --password=./password"
```
Note: `./password"` The unlock password of address from keystore is located in the current directory.