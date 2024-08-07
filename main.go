package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"code.dogecoin.org/gossip/iden"
	"code.dogecoin.org/governor"
)

func main() {
	gov := governor.New().CatchSignals().Restart(1 * time.Second)

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(fmt.Sprintf("cannot generate signing key: %v", err))
	}
	log.Printf("Identity PubKey is: %v", hex.EncodeToString(pub))

	newIden := make(chan iden.IdentityMsg, 2)
	gov.Add("ident", NewIdentService(priv, newIden))
	gov.Add("web", NewWebAPI("localhost", 8099, newIden))

	gov.Start()
	gov.WaitForShutdown()
}
