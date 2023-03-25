package meer

import "github.com/ethereum/go-ethereum"

const QngJs = `
web3._extend({
	property: 'qng',
	methods: [
		new web3._extend.Method({
			name: 'getPeerInfo',
			call: 'qng_getPeerInfo',
			params: 2,
			inputFormatter: [null, null]
		}),

		new web3._extend.Method({
			name: 'removeBan',
			call: 'qng_removeBan',
			params: 1,
			inputFormatter: [null]
		}),
		new web3._extend.Method({
			name: 'setRpcMaxClients',
			call: 'qng_setRpcMaxClients',
			params: 1,
		}),
		new web3._extend.Method({
			name: 'setLogLevel',
			call: 'qng_setLogLevel',
			params: 1,
		}),

		new web3._extend.Method({
			name: 'checkAddress',
			call: 'qng_checkAddress',
			params: 2,
		}),
		new web3._extend.Method({
			name: 'getBalance',
			call: 'qng_getBalance',
			params: 2,
		}),
		new web3._extend.Method({
			name: 'getAddresses',
			call: 'qng_getAddresses',
			params: 1,
		}),

		new web3._extend.Method({
			name: 'getBlockhash',
			call: 'qng_getBlockhash',
			params: 1,
		}),
		new web3._extend.Method({
			name: 'getBlockhashByRange',
			call: 'qng_getBlockhashByRange',
			params: 2,
		}),
		new web3._extend.Method({
			name: 'getBlockByOrder',
			call: 'qng_getBlockByOrder',
			params: 4,
			inputFormatter: [null, null, null, null]
		}),
		new web3._extend.Method({
			name: 'getBlock',
			call: 'qng_getBlock',
			params: 4,
			inputFormatter: [null, null, null, null]
		}),
		new web3._extend.Method({
			name: 'getBlockV2',
			call: 'qng_getBlockV2',
			params: 4,
			inputFormatter: [null, null, null, null]
		}),
		new web3._extend.Method({
			name: 'getBlockHeader',
			call: 'qng_getBlockHeader',
			params: 2,
		}),
		new web3._extend.Method({
			name: 'isOnMainChain',
			call: 'qng_isOnMainChain',
			params: 1,
		}),
		new web3._extend.Method({
			name: 'getBlockWeight',
			call: 'qng_getBlockWeight',
			params: 1,
		}),
		new web3._extend.Method({
			name: 'getBlockByID',
			call: 'qng_getBlockByID',
			params: 4,
			inputFormatter: [null, null, null, null]
		}),
		new web3._extend.Method({
			name: 'getBlockByNum',
			call: 'qng_getBlockByNum',
			params: 4,
			inputFormatter: [null, null, null, null]
		}),
		new web3._extend.Method({
			name: 'isBlue',
			call: 'qng_isBlue',
			params: 1,
		}),
		new web3._extend.Method({
			name: 'getCoinbase',
			call: 'qng_getCoinbase',
			params: 2,
			inputFormatter: [null, null]
		}),
		new web3._extend.Method({
			name: 'getFees',
			call: 'qng_getFees',
			params: 1,
		}),

		new web3._extend.Method({
			name: 'getMempool',
			call: 'qng_getMempool',
			params: 2,
			inputFormatter: [null, null]
		}),
		new web3._extend.Method({
			name: 'estimateFee',
			call: 'qng_estimateFee',
			params: 1,
		}),

		new web3._extend.Method({
			name: 'getBlockTemplate',
			call: 'qng_getBlockTemplate',
			params: 2,
		}),
		new web3._extend.Method({
			name: 'submitBlock',
			call: 'qng_submitBlock',
			params: 1,
		}),
		new web3._extend.Method({
			name: 'getRemoteGBT',
			call: 'qng_getRemoteGBT',
			params: 1,
		}),
		new web3._extend.Method({
			name: 'submitBlockHeader',
			call: 'qng_submitBlockHeader',
			params: 1,
		}),
		new web3._extend.Method({
			name: 'generate',
			call: 'qng_generate',
			params: 1,
		}),

		new web3._extend.Method({
			name: 'createRawTransaction',
			call: 'qng_createRawTransaction',
			params: 3,
		}),
		new web3._extend.Method({
			name: 'createRawTransactionV2',
			call: 'qng_createRawTransactionV2',
			params: 3,
		}),
		new web3._extend.Method({
			name: 'decodeRawTransaction',
			call: 'qng_decodeRawTransaction',
			params: 1,
		}),
		new web3._extend.Method({
			name: 'sendRawTransaction',
			call: 'qng_sendRawTransaction',
			params: 2,
		}),
		new web3._extend.Method({
			name: 'getRawTransaction',
			call: 'qng_getRawTransaction',
			params: 2,
		}),
		new web3._extend.Method({
			name: 'getUtxo',
			call: 'qng_getUtxo',
			params: 3,
		}),
		new web3._extend.Method({
			name: 'getRawTransactions',
			call: 'qng_getRawTransactions',
			params: 7,
		}),
		new web3._extend.Method({
			name: 'getRawTransactionByHash',
			call: 'qng_getRawTransactionByHash',
			params: 2,
		}),
		new web3._extend.Method({
			name: 'txSign',
			call: 'qng_txSign',
			params: 3,
		}),
		new web3._extend.Method({
			name: 'createTokenRawTransaction',
			call: 'qng_createTokenRawTransaction',
			params: 9,
		}),
		new web3._extend.Method({
			name: 'createImportRawTransaction',
			call: 'qng_createImportRawTransaction',
			params: 2,
		}),
		new web3._extend.Method({
			name: 'createExportRawTransaction',
			call: 'qng_createExportRawTransaction',
			params: 4,
		}),
		new web3._extend.Method({
			name: 'createExportRawTransactionV2',
			call: 'qng_createExportRawTransactionV2',
			params: 3,
		}),
	],
	properties: [
		new web3._extend.Property({
			name: 'getNodeInfo',
			getter: 'qng_getNodeInfo'
		}),
		new web3._extend.Property({
			name: 'getRpcInfo',
			getter: 'qng_getRpcInfo'
		}),
		new web3._extend.Property({
			name: 'getTimeInfo',
			getter: 'qng_getTimeInfo'
		}),
		new web3._extend.Property({
			name: 'getNetworkInfo',
			getter: 'qng_getNetworkInfo'
		}),
		new web3._extend.Property({
			name: 'getSubsidy',
			getter: 'qng_getSubsidy'
		}),
		new web3._extend.Property({
			name: 'stop',
			getter: 'qng_stop'
		}),
		new web3._extend.Property({
			name: 'banlist',
			getter: 'qng_banlist'
		}),

		new web3._extend.Property({
			name: 'getBestBlockHash',
			getter: 'qng_getBestBlockHash'
		}),
		new web3._extend.Property({
			name: 'getBlockCount',
			getter: 'qng_getBlockCount'
		}),
		new web3._extend.Property({
			name: 'getBlockTotal',
			getter: 'qng_getBlockTotal'
		}),
		new web3._extend.Property({
			name: 'getMainChainHeight',
			getter: 'qng_getMainChainHeight'
		}),
		new web3._extend.Property({
			name: 'getOrphansTotal',
			getter: 'qng_getOrphansTotal'
		}),
		new web3._extend.Property({
			name: 'isCurrent',
			getter: 'qng_isCurrent'
		}),
		new web3._extend.Property({
			name: 'tips',
			getter: 'qng_tips'
		}),
		new web3._extend.Property({
			name: 'getTokenInfo',
			getter: 'qng_getTokenInfo'
		}),

		new web3._extend.Property({
			name: 'getMempoolCount',
			getter: 'qng_getMempoolCount'
		}),
		new web3._extend.Property({
			name: 'saveMempool',
			getter: 'qng_saveMempool'
		}),

		new web3._extend.Property({
			name: 'getMinerInfo',
			getter: 'qng_getMinerInfo'
		}),

		new web3._extend.Property({
			name: 'getVMsInfo',
			getter: 'qng_getVMsInfo'
		}),

		new web3._extend.Property({
			name: 'getAmanaNodeInfo',
			getter: 'qng_getAmanaNodeInfo'
		}),

		new web3._extend.Property({
			name: 'getAmanaPeerInfo',
			getter: 'qng_getAmanaPeerInfo'
		}),
	]
});
`

func init() {
	ethereum.Modules["qng"] = QngJs
}
