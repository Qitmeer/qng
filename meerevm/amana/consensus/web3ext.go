package consensus

import "github.com/ethereum/go-ethereum"


const AmanaJs = `
web3._extend({
	property: 'amana',
	methods: [
		new web3._extend.Method({
			name: 'getSnapshot',
			call: 'amana_getSnapshot',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter]
		}),
		new web3._extend.Method({
			name: 'getSnapshotAtHash',
			call: 'amana_getSnapshotAtHash',
			params: 1
		}),
		new web3._extend.Method({
			name: 'getSigners',
			call: 'amana_getSigners',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter]
		}),
		new web3._extend.Method({
			name: 'getSignersAtHash',
			call: 'amana_getSignersAtHash',
			params: 1
		}),
		new web3._extend.Method({
			name: 'propose',
			call: 'amana_propose',
			params: 2
		}),
		new web3._extend.Method({
			name: 'discard',
			call: 'amana_discard',
			params: 1
		}),
		new web3._extend.Method({
			name: 'status',
			call: 'amana_status',
			params: 0
		}),
		new web3._extend.Method({
			name: 'getSigner',
			call: 'amana_getSigner',
			params: 1,
			inputFormatter: [null]
		}),
	],
	properties: [
		new web3._extend.Property({
			name: 'proposals',
			getter: 'amana_proposals'
		}),
	]
});
`

func init() {
	ethereum.Modules["amana"] = AmanaJs
}
