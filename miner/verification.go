package miner

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"github.com/sfontanach/bazo-miner/protocol"
	"github.com/sfontanach/bazo-miner/storage"
	"math/big"
	"reflect"
)

//We can't use polymorphism, e.g. we can't use tx.verify() because the Transaction interface doesn't declare
//the verify method. This is because verification depends on the State (e.g., dynamic properties), which
//should only be of concern to the miner, not to the protocol package. However, this has the disadvantage
//that we have to do case distinction here.
func verify(tx protocol.Transaction) bool {
	var verified bool

	switch tx.(type) {
	case *protocol.FundsTx:
		verified = verifyFundsTx(tx.(*protocol.FundsTx))
	case *protocol.AccTx:
		verified = verifyAccTx(tx.(*protocol.AccTx))
	case *protocol.ConfigTx:
		verified = verifyConfigTx(tx.(*protocol.ConfigTx))
	case *protocol.StakeTx:
		verified = verifyStakeTx(tx.(*protocol.StakeTx))
	}

	return verified
}

func verifyFundsTx(tx *protocol.FundsTx) bool {
	if tx == nil {
		return false
	}

	pubKey1Sig1, pubKey2Sig1 := new(big.Int), new(big.Int)
	r, s := new(big.Int), new(big.Int)

	//fundsTx only makes sense if amount > 0
	if tx.Amount == 0 || tx.Amount > MAX_MONEY {
		logger.Printf("Invalid transaction amount %v\n", tx.Amount)
		return false
	}

	//Check if accounts are present in the actual state
	accFrom := storage.State[tx.From]
	accTo := storage.State[tx.To]

	//Accounts non existent
	if accFrom == nil || accTo == nil {
		logger.Printf("Account non existent. From: %v\nTo: %v\n", accFrom, accTo)
		return false
	}

	accFromHash := protocol.SerializeHashContent(accFrom.Address)
	accToHash := protocol.SerializeHashContent(accTo.Address)

	pubKey1Sig1.SetBytes(accFrom.Address[:32])
	pubKey2Sig1.SetBytes(accFrom.Address[32:])

	r.SetBytes(tx.Sig1[:32])
	s.SetBytes(tx.Sig1[32:])

	tx.From = accFromHash
	tx.To = accToHash

	txHash := tx.Hash()

	var validSig1, validSig2 bool

	pubKey := ecdsa.PublicKey{elliptic.P256(), pubKey1Sig1, pubKey2Sig1}
	if ecdsa.Verify(&pubKey, txHash[:], r, s) && !reflect.DeepEqual(accFrom, accTo) {
		tx.From = accFromHash
		tx.To = accToHash
		validSig1 = true
	} else {
		//	TODO error logging
	}

	r.SetBytes(tx.Sig2[:32])
	s.SetBytes(tx.Sig2[32:])

	if ecdsa.Verify(multisigPubKey, txHash[:], r, s) {
		validSig2 = true
	} else {
		//	TODO error logging
	}

	return validSig1 && validSig2
}

func verifyAccTx(tx *protocol.AccTx) bool {
	if tx == nil {
		return false
	}

	r, s := new(big.Int), new(big.Int)
	pub1, pub2 := new(big.Int), new(big.Int)

	r.SetBytes(tx.Sig[:32])
	s.SetBytes(tx.Sig[32:])

	for _, rootAcc := range storage.RootKeys {
		pub1.SetBytes(rootAcc.Address[:32])
		pub2.SetBytes(rootAcc.Address[32:])

		pubKey := ecdsa.PublicKey{elliptic.P256(), pub1, pub2}
		txHash := tx.Hash()

		//Only the hash of the pubkey is hashed and verified here
		if ecdsa.Verify(&pubKey, txHash[:], r, s) == true {
			return true
		}
	}

	return false
}

func verifyConfigTx(tx *protocol.ConfigTx) bool {
	if tx == nil {
		return false
	}

	//account creation can only be done with a valid priv/pub key which is hard-coded
	r, s := new(big.Int), new(big.Int)
	pub1, pub2 := new(big.Int), new(big.Int)

	r.SetBytes(tx.Sig[:32])
	s.SetBytes(tx.Sig[32:])

	for _, rootAcc := range storage.RootKeys {
		pub1.SetBytes(rootAcc.Address[:32])
		pub2.SetBytes(rootAcc.Address[32:])

		pubKey := ecdsa.PublicKey{elliptic.P256(), pub1, pub2}
		txHash := tx.Hash()
		if ecdsa.Verify(&pubKey, txHash[:], r, s) == true {
			return true
		}
	}

	return false
}

func verifyStakeTx(tx *protocol.StakeTx) bool {
	if tx == nil {
		logger.Println("Transactions does not exist.")
		return false
	}

	pub1, pub2 := new(big.Int), new(big.Int)
	r, s := new(big.Int), new(big.Int)

	//Check if account is present in the actual state
	accFrom := storage.State[tx.Account]

	//Account non existent
	if accFrom == nil {
		logger.Println("Account does not exist.")
		return false
	}

	accFromHash := protocol.SerializeHashContent(accFrom.Address)

	pub1.SetBytes(accFrom.Address[:32])
	pub2.SetBytes(accFrom.Address[32:])

	r.SetBytes(tx.Sig[:32])
	s.SetBytes(tx.Sig[32:])

	tx.Account = accFromHash

	txHash := tx.Hash()

	pubKey := ecdsa.PublicKey{elliptic.P256(), pub1, pub2}

	if ecdsa.Verify(&pubKey, txHash[:], r, s) == true {
		tx.Account = accFromHash
		return true
	}

	return false
}

//Returns true if id is in the list of possible ids and rational value for payload parameter.
//Some values just don't make any sense and have to be restricted accordingly
func parameterBoundsChecking(id uint8, payload uint64) bool {
	switch id {
	case protocol.BLOCK_SIZE_ID:
		if payload >= protocol.MIN_BLOCK_SIZE && payload <= protocol.MAX_BLOCK_SIZE {
			return true
		}
	case protocol.DIFF_INTERVAL_ID:
		if payload >= protocol.MIN_DIFF_INTERVAL && payload <= protocol.MAX_DIFF_INTERVAL {
			return true
		}
	case protocol.FEE_MINIMUM_ID:
		if payload >= protocol.MIN_FEE_MINIMUM && payload <= protocol.MAX_FEE_MINIMUM {
			return true
		}
	case protocol.BLOCK_INTERVAL_ID:
		if payload >= protocol.MIN_BLOCK_INTERVAL && payload <= protocol.MAX_BLOCK_INTERVAL {
			return true
		}
	case protocol.BLOCK_REWARD_ID:
		if payload >= protocol.MIN_BLOCK_REWARD && payload <= protocol.MAX_BLOCK_REWARD {
			return true
		}
	case protocol.STAKING_MINIMUM_ID:
		if payload >= protocol.MIN_STAKING_MINIMUM && payload <= protocol.MAX_STAKING_MINIMUM {
			return true
		}
	case protocol.WAITING_MINIMUM_ID:
		if payload >= protocol.MIN_WAITING_TIME && payload <= protocol.MAX_WAITING_TIME {
			return true
		}
	case protocol.ACCEPTANCE_TIME_DIFF_ID:
		if payload >= protocol.MIN_ACCEPTANCE_TIME_DIFF && payload <= protocol.MAX_ACCEPTANCE_TIME_DIFF {
			return true
		}
	case protocol.SLASHING_WINDOW_SIZE_ID:
		if payload >= protocol.MIN_SLASHING_WINDOW_SIZE && payload <= protocol.MAX_SLASHING_WINDOW_SIZE {
			return true
		}
	case protocol.SLASHING_REWARD_ID:
		if payload >= protocol.MIN_SLASHING_REWARD && payload <= protocol.MAX_SLASHING_REWARD {
			return true
		}
	}

	return false
}
