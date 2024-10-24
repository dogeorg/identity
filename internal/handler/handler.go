package handler

import (
	"bufio"
	"encoding/hex"
	"io"
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

const OneUnixDay = 86400
const GossipIdentityInverval = 71 * time.Second // gossip a random identity to peers

var ChanIden = dnet.NewTag("Iden")

type IdentityService struct {
	governor.ServiceCtx
	_store          spec.Store
	store           spec.StoreCtx
	bind            spec.BindTo
	sock            net.Conn
	idenKey         dnet.KeyPair
	newIden         chan dnet.RawMessage // from announce.go
	announceChanges chan any
	idenMsg         dnet.RawMessage
}

func New(bind spec.BindTo, store spec.Store, idenKey dnet.KeyPair, newIden chan dnet.RawMessage, announceChanges chan any) governor.Service {
	return &IdentityService{
		_store:          store,
		bind:            bind,
		idenKey:         idenKey,
		newIden:         newIden,
		announceChanges: announceChanges,
	}
}

func (s *IdentityService) Run() {
	// bind store to context
	s.store = s._store.WithCtx(s.Context)
	// connect to dogenet service
	sock, err := net.Dial(s.bind.Network, s.bind.Address)
	if err != nil {
		log.Printf("[Iden] cannot connect: %v", err)
		return
	}
	log.Printf("[Iden] connected to dogenet.")
	// send channel bind request
	bind := dnet.BindMessage{Version: 1, Chan: ChanIden, PubKey: *s.idenKey.Pub}
	_, err = sock.Write(bind.Encode())
	if err != nil {
		log.Printf("[Iden] cannot send BindMessage: %v", err)
		sock.Close()
		return
	}
	// wait for the return bind request
	reader := bufio.NewReader(sock)
	br_buf := [dnet.BindMessageSize]byte{}
	_, err = io.ReadAtLeast(reader, br_buf[:], len(br_buf))
	if err != nil {
		log.Printf("[Iden] reading BindMessage reply: %v", err)
		sock.Close()
		return
	}
	if br, ok := dnet.DecodeBindMessage(br_buf[:]); ok {
		// send the node's pubkey to the announce service
		// so it can include the node key in the identity announcement
		s.announceChanges <- spec.NodePubKeyMsg{PubKey: br.PubKey[:]}
	} else {
		log.Printf("[Iden] invalid BindMessage reply: %v", err)
		sock.Close()
		return
	}
	log.Printf("[Iden] completed handshake.")
	// begin sending and listening for messages
	s.sock = sock // for Stop()
	go s.gossipMyIdentity(sock)
	go s.gossipRandomIdentities(sock)
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
		// gossip my identity when it changes
		rawMsg := <-s.newIden
		s.idenMsg = rawMsg
		log.Printf("[Iden] gossiping new identity")
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
		time.Sleep(GossipIdentityInverval)

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
		msg := dnet.ReEncodeMessage(ChanIden, iden.TagIdentity, (*[32]byte)(pub), sig, payload)
		err = msg.Send(sock)
		if err != nil {
			log.Printf("[Iden] cannot send to dogenet: %v", err)
			sock.Close()
			return
		}
		log.Printf("[Iden] sent message: %v %v", ChanIden, iden.TagIdentity)
	}
}
