package handler

import (
	"bufio"
	"encoding/hex"
	"log"
	"net"
	"time"

	"code.dogecoin.org/gossip/dnet"
	"code.dogecoin.org/gossip/iden"
	"code.dogecoin.org/governor"
	"code.dogecoin.org/identity/internal/spec"
)

// Identity Cache and Protocol-Handler

// Identities are broadcast on the "Iden" channel.x
// An identity stays active for 30 days after signing.
// Tracks all currently active identities on the network.
// Allows Identities to be pinned.
// Prepares a set of identities to gossip to peers.

const ProtocolSocket = "/tmp/dogenet.sock"
const OneUnixDay = 86400

var ChanIden = dnet.NewTag("Iden")

type IdentityService struct {
	governor.ServiceCtx
	_store  spec.Store
	store   spec.StoreCtx
	sock    net.Conn
	idenKey dnet.KeyPair
	newIden chan dnet.RawMessage
	idenMsg dnet.RawMessage
}

func New(store spec.Store, idenKey dnet.KeyPair, newIden chan dnet.RawMessage) governor.Service {
	return &IdentityService{
		_store:  store,
		idenKey: idenKey,
		newIden: newIden,
	}
}

func (s *IdentityService) Run() {
	// bind store to context
	s.store = s._store.WithCtx(s.Context)
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
	go s.gossipMyIdentity(sock)
	go s.gossipRandomIdentities(sock)
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
			log.Printf("[Iden] ignored message: [%s][%s]", msg.Chan, msg.Tag)
			continue
		}
		switch msg.Tag {
		case iden.TagIdentity:
			s.recvIden(msg)
		default:
			log.Printf("[Iden] unknown message: [%s][%s]", msg.Chan, msg.Tag)
		}
	}
}

func (s *IdentityService) recvIden(msg dnet.Message) {
	id := iden.DecodeIdentityMsg(msg.Payload)
	days := (id.Time.Local().Unix() - time.Now().Unix()) / OneUnixDay
	log.Printf("[Iden] received identity: %v %v %v %v %v signed by: %v (%v days remain)", id.Name, id.Country, id.City, id.Lat, id.Long, hex.EncodeToString(msg.PubKey), days)
	s.store.SetIdentity(msg.PubKey, msg.Payload, msg.Signature, id.Time.Local().Unix())
}

func (s *IdentityService) Stop() {
	s.sock.Close()
}

// goroutine
func (s *IdentityService) gossipMyIdentity(sock net.Conn) {
	for !s.Stopping() {
		// update identity if it has changed
		select {
		case rawMsg := <-s.newIden:
			s.idenMsg = rawMsg
			log.Printf("[Iden] signed new identity: %v", s.idenMsg)
		case <-time.After(9 * time.Second):
		}
		if s.idenMsg.Header != nil {
			err := s.idenMsg.Send(sock)
			if err != nil {
				log.Printf("[Iden] cannot send to dogenet: %v", err)
				sock.Close()
				return
			}
			log.Printf("[Iden] sent message: %v %v", ChanIden, iden.TagIdentity)
		}
	}
}

// goroutine
func (s *IdentityService) gossipRandomIdentities(sock net.Conn) {
	for !s.Stopping() {
		// wait for next turn
		time.Sleep(11 * time.Second)

		// choose a random identity
		pub, payload, sig, _, err := s.store.ChooseIdentity()
		if err != nil {
			if spec.IsNotFoundError(err) {
				log.Printf("[Iden]: no identities to gossip")
			} else {
				log.Printf("[Iden]: %v", err)
			}
			continue
		}

		// send the message to peers
		msg := dnet.ReEncodeMessage(ChanIden, iden.TagIdentity, pub, sig, payload)
		err = msg.Send(sock)
		if err != nil {
			log.Printf("[Iden] cannot send to dogenet: %v", err)
			sock.Close()
			return
		}
		log.Printf("[Iden] sent message: %v %v", ChanIden, iden.TagIdentity)
	}
}
