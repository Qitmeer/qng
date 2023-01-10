package consensus

import "github.com/ethereum/go-ethereum"


const QitJs = `
web3._extend({
	property: 'qit',
	methods: [
		new web3._extend.Method({
			name: 'getSnapshot',
			call: 'qit_getSnapshot',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter]
		}),
		new web3._extend.Method({
			name: 'getSnapshotAtHash',
			call: 'qit_getSnapshotAtHash',
			params: 1
		}),
		new web3._extend.Method({
			name: 'getSigners',
			call: 'qit_getSigners',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter]
		}),
		new web3._extend.Method({
			name: 'getSignersAtHash',
			call: 'qit_getSignersAtHash',
			params: 1
		}),
		new web3._extend.Method({
			name: 'propose',
			call: 'qit_propose',
			params: 2
		}),
		new web3._extend.Method({
			name: 'discard',
			call: 'qit_discard',
			params: 1
		}),
		new web3._extend.Method({
			name: 'status',
			call: 'qit_status',
			params: 0
		}),
		new web3._extend.Method({
			name: 'getSigner',
			call: 'qit_getSigner',
			params: 1,
			inputFormatter: [null]
		}),
	],
	properties: [
		new web3._extend.Property({
			name: 'proposals',
			getter: 'qit_proposals'
		}),
	]
});
`

func init() {
	ethereum.Modules["qit"] = QitJs
}
