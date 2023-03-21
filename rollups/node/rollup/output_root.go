package rollup

import (
	"errors"

	"github.com/Qitmeer/qng/rollups/bindings/bindings"
	"github.com/Qitmeer/qng/rollups/node/eth"
	"github.com/Qitmeer/qit/crypto"
)

var NilProof = errors.New("Output root proof is nil")

// ComputeL2OutputRoot computes the L2 output root by hashing an output root proof.
func ComputeL2OutputRoot(proofElements *bindings.TypesOutputRootProof) (eth.Bytes32, error) {
	if proofElements == nil {
		return eth.Bytes32{}, NilProof
	}

	digest := crypto.Keccak256Hash(
		proofElements.Version[:],
		proofElements.StateRoot[:],
		proofElements.MessagePasserStorageRoot[:],
		proofElements.LatestBlockhash[:],
	)
	return eth.Bytes32(digest), nil
}
