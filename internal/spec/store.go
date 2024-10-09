package spec

import (
	"context"
	"errors"
	"time"
)

// Keep identities for 30 days before expiry
const ExpiryTime = time.Duration(30 * 24 * time.Hour)

// Store is the top-level interface (e.g. SQLiteStore)
type Store interface {
	WithCtx(ctx context.Context) StoreCtx
}

// StoreCtx is a Store bound to a cancellable Context
type StoreCtx interface {
	// Insert or Update an Identity (only update if time is newer!)
	SetIdentity(pub []byte, payload []byte, sig []byte, time int64) error
	// Get stored identity by pubkey.
	GetIdentity(pub []byte) (payload []byte, sig []byte, time int64, err error)
	// Get a random stored identity (to gossip)
	ChooseIdentity() (pubkey []byte, payload []byte, sig []byte, time int64, err error)
	// Get the stored announcement, if any.
	GetAnnounce() (payload []byte, sig []byte, time int64, err error)
	SetAnnounce(payload []byte, sig []byte, time int64) error
	GetProfile() (profile Profile, err error)
	SetProfile(profile Profile) error
	GetProfileNodes() (nodeList [][]byte, err error)
	AddProfileNode(pubkey []byte) error
	Trim() (advanced bool, err error)
}

var ErrNotFound = errors.New("not found")
var ErrAlreadyExists = errors.New("already exists")
var ErrDBConflict = errors.New("conflict")

func IsNotFoundError(err error) bool {
	return errors.Is(err, ErrNotFound)
}

func IsAlreadyExistsError(err error) bool {
	return errors.Is(err, ErrAlreadyExists)
}
