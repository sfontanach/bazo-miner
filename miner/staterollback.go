package miner

import (
	"github.com/sfontanach/bazo-miner/protocol"
	"github.com/sfontanach/bazo-miner/storage"
)

func accStateChangeRollback(txSlice []*protocol.AccTx) {
	for _, tx := range txSlice {
		if tx.Header == 0 || tx.Header == 1 || tx.Header == 2 {
			accHash := protocol.SerializeHashContent(tx.PubKey)
			acc := storage.State[accHash]
			if acc == nil {
				logger.Fatal("CRITICAL: An account that should have been saved does not exist!")
			}

			delete(storage.State, accHash)

			switch tx.Header {
			case 1:
				delete(storage.RootKeys, accHash)
			case 2:
				storage.RootKeys[accHash] = acc
			}
		}
	}
}

func fundsStateChangeRollback(txSlice []*protocol.FundsTx) {
	//Rollback in reverse order than original state change
	for cnt := len(txSlice) - 1; cnt >= 0; cnt-- {
		tx := txSlice[cnt]

		accSender, accReceiver := storage.GetAccount(tx.From), storage.GetAccount(tx.To)
		accSender.TxCnt -= 1
		accSender.Balance += tx.Amount
		accReceiver.Balance -= tx.Amount

		//If new coins were issued, revert
		if rootAcc := storage.RootKeys[tx.From]; rootAcc != nil {
			rootAcc.Balance -= tx.Amount
			rootAcc.Balance -= tx.Fee
		}
	}
}

func configStateChangeRollback(txSlice []*protocol.ConfigTx, blockHash [32]byte) {
	if len(txSlice) == 0 {
		return
	}

	//Only rollback if the config changes lead to a parameterChange
	//there might be the case that the client is not running the latest version, it's still confirming
	//the transaction but does not understand the ID and thus is not changing the state
	if parameterSlice[len(parameterSlice)-1].BlockHash != blockHash {
		return
	}

	//remove the latest entry in the parameters slice$
	parameterSlice = parameterSlice[:len(parameterSlice)-1]
	activeParameters = &parameterSlice[len(parameterSlice)-1]
	logger.Printf("Config parameters rolled back. New configuration: %v", *activeParameters)
}

func stakeStateChangeRollback(txSlice []*protocol.StakeTx) {
	//Rollback in reverse order than original state change
	for cnt := len(txSlice) - 1; cnt >= 0; cnt-- {
		tx := txSlice[cnt]
		accSender := storage.GetAccount(tx.Account)
		accSender.IsStaking = !accSender.IsStaking
	}
}

func collectTxFeesRollback(accTx []*protocol.AccTx, fundsTx []*protocol.FundsTx, configTx []*protocol.ConfigTx, stakeTx []*protocol.StakeTx, minerHash [32]byte) {
	minerAcc := storage.GetAccount(minerHash)
	//subtract fees from sender (check if that is allowed has already been done in the block validation)
	for _, tx := range accTx {
		//Money was created out of thin air, no need to write back
		minerAcc.Balance -= tx.Fee
	}

	for _, tx := range fundsTx {
		minerAcc.Balance -= tx.Fee
		senderAcc := storage.GetAccount(tx.From)
		senderAcc.Balance += tx.Fee
	}

	for _, tx := range configTx {
		//Money was created out of thin air, no need to write back
		minerAcc.Balance -= tx.Fee
	}

	for _, tx := range stakeTx {
		minerAcc.Balance -= tx.Fee
		senderAcc := storage.GetAccount(tx.Account)
		senderAcc.Balance += tx.Fee
	}
}

func collectBlockRewardRollback(reward uint64, minerHash [32]byte) {
	minerAcc := storage.GetAccount(minerHash)
	minerAcc.Balance -= reward
}
