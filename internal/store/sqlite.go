package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"code.dogecoin.org/identity/internal/spec"
	"github.com/mattn/go-sqlite3"
)

const SecondsPerDay = 24 * 60 * 60

type SQLiteStore struct {
	db *sql.DB
}

type SQLiteStoreCtx struct {
	_db *sql.DB
	ctx context.Context
}

var _ spec.Store = &SQLiteStore{}

// The common read-only parts of sql.DB and sql.Tx interfaces
type Queryable interface {
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

// WITHOUT ROWID: SQLite version 3.8.2 (2013-12-06) or later

const SQL_SCHEMA string = `
CREATE TABLE IF NOT EXISTS identity (
	pubkey BLOB PRIMARY KEY NOT NULL,
	payload BLOB NOT NULL,
	sig BLOB NOT NULL,
	time INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS identity_time_i ON identity (time);
`

// New returns a spec.Store implementation that uses SQLite
func New(fileName string, ctx context.Context) (spec.Store, error) {
	backend := "sqlite3"
	db, err := sql.Open(backend, fileName)
	store := &SQLiteStore{db: db}
	if err != nil {
		return store, dbErr(err, "opening database")
	}
	setup_sql := SQL_SCHEMA
	if backend == "sqlite3" {
		// limit concurrent access until we figure out a way to start transactions
		// with the BEGIN CONCURRENT statement in Go.
		db.SetMaxOpenConns(1)
	}
	// init tables / indexes
	_, err = db.Exec(setup_sql)
	if err != nil {
		return store, dbErr(err, "creating database schema")
	}
	return store, err
}

func (s *SQLiteStore) Close() {
	s.db.Close()
}

func (s *SQLiteStore) WithCtx(ctx context.Context) spec.StoreCtx {
	return &SQLiteStoreCtx{
		_db: s.db,
		ctx: ctx,
	}
}

// The number of whole days since the unix epoch.
func unixDayStamp() int64 {
	return time.Now().Unix() / SecondsPerDay
}

func IsConflict(err error) bool {
	if sqErr, isSq := err.(sqlite3.Error); isSq {
		if sqErr.Code == sqlite3.ErrBusy || sqErr.Code == sqlite3.ErrLocked {
			return true
		}
	}
	return false
}

func (s SQLiteStoreCtx) doTxn(name string, work func(tx *sql.Tx) error) error {
	db := s._db
	limit := 120
	for {
		tx, err := db.Begin()
		if err != nil {
			if IsConflict(err) {
				s.Sleep(250 * time.Millisecond)
				limit--
				if limit != 0 {
					continue
				}
			}
			return fmt.Errorf("[Store] cannot begin transaction: %v", err)
		}
		defer tx.Rollback()
		err = work(tx)
		if err != nil {
			if IsConflict(err) {
				s.Sleep(250 * time.Millisecond)
				limit--
				if limit != 0 {
					continue
				}
			}
			return fmt.Errorf("[Store] %v: %v", name, err)
		}
		err = tx.Commit()
		if err != nil {
			if IsConflict(err) {
				s.Sleep(250 * time.Millisecond)
				limit--
				if limit != 0 {
					continue
				}
			}
			return fmt.Errorf("[Store] cannot commit %v: %v", name, err)
		}
		return nil
	}
}

func (s SQLiteStoreCtx) Sleep(dur time.Duration) {
	select {
	case <-s.ctx.Done():
	case <-time.After(dur):
	}
}

func dbErr(err error, where string) error {
	if sqErr, isSq := err.(sqlite3.Error); isSq {
		if sqErr.Code == sqlite3.ErrConstraint {
			// MUST detect 'AlreadyExists' to fulfil the API contract!
			// Constraint violation, e.g. a duplicate key.
			return spec.ErrAlreadyExists
		}
		if sqErr.Code == sqlite3.ErrBusy || sqErr.Code == sqlite3.ErrLocked {
			// SQLite has a single-writer policy, even in WAL (write-ahead) mode.
			// SQLite will return BUSY if the database is locked by another connection.
			// We treat this as a transient database conflict, and the caller should retry.
			return spec.ErrDBConflict
		}
	}
	return fmt.Errorf("SQLiteStore: db-problem: %s: %w", where, err)
}

// STORE INTERFACE

func (s SQLiteStoreCtx) SetIdentity(pubkey []byte, payload []byte, sig []byte, time int64) error {
	return s.doTxn("SetIdentity", func(tx *sql.Tx) error {
		res, err := tx.Exec("UPDATE identity SET payload=?,sig=?,time=? WHERE pubkey=?", payload, sig, time, pubkey)
		if err != nil {
			return err
		}
		num, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if num == 0 {
			_, err = tx.Exec("INSERT INTO identity (pubkey,payload,sig,time) VALUES (?,?,?,?)", pubkey, payload, sig, time)
		}
		return err
	})
}

func (s SQLiteStoreCtx) GetIdentity(pubkey []byte) (payload []byte, sig []byte, time int64, err error) {
	err = s.doTxn("GetIdentity", func(tx *sql.Tx) error {
		row := tx.QueryRow("SELECT payload,sig,time FROM identity WHERE pubkey=? LIMIT 1", pubkey)
		e := row.Scan(&payload, &sig, &time)
		if e != nil {
			if errors.Is(e, sql.ErrNoRows) {
				return spec.ErrNotFound
			} else {
				return fmt.Errorf("GetIdentity: %w", e)
			}
		}
		return nil
	})
	return
}

func (s SQLiteStoreCtx) ChooseIdentity() (pubkey []byte, payload []byte, sig []byte, time int64, err error) {
	err = s.doTxn("ChooseIdentity", func(tx *sql.Tx) error {
		row := tx.QueryRow("SELECT pubkey,payload,sig,time FROM identity WHERE oid IN (SELECT oid FROM identity ORDER BY RANDOM() LIMIT 1)")
		e := row.Scan(&pubkey, &payload, &sig, &time)
		if e != nil {
			if errors.Is(e, sql.ErrNoRows) {
				return spec.ErrNotFound
			} else {
				return fmt.Errorf("ChooseIdentity: %w", e)
			}
		}
		return nil
	})
	return
}
