package announce

import (
	"bytes"
	"encoding/hex"
	"log"
	"time"

	"code.dogecoin.org/gossip/dnet"
	"code.dogecoin.org/gossip/iden"
	"code.dogecoin.org/gossip/node"
	"code.dogecoin.org/governor"
	"code.dogecoin.org/identity/internal/spec"
)

// const AnnounceLongevity = 24 * time.Hour
const AnnounceLongevity = 5 * time.Minute
const QueueAnnouncement = 10 * time.Second

type Announce struct {
	governor.ServiceCtx
	_store       spec.Store
	store        spec.StoreCtx
	idenKey      dnet.KeyPair         // identity keypair for signing address messages
	changes      chan any             // input: changes to the profile
	receiver     chan dnet.RawMessage // output: receives new announcement RawMessages
	profile      iden.IdentityMsg     // next identity profile to encode and sign
	nodePubList  [][]byte             // list of node pubkeys to include in identity annoucement
	profileValid bool                 // we have stored profile
}

func New(idenKey dnet.KeyPair, store spec.Store, receiver chan dnet.RawMessage, changes chan any) *Announce {
	return &Announce{
		_store:   store,
		idenKey:  idenKey,
		changes:  changes,
		receiver: receiver,
	}
}

// goroutine
func (ns *Announce) Run() {
	ns.store = ns._store.WithCtx(ns.Context) // Service Context is first available here
	ns.loadProfile()
	ns.updateAnnounce()
}

func (ns *Announce) updateAnnounce() {
	remain := AnnounceLongevity
	if ns.profileValid {
		msg, rem, ok := ns.loadOrGenerateAnnounce()
		remain = rem
		if ok {
			ns.receiver <- msg
		}
	}
	timer := time.NewTimer(remain)
	for !ns.Stopping() {
		select {
		case change := <-ns.changes:
			changed := false
			switch msg := change.(type) {
			case iden.IdentityMsg:
				if msg.IsValid() {
					log.Printf("[announce] received new profile: %v", msg.Name)
					ns.profile = msg
					ns.profileValid = true
					ns.saveProfile(msg)
					changed = true
				} else {
					log.Printf("[announce] received invalid profile (ingored)")
				}
			case spec.NodePubKeyMsg:
				log.Printf("[announce] received node pubkey: %v", hex.EncodeToString(msg.PubKey))
				if !ns.nodeListContains(msg.PubKey) {
					ns.nodePubList = append(ns.nodePubList, msg.PubKey)
					changed = true
				}
			default:
				log.Printf("[announce] received unknown change: %v", msg)
			}
			// whenever a change is received, reset timer.
			if changed {
				if !timer.Stop() {
					<-timer.C
				}
				timer.Reset(QueueAnnouncement)
			}

		case <-timer.C:
			// every 24 hours, re-sign and gossip the announcement.
			remain := AnnounceLongevity
			if ns.profileValid {
				msg, rem, ok := ns.generateAnnounce(ns.profile)
				remain = rem
				if ok {
					ns.receiver <- msg
					log.Printf("[announce] sending announcement to all peers")
				}
			}
			// restart the timer
			timer.Reset(remain)

		case <-ns.Context.Done():
			timer.Stop()
			return
		}
	}
}

func (ns *Announce) nodeListContains(key []byte) bool {
	// check if nodePubList contains the specified key
	for _, k := range ns.nodePubList {
		if bytes.Equal(k, key) {
			return true
		}
	}
	return false
}

func (ns *Announce) loadOrGenerateAnnounce() (dnet.RawMessage, time.Duration, bool) {
	// load the stored announcement from the database
	oldPayload, sig, expires, err := ns.store.GetAnnounce()
	now := time.Now().Unix()
	if err != nil {
		log.Printf("[announce] cannot load announcement: %v", err)
	} else if len(oldPayload) >= node.AddrMsgMinSize && len(sig) == 64 && now < expires {
		// determine if the announcement we stored is the same as the announcement
		// we would produce now; if so, avoid gossiping a new announcement
		// XXX broken by delayed NodePubKeyMsg message (never Equals)
		oldMsg := node.DecodeAddrMsg(oldPayload) // for Time
		newMsg := ns.profile                     // copy
		newMsg.Time = oldMsg.Time                // ignore Time for Equals()
		if bytes.Equal(newMsg.Encode(), oldPayload) {
			// re-encode the stored announcement
			log.Printf("[announce] re-using stored announcement for %v seconds", expires-now)
			msg := dnet.ReEncodeMessage(dnet.ChannelIdentity, iden.TagIdentity, ns.idenKey.Pub, sig, oldPayload)
			return msg, time.Duration(expires-now) * time.Second, true
		}
	}
	// create a new announcement and store it
	return ns.generateAnnounce(ns.profile)
}

func (ns *Announce) generateAnnounce(profile iden.IdentityMsg) (dnet.RawMessage, time.Duration, bool) {
	// wait for at least one node key
	if !(profile.IsValid() && len(profile.Nodes) > 0) {
		return dnet.RawMessage{}, AnnounceLongevity, false
	}
	log.Printf("[announce] signing a new announcement")
	now := time.Now()
	profile.Time = dnet.UnixToDoge(now)
	profile.Nodes = ns.nodePubList
	payload := profile.Encode()
	msg := dnet.EncodeMessage(dnet.ChannelIdentity, iden.TagIdentity, ns.idenKey, payload)
	view := dnet.MsgView(msg)
	err := ns.store.SetAnnounce(payload, view.Signature()[:], now.Add(AnnounceLongevity).Unix())
	if err != nil {
		log.Printf("[announce] cannot store announcement: %v", err)
	}
	return dnet.RawMessage{Header: view.Header(), Payload: payload}, AnnounceLongevity, true
}

func (ns *Announce) loadProfile() {
	// load the user's configured profile information.
	p, err := ns.store.GetProfile()
	if err != nil {
		if spec.IsNotFoundError(err) {
			log.Printf("[announce] no profile stored.")
		} else {
			log.Printf("[announce] cannot load profile: %v", err)
		}
		return
	}
	// messy, but profile will end up with a lot more data
	ns.profile.Name = p.Name
	ns.profile.Bio = p.Bio
	ns.profile.Lat = int16(p.Lat)
	ns.profile.Long = int16(p.Long)
	ns.profile.Country = p.Country
	ns.profile.City = p.City
	ns.profile.Icon = p.Icon
	ns.profileValid = ns.profile.IsValid()
}

func (ns *Announce) saveProfile(p iden.IdentityMsg) {
	// not sure this is the announce service's responsibility;
	// most of this will come from "profile editor" with additional data
	// and we just need a signal that it has changed.
	pro := spec.Profile{
		Name:    p.Name,
		Bio:     p.Bio,
		Lat:     int(p.Lat),
		Long:    int(p.Long),
		Country: p.Country,
		City:    p.City,
		Icon:    p.Icon,
	}
	ns.store.SetProfile(pro)
}
