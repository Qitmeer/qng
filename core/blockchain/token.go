package blockchain

import (
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/blockchain/token"
	"github.com/Qitmeer/qng/core/blockchain/utxo"
	"github.com/Qitmeer/qng/core/dbnamespace"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/database/legacydb"
	"github.com/Qitmeer/qng/meerdag"
)

func (b *BlockChain) CheckTokenTransactionInputs(tx *types.Tx, utxoView *utxo.UtxoViewpoint) error {
	msgTx := tx.Transaction()
	totalAtomIn := int64(0)
	targets := []uint{}

	for idx, txIn := range msgTx.TxIn {
		if idx == 0 {
			continue
		}
		utxoEntry := utxoView.LookupEntry(txIn.PreviousOut)
		if utxoEntry == nil || utxoEntry.IsSpent() {
			str := fmt.Sprintf("output %v referenced from "+
				"transaction %s:%d either does not exist or "+
				"has already been spent", txIn.PreviousOut,
				tx.Hash(), idx)
			return ruleError(ErrMissingTxOut, str)
		}
		if !utxoEntry.Amount().Id.IsBase() {
			return fmt.Errorf("Token transaction(%s) input (%s %d) must be MEERA\n", tx.Hash(), txIn.PreviousOut.Hash, txIn.PreviousOut.OutIndex)
		}

		originTxAtom := utxoEntry.Amount()
		if originTxAtom.Value < 0 {
			str := fmt.Sprintf("transaction output has negative "+
				"value of %v", originTxAtom)
			return ruleError(ErrInvalidTxOutValue, str)
		}
		if originTxAtom.Value > types.MaxAmount {
			str := fmt.Sprintf("transaction output value of %v is "+
				"higher than max allowed value of %v",
				originTxAtom, types.MaxAmount)
			return ruleError(ErrInvalidTxOutValue, str)
		}

		if utxoEntry.IsCoinBase() {
			ubhIB := b.bd.GetBlock(utxoEntry.BlockHash())
			if ubhIB == nil {
				str := fmt.Sprintf("utxoEntry blockhash error:%s", utxoEntry.BlockHash())
				return ruleError(ErrNoViewpoint, str)
			}
			targets = append(targets, ubhIB.GetID())
			if !utxoEntry.BlockHash().IsEqual(b.params.GenesisHash) {
				if originTxAtom.Id == types.MEERA {
					if txIn.PreviousOut.OutIndex == CoinbaseOutput_subsidy {
						originTxAtom.Value += b.GetFeeByCoinID(utxoEntry.BlockHash(), originTxAtom.Id)
					}
				} else {
					originTxAtom.Value = b.GetFeeByCoinID(utxoEntry.BlockHash(), originTxAtom.Id)
				}
			}
		}

		totalAtomIn += originTxAtom.Value
	}

	lockMeer := int64(dbnamespace.ByteOrder.Uint64(msgTx.TxIn[0].PreviousOut.Hash[0:8]))
	if totalAtomIn != lockMeer {
		return fmt.Errorf("Utxo (%d) and input amount (%d) are inconsistent\n", totalAtomIn, lockMeer)
	}

	//
	if len(targets) > 0 {
		viewpoints := []uint{}
		for _, blockHash := range utxoView.Viewpoints() {
			vIB := b.bd.GetBlock(blockHash)
			if vIB != nil {
				viewpoints = append(viewpoints, vIB.GetID())
			}
		}
		if len(viewpoints) == 0 {
			str := fmt.Sprintf("transaction %s has no viewpoints", tx.Hash())
			return ruleError(ErrNoViewpoint, str)
		}
		err := b.bd.CheckBlueAndMatureMT(targets, viewpoints, uint(b.params.CoinbaseMaturity))
		if err != nil {
			return ruleError(ErrImmatureSpend, err.Error())
		}
	}
	//

	totalAtomOut := int64(0)
	state := b.GetTokenState(b.TokenTipID)
	if state == nil {
		return fmt.Errorf("Token state error\n")
	}
	coinId := msgTx.TxOut[0].Amount.Id
	tt, ok := state.Types[coinId]
	if !ok {
		return fmt.Errorf("It doesn't exist: Coin id (%d)\n", coinId)
	}
	tokenAmount := int64(0)
	tb, ok := state.Balances[coinId]
	if ok {
		tokenAmount = tb.Balance
	}

	for idx, txOut := range tx.Transaction().TxOut {
		if txOut.Amount.Id != coinId {
			return fmt.Errorf("Transaction(%s) output(%d) coin id is invalid\n", tx.Hash(), idx)
		}
		totalAtomOut += txOut.Amount.Value
	}
	if totalAtomOut+tokenAmount > int64(tt.UpLimit) {
		return fmt.Errorf("Token transaction mint (%d) exceeds the maximum (%d)\n", totalAtomOut, tt.UpLimit)
	}

	return nil
}

func (b *BlockChain) updateTokenState(node meerdag.IBlock, block *types.SerializedBlock, rollback bool) error {
	if rollback {
		if uint32(node.GetID()) == b.TokenTipID {
			state := b.GetTokenState(b.TokenTipID)
			if state != nil {
				err := b.db.Update(func(dbTx legacydb.Tx) error {
					return token.DBRemoveTokenState(dbTx, uint32(node.GetID()))
				})
				if err != nil {
					return err
				}
				b.TokenTipID = state.PrevStateID
			}

		}
		return nil
	}
	updates := []token.ITokenUpdate{}
	for _, tx := range block.Transactions() {
		if tx.IsDuplicate {
			//log.Trace(fmt.Sprintf("updateTokenBalance skip duplicate tx %v", tx.Hash()))
			continue
		}

		if types.IsTokenTx(tx.Tx) {
			update, err := token.NewUpdateFromTx(tx.Tx)
			if err != nil {
				return err
			}
			updates = append(updates, update)
		}
	}
	if len(updates) <= 0 {
		return nil
	}
	state := b.GetTokenState(b.TokenTipID)
	if state == nil {
		state = &token.TokenState{PrevStateID: uint32(meerdag.MaxId), Updates: updates}
	} else {
		state.PrevStateID = b.TokenTipID
		state.Updates = updates
	}

	err := state.Update()
	if err != nil {
		return err
	}

	err = b.db.Update(func(dbTx legacydb.Tx) error {
		return token.DBPutTokenState(dbTx, uint32(node.GetID()), state)
	})
	if err != nil {
		return err
	}
	b.TokenTipID = uint32(node.GetID())
	return state.Commit()
}

func (b *BlockChain) GetTokenState(bid uint32) *token.TokenState {
	var state *token.TokenState
	err := b.db.View(func(dbTx legacydb.Tx) error {
		ts, err := token.DBFetchTokenState(dbTx, bid)
		if err != nil {
			return err
		}
		state = ts
		return nil
	})
	if err != nil {
		log.Error(err.Error())
		return nil
	}
	return state
}

func (b *BlockChain) GetCurTokenState() *token.TokenState {
	b.ChainRLock()
	defer b.ChainRUnlock()
	return b.GetTokenState(b.TokenTipID)
}

func (b *BlockChain) GetCurTokenOwners(coinId types.CoinID) ([]byte, error) {
	b.ChainRLock()
	defer b.ChainRUnlock()
	state := b.GetTokenState(b.TokenTipID)
	if state == nil {
		return nil, fmt.Errorf("Token state error\n")
	}
	tt, ok := state.Types[coinId]
	if !ok {
		return nil, fmt.Errorf("It doesn't exist: Coin id (%d)\n", coinId)
	}
	return tt.Owners, nil
}

func (b *BlockChain) CheckTokenState(block *types.SerializedBlock) error {
	updates := []token.ITokenUpdate{}
	for _, tx := range block.Transactions() {
		if tx.IsDuplicate {
			//log.Trace(fmt.Sprintf("updateTokenBalance skip duplicate tx %v", tx.Hash()))
			continue
		}

		if types.IsTokenTx(tx.Tx) {
			update, err := token.NewUpdateFromTx(tx.Tx)
			if err != nil {
				return err
			}
			updates = append(updates, update)
		}
	}
	if len(updates) <= 0 {
		return nil
	}
	state := b.GetTokenState(b.TokenTipID)
	if state == nil {
		state = &token.TokenState{PrevStateID: uint32(meerdag.MaxId), Updates: updates}
	} else {
		state.PrevStateID = b.TokenTipID
		state.Updates = updates
	}
	return state.Update()
}

func (b *BlockChain) GetTokenTipHash() *hash.Hash {
	if uint(b.TokenTipID) == meerdag.MaxId {
		return nil
	}
	ib := b.bd.GetBlockById(uint(b.TokenTipID))
	if ib == nil {
		return nil
	}
	return ib.GetHash()
}
