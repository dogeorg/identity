package main

import (
	"bufio"
	"encoding/hex"
	"errors"
	"log"
	"math/rand/v2"
	"net"
	"rad/gossip/dnet"
	"rad/gossip/iden"
	"rad/governor"
	"time"
)

// Identity Cache and Protocol-Handler

// Identities are broadcast on the "Iden" channel.
// An identity stays active for 30 days after signing.
// Tracks all currently active identities on the network.
// Allows Identities to be pinned.
// Prepares a set of identities to gossip to peers.

const ProtocolSocket = "/tmp/dogenet.sock"

var ChanIden = dnet.NewTag("Iden")

type IdentityService struct {
	governor.ServiceCtx
	sock    net.Conn
	signKey dnet.PrivKey
	newIden chan iden.IdentityMsg
	idenMsg []byte
	known   map[string]dnet.Message
}

func NewIdentService(signKey dnet.PrivKey, newIden chan iden.IdentityMsg) governor.Service {
	return &IdentityService{
		signKey: signKey,
		newIden: newIden,
	}
}

func (s *IdentityService) Run() {
	for !s.Stopping() {
		err := s.msgLoop()
		if err != nil {
			log.Printf("[Iden] caught panic: %v", err)
		}
		s.Sleep(time.Second)
	}
}

func (s *IdentityService) msgLoop() (e error) {
	// recover and return from a panic
	defer func() {
		if err := recover(); err != nil {
			if er, ok := err.(error); ok {
				e = er
			} else {
				e = errors.New("panic")
			}
		}
	}()
	// connect to dogenet service
	sock, err := net.Dial("unix", ProtocolSocket)
	if err != nil {
		log.Printf("[Iden] cannot connect: %v", err)
		return
	}
	log.Printf("[Iden] connected to dogenet.")
	// send channel bind request
	bind := dnet.BindMessage{Version: 1, Chan: ChanIden}
	_, err = sock.Write(bind.Encode())
	if err != nil {
		log.Printf("[Iden] cannot send BindMessage: %v", err)
		sock.Close()
		return
	}
	s.sock = sock // for Stop()
	go s.chatterIdent(sock)
	reader := bufio.NewReader(sock)
	// read messages until reading fails
	for !s.Stopping() {
		msg, err := dnet.ReadMessage(reader)
		if err != nil {
			log.Printf("[Iden] cannot receive from peer: %v", err)
			sock.Close()
			return
		}
		if msg.Chan != ChanIden {
			log.Printf("[Iden] ignored message: [%s] %s", msg.Chan, msg.Tag)
			continue
		}
		switch msg.Tag {
		case iden.TagIdentity:
			s.recvIden(msg)
		default:
			log.Printf("[Iden] unknown message: [%s] %s", msg.Chan, msg.Tag)
		}
	}
	return
}

func (s *IdentityService) recvIden(msg dnet.Message) {
	id := iden.DecodeIdentityMsg(msg.Payload)
	log.Printf("[Iden] received identity: %v %v %v %v %v signed by: %v", id.Name, id.Country, id.City, id.Lat, id.Long, hex.EncodeToString(msg.PubKey))
	key := hex.EncodeToString(msg.PubKey)
	s.known[key] = msg
}

func (s *IdentityService) Stop() {
	s.sock.Close()
}

func (s *IdentityService) chatterIdent(sock net.Conn) {
	for !s.Stopping() {
		// update identity if it has changed
		select {
		case id := <-s.newIden:
			s.idenMsg = dnet.EncodeMessage(ChanIden, iden.TagIdentity, s.signKey, id.Encode())
			log.Printf("[Iden] signed new identity: %v", s.idenMsg)
		case <-time.After(10 * time.Second):
		}

		count := len(s.known)
		n := rand.N(count + 1)
		if n < count {
			var r dnet.Message
			for _, v := range s.known {
				count--
				if count == 0 {
					r = v
					break
				}
			}
			err := dnet.ForwardMessage(sock, r)
			if err != nil {
				log.Printf("[Iden] cannot send to dogenet: %v", err)
				sock.Close()
				return
			}
			log.Printf("[Iden] sent message: %v %v", r.Chan, r.Tag)
		} else if s.idenMsg != nil {
			_, err := sock.Write(s.idenMsg)
			if err != nil {
				log.Printf("[Iden] cannot send to dogenet: %v", err)
				sock.Close()
				return
			}
			log.Printf("[Iden] sent message: %v %v", ChanIden, iden.TagIdentity)
		}
	}
}
