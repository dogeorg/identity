package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path"
	"strings"
	"time"

	"code.dogecoin.org/gossip/dnet"
	"code.dogecoin.org/governor"
	"code.dogecoin.org/identity/internal"
	"code.dogecoin.org/identity/internal/announce"
	"code.dogecoin.org/identity/internal/handler"
	"code.dogecoin.org/identity/internal/store"
	"code.dogecoin.org/identity/internal/web"
)

const WebServerDefaultHost = "127.0.0.1"
const WebServerDefaultPort = 8099
const DBFileName = "identity.db"

func main() {
	bind := internal.Address{
		Host: net.ParseIP(WebServerDefaultHost),
		Port: WebServerDefaultPort,
	}

	dir := "./storage"
	webdir := "./web"
	stderr := log.New(os.Stderr, "", 0)
	flag.Func("dir", "<path> - storage directory (default './storage')", func(arg string) error {
		ent, err := os.Stat(arg)
		if err != nil {
			stderr.Fatalf("--dir: %v", err)
		}
		if !ent.IsDir() {
			stderr.Fatalf("--dir: not a directory: %v", arg)
		}
		dir = arg
		return nil
	})
	flag.Func("web", "<path> - web directory (default './web')", func(arg string) error {
		ent, err := os.Stat(arg)
		if err != nil {
			stderr.Fatalf("--web: %v", err)
		}
		if !ent.IsDir() {
			stderr.Fatalf("--web: not a directory: %v", arg)
		}
		webdir = arg
		return nil
	})
	flag.Func("bind", "<ip>:<port> (use [<ip>]:<port> for IPv6)", func(arg string) error {
		addr, err := parseIPPort(arg, "bind", WebServerDefaultPort)
		if err != nil {
			stderr.Fatalf("%v", err)
		}
		bind = addr
		return nil
	})
	flag.Parse()

	gov := governor.New().CatchSignals().Restart(1 * time.Second)

	// get the private key from the KEY env-var
	idenKey := keyFromEnv()
	log.Printf("Identity PubKey is: %v", hex.EncodeToString(idenKey.Pub[:]))

	storeFilename := path.Join(dir, DBFileName)
	db, err := store.New(storeFilename, gov.GlobalContext())
	if err != nil {
		log.Printf("Error opening database: %v [%s]\n", err, storeFilename)
		os.Exit(1)
	}

	newIdentity := make(chan dnet.RawMessage, 10) // announce -> handler
	announceChanges := make(chan any, 10)         // handler,web -> announce

	identSvc := handler.New(db, idenKey, newIdentity, announceChanges)
	gov.Add("ident", identSvc)
	gov.Add("announce", announce.New(idenKey, db, newIdentity, announceChanges))
	gov.Add("web", web.New(bind.Host, bind.Port, webdir, announceChanges, db))

	gov.Start()
	gov.WaitForShutdown()
}

func keyFromEnv() dnet.KeyPair {
	// get the private key from the KEY env-var
	idenHex := os.Getenv("KEY")
	os.Setenv("KEY", "") // don't leave the key in the environment
	if idenHex == "" {
		log.Printf("Missing KEY env-var: identity private key (32 bytes)")
		os.Exit(3)
	}
	idenKeyB, err := hex.DecodeString(idenHex)
	if err != nil {
		log.Printf("Invalid KEY hex in env-var: %v", err)
		os.Exit(3)
	}
	if len(idenKeyB) != 32 {
		log.Printf("Invalid KEY hex in env-var: must be 32 bytes")
		os.Exit(3)
	}
	return dnet.KeyPairFromPrivKey((*[32]byte)(idenKeyB))
}

func parseIPPort(arg string, name string, defaultPort uint16) (internal.Address, error) {
	// net.SplitHostPort doesn't return a specific error code,
	// so we need to detect if the port it present manually.
	colon := strings.LastIndex(arg, ":")
	bracket := strings.LastIndex(arg, "]")
	if colon == -1 || (arg[0] == '[' && bracket != -1 && colon < bracket) {
		ip := net.ParseIP(arg)
		if ip == nil {
			return internal.Address{}, fmt.Errorf("bad --%v: invalid IP address: %v (use [<ip>]:port for IPv6)", name, arg)
		}
		return internal.Address{
			Host: ip,
			Port: defaultPort,
		}, nil
	}
	res, err := internal.ParseAddress(arg)
	if err != nil {
		return internal.Address{}, fmt.Errorf("bad --%v: invalid IP address: %v (use [<ip>]:port for IPv6)", name, arg)
	}
	return res, nil
}
