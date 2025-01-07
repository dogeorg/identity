package announce

import (
	"bytes"
	"log"
	"time"

	"code.dogecoin.org/gossip/dnet"
	"code.dogecoin.org/gossip/iden"
	"code.dogecoin.org/governor"
	"code.dogecoin.org/identity/internal/spec"
)

const AnnounceLongevity = 24 * time.Hour
const QueueAnnouncement = 10 * time.Second

type Announce struct {
	governor.ServiceCtx
	_store       spec.Store
	store        spec.StoreCtx
	idenKey      dnet.KeyPair         // identity keypair for signing address messages
	changes      chan any             // input: changes to the profile
	receiver     chan dnet.RawMessage // output: receives new announcement RawMessages
	profile      iden.IdentityMsg     // next identity profile to encode and sign
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
			case spec.Profile:
				// new profile from web API (already stored in db)
				newIden := iden.IdentityMsg{
					Time:    dnet.DogeNow(),
					Name:    msg.Name,
					Bio:     msg.Bio,
					Lat:     int16(msg.Lat),
					Long:    int16(msg.Lon),
					Country: msg.Country,
					City:    msg.City,
					Nodes:   ns.profile.Nodes, // preserve nodeList
					Icon:    msg.Icon,
				}
				if newIden.IsValid() {
					log.Printf("[announce] received new profile: %v %v %v %v %v", msg.Name, msg.Lat, msg.Lon, newIden.Lat, newIden.Long)
					ns.profile = newIden
					ns.profileValid = true
					changed = true
				} else {
					log.Printf("[announce] received invalid profile (ingored)")
				}
			case spec.NodePubKeyMsg:
				log.Printf("[announce] received node pubkey: %x", msg.PubKey)
				if !ns.nodeListContains(msg.PubKey) {
					ns.profile.Nodes = append(ns.profile.Nodes, msg.PubKey)
					changed = true
					err := ns.store.AddProfileNode(msg.PubKey)
					if err != nil {
						log.Printf("[announce] cannot save announcement node: '%x': %v", msg.PubKey, err)
					}
				}
			default:
				log.Printf("[announce] received unknown change: %v", msg)
			}
			// whenever a change is received, re-sign and gossip the announcement.
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
	for _, k := range ns.profile.Nodes {
		if bytes.Equal(k, key) {
			return true
		}
	}
	return false
}

func (ns *Announce) loadOrGenerateAnnounce() (msg dnet.RawMessage, remaining time.Duration, isValid bool) {
	defer func() {
		if err := recover(); err != nil {
			// crash during decode; create a new announcement and store it
			log.Printf("[announce] crash during announcement decode: %v", err)
			msg, remaining, isValid = ns.generateAnnounce(ns.profile)
		}
	}()
	// load the stored announcement from the database
	oldPayload, sig, expires, err := ns.store.GetAnnounce()
	if err != nil {
		log.Printf("[announce] cannot load announcement: %v", err)
		return ns.generateAnnounce(ns.profile)
	}
	now := time.Now().Unix()
	if len(oldPayload) >= iden.IdenMsgMinSize && len(sig) == 64 && now < expires {
		// determine if the identity message we stored is the same as the identity
		// we would produce now; if so, avoid gossiping a new identity
		oldMsg := iden.DecodeIdentityMsg(oldPayload) // for Time
		newMsg := ns.profile                         // copy
		newMsg.Time = oldMsg.Time                    // ignore Time for Equals()
		if bytes.Equal(newMsg.Encode(), oldPayload) {
			// re-encode the stored identity
			log.Printf("[announce] re-using stored identity for %v seconds", expires-now)
			msg = dnet.ReEncodeMessage(dnet.ChannelIdentity, iden.TagIdentity, ns.idenKey.Pub, sig, oldPayload)
			remaining = time.Duration(expires-now) * time.Second
			isValid = true
			return
		}
	}
	// create a new announcement and store it
	return ns.generateAnnounce(ns.profile)
}

func (ns *Announce) generateAnnounce(profile iden.IdentityMsg) (dnet.RawMessage, time.Duration, bool) {
	// wait for at least one node pubkey.
	// an identity without any nodes is useless, and if we sign an identity
	// now, it will be invalidated when we add the local node's pubkey.
	if !(profile.IsValid() && len(profile.Nodes) > 0) {
		return dnet.RawMessage{}, AnnounceLongevity, false
	}

	// create and sign the new announcement.
	log.Printf("[announce] signing a new announcement")
	now := time.Now()
	profile.Time = dnet.UnixToDoge(now)
	payload := profile.Encode()
	msg := dnet.EncodeMessage(dnet.ChannelIdentity, iden.TagIdentity, ns.idenKey, payload)
	view := dnet.MsgView(msg)
	sig := view.Signature()[:]

	// store the announcement to re-use on next startup.
	expires := now.Add(AnnounceLongevity).Unix()
	err := ns.store.SetAnnounce(payload, sig, expires)
	if err != nil {
		log.Printf("[announce] cannot store announcement: %v", err)
	}

	// update this node's identity in the local identity database.
	// this makes the identity visible to services on the local node.
	idenPub := ns.idenKey.Pub[:]
	err = ns.store.SetIdentity(idenPub, payload, sig, now.Unix())
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
	nodeList, err := ns.store.GetProfileNodes()
	if err != nil {
		log.Printf("[announce] cannot load profile nodes: %v", err)
		return
	}
	ns.profile = iden.IdentityMsg{
		Name:    p.Name,
		Bio:     p.Bio,
		Lat:     int16(p.Lat),
		Long:    int16(p.Lon),
		Country: p.Country,
		City:    p.City,
		Nodes:   nodeList,
		Icon:    p.Icon,
	}
	ns.profileValid = ns.profile.IsValid()
}
