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
	SetIdentity(pub []byte, payload []byte, sig []byte, time int64) error
	GetIdentity(pub []byte) (payload []byte, sig []byte, time int64, err error)
	ChooseIdentity() (pubkey []byte, payload []byte, sig []byte, time int64, err error)
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
