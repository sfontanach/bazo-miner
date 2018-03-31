package main

import (
	"github.com/sfontanach/bazo-miner/miner"
	"github.com/sfontanach/bazo-miner/p2p"
	"github.com/sfontanach/bazo-miner/storage"
	"os"
)

func main() {
	logger := storage.InitLogger()

	if len(os.Args) != 6 {
		logger.Println("Usage: bazo-miner <dbname> <ipport> <validator> <seedfile> <multisig>")
		return
	}

	dbname := os.Args[1]
	ipport := os.Args[2]
	validator := os.Args[3]
	seedFileName := os.Args[4]
	multisig := os.Args[5]
	isBootstrap := false

	if os.Args[2] == ":8000"{
		isBootstrap = true
	}

	storage.Init(dbname, ipport)
	p2p.Init(ipport)

	validatorPubKey, _, err := storage.ExtractKeyFromFile(validator)
	if err != nil {
		logger.Printf("%v\n", err)
		return
	}

	multisigPubKey, _, err := storage.ExtractKeyFromFile(multisig)
	if err != nil {
		logger.Printf("%v\n", err)
		return
	}

	miner.Init(&validatorPubKey, &multisigPubKey, seedFileName, isBootstrap)
}
