package main

import (
	"encoding/hex"
	"log"
	"os"
	"time"

	"code.dogecoin.org/gossip/dnet"
	"code.dogecoin.org/gossip/iden"
	"code.dogecoin.org/governor"
	"code.dogecoin.org/identity/internal/announce"
	"code.dogecoin.org/identity/internal/handler"
	"code.dogecoin.org/identity/internal/store"
	"code.dogecoin.org/identity/internal/web"
)

const StoreFilename = "storage/identity.db"

func main() {
	gov := governor.New().CatchSignals().Restart(1 * time.Second)

	// get the private key from the KEY env-var
	idenKey, nodePub := keyFromEnv()
	log.Printf("Identity PubKey is: %v", hex.EncodeToString(idenKey.Pub))

	db, err := store.New(StoreFilename, gov.GlobalContext())
	if err != nil {
		log.Printf("Error opening database: %v [%s]\n", err, StoreFilename)
		os.Exit(1)
	}

	newIdentity := make(chan dnet.RawMessage, 10)
	newProfile := make(chan iden.IdentityMsg, 10)
	nodePubList := [][]byte{nodePub}

	identSvc := handler.New(db, idenKey, newIdentity)
	gov.Add("ident", identSvc)
	gov.Add("announce", announce.New(idenKey, db, newIdentity, newProfile, nodePubList))
	gov.Add("web", web.New("localhost", 8099, newProfile))

	gov.Start()
	gov.WaitForShutdown()
}

func keyFromEnv() (idenKey dnet.KeyPair, nodePub dnet.PubKey) {
	// get the private key from the KEY env-var
	idenHex := os.Getenv("KEY")
	os.Setenv("KEY", "") // don't leave the key in the environment
	if idenHex == "" {
		log.Printf("Missing KEY env-var: identity private key (64 bytes)")
		os.Exit(3)
	}
	idenKeyB, err := hex.DecodeString(idenHex)
	if err != nil {
		log.Printf("Invalid KEY hex in env-var: %v", err)
		os.Exit(3)
	}
	if len(idenKeyB) != 64 {
		log.Printf("Invalid KEY hex in env-var: must be 64 bytes")
		os.Exit(3)
	}
	// get the node pubkey from NODE env-var
	nodeHex := os.Getenv("NODE")
	if nodeHex == "" {
		log.Printf("Missing NODE env-var: node public key (32 bytes)")
		os.Exit(3)
	}
	nodePubB, err := hex.DecodeString(nodeHex)
	if err != nil {
		log.Printf("Invalid NODE hex in env-var: %v", err)
		os.Exit(3)
	}
	if len(nodePubB) != 32 {
		log.Printf("Invalid NODE hex in env-var: must be 32 bytes")
		os.Exit(3)
	}
	return dnet.KeyPairFromPrivKey(idenKeyB), nodePubB
}
