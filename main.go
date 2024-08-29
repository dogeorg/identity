package main

import (
	"encoding/hex"
	"log"
	"os"
	"time"

	"code.dogecoin.org/gossip/dnet"
	"code.dogecoin.org/gossip/iden"
	"code.dogecoin.org/governor"
	"code.dogecoin.org/identity/internal/handler"
	"code.dogecoin.org/identity/internal/store"
	"code.dogecoin.org/identity/internal/web"
)

const StoreFilename = "storage/dogenet.db"

func main() {
	gov := governor.New().CatchSignals().Restart(1 * time.Second)

	// get the private key from the KEY env-var
	idenKey := keyFromEnv()
	log.Printf("Identity PubKey is: %v", hex.EncodeToString(idenKey.Pub))

	db, err := store.New(StoreFilename, gov.GlobalContext())
	if err != nil {
		log.Printf("Error opening database: %v [%s]\n", err, StoreFilename)
		os.Exit(1)
	}

	newIden := make(chan iden.IdentityMsg, 10)
	gov.Add("ident", handler.New(db, idenKey, newIden))
	gov.Add("web", web.New("localhost", 8099, newIden))

	gov.Start()
	gov.WaitForShutdown()
}

func keyFromEnv() dnet.KeyPair {
	// get the private key from the KEY env-var
	idenHex := os.Getenv("KEY")
	os.Setenv("KEY", "") // don't leave the key in the environment
	if idenHex == "" {
		log.Printf("Missing KEY env-var: identity private key (64 bytes)")
		os.Exit(3)
	}
	idenKey, err := hex.DecodeString(idenHex)
	if err != nil {
		log.Printf("Invalid KEY hex in env-var: %v", err)
		os.Exit(3)
	}
	if len(idenKey) != 64 {
		log.Printf("Invalid KEY hex in env-var: must be 64 bytes")
		os.Exit(3)
	}
	return dnet.KeyPairFromPrivKey(idenKey)
}
