# qng
The next generation of the Qitmeer network implementation with the plug-able VMs under the MeerDAG consensus.

### Installation
* Build from source
```bash
~ git clone https://github.com/Qitmeer/qng.git
~ make
```

or
* Install the latest qng available here:
https://github.com/Qitmeer/qng/releases 


### Getting Started
* We take the construction of test network nodes as an example:
```
~ cd ./build/bin
~ ./qng --testnet
~ 
``` 

### Miner

* If you are a miner, you also need to configure your reward address:
```
~ ./qng --testnet --miningaddr=Tk6uXJ3kjh3yA4q94KQF9DTL14rDbd4vb2kztbkfhMBziR35HYkkx 
``` 

*  Please note that the mining address here is a PK address:
```
~ ./qx ec-to-public [Your_Private_Key] | ./qx ec-to-pkaddr -v=testnet
``` 
*  If you use the old address(`PKH Address`), you will only be unable to package the cross chain transaction.

### MeerEVM
* If you want to use our MeerEVM function, the required interface information can be queried in this RPC:
```
~ cd ./script
~ ./cli.sh vmsinfo
``` 
* If you don't need the default configuration, we provide an environment configuration parameter to meet your custom configuration for MeerEVM:
```
~ ./qng --testnet --evmenv="--http"
or
~ ./qng --testnet --evmenv="--http --http.port=18545 --ws --ws.port=18546"
~ 
``` 


* You first need to transfer your money in qitmeer to MeerEVM:`createExportRawTx <txid> <vout> <PKAdress> <amount>`
``` 
~ ./cli.sh createExportRawTx ce28ec92cc99b13d9f7a658d2f1e08aa9e4f27ebcfaf5344750bb77484a79657 0 Tk6uXJ3kjh3yA4q94KQF9DTL14rDbd4vb2kztbkfhMBziR35HYkkx 11000000000
~ ./cli.sh txSign [Your_Private_Key] [rawTx]
~ ./cli.sh sendRawTx [signRawTx]
``` 
* Finally, wait for the miner to pack your transaction into the block. Then you have the money to start operating your MeerEVM ecosystem.


### How can I transfer my money in meerevm to the qitmeer account system ?
```
~ ./cli.sh createImportRawTx Tk6uXJ3kjh3yA4q94KQF9DTL14rDbd4vb2kztbkfhMBziR35HYkkx [amount] 
~ ./cli.sh txSign [Your_Private_Key] [rawTx]
~ ./cli.sh sendRawTx [signRawTx]
``` 
* Finally, wait for the miner to pack your transaction into the block. 