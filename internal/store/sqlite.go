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
CREATE TABLE IF NOT EXISTS config (
	dayc INTEGER NOT NULL,
	last INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS announce (
	payload BLOB NOT NULL,
	sig BLOB NOT NULL,
	time INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS identity (
	pubkey BLOB PRIMARY KEY NOT NULL,
	payload BLOB NOT NULL,
	sig BLOB NOT NULL,
	time INTEGER NOT NULL,
	dayc INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS profile (
	name TEXT NOT NULL,
	bio TEXT NOT NULL,
	lat INTEGER NOT NULL,
	long INTEGER NOT NULL,
	country TEXT NOT NULL,
	city TEXT NOT NULL,
	icon BLOB NOT NULL
);
CREATE TABLE IF NOT EXISTS nodes (
	pubkey BLOB PRIMARY KEY NOT NULL,
	time INTEGER NOT NULL
);
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
	// init config table
	err = store.initConfig(ctx)
	return store, err
}

func (s *SQLiteStore) Close() {
	s.db.Close()
}

func (s *SQLiteStore) initConfig(ctx context.Context) error {
	sctx := SQLiteStoreCtx{_db: s.db, ctx: ctx}
	return sctx.doTxn("init config", func(tx *sql.Tx) error {
		config := tx.QueryRow("SELECT dayc,last FROM config LIMIT 1")
		var dayc int64
		var last int64
		err := config.Scan(&dayc, &last)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				_, err = tx.Exec("INSERT INTO config (dayc,last) VALUES (1,?)", unixDayStamp())
			}
			return err
		}
		return nil
	})
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

func IsConstraint(err error) bool {
	if sqErr, isSq := err.(sqlite3.Error); isSq {
		if sqErr.Code == sqlite3.ErrConstraint {
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
			return dbErr(err, name+": begin transaction")
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
			return err
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
			return dbErr(err, name+": commit transaction")
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
	if errors.Is(err, spec.ErrNotFound) {
		return err // pass through
	}
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
		// identity expires after 30 days
		res, err := tx.Exec("UPDATE identity SET payload=?,sig=?,time=?,dayc=30+(SELECT dayc FROM config LIMIT 1) WHERE pubkey=? AND time<?", payload, sig, time, pubkey, time)
		if err != nil {
			return err
		}
		num, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if num == 0 {
			_, err = tx.Exec("INSERT INTO identity (pubkey,payload,sig,time,dayc) VALUES (?,?,?,?,30+(SELECT dayc FROM config LIMIT 1))", pubkey, payload, sig, time)
			if IsConstraint(err) {
				return nil // key conflict: means the new time was earlier than the stored record.
			}
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

func (s SQLiteStoreCtx) GetAnnounce() (payload []byte, sig []byte, time int64, err error) {
	err = s.doTxn("GetAnnounce", func(tx *sql.Tx) error {
		row := tx.QueryRow("SELECT payload, sig, time FROM announce LIMIT 1")
		e := row.Scan(&payload, &sig, &time)
		if e != nil {
			if errors.Is(e, sql.ErrNoRows) {
				return spec.ErrNotFound
			} else {
				return fmt.Errorf("query: %v", e)
			}
		}
		return nil
	})
	return
}

func (s SQLiteStoreCtx) SetAnnounce(payload []byte, sig []byte, time int64) error {
	return s.doTxn("SetAnnounce", func(tx *sql.Tx) error {
		res, err := tx.Exec("UPDATE announce SET payload=?,sig=?,time=?", payload, sig, time)
		if err != nil {
			return err
		}
		num, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if num == 0 {
			_, err = tx.Exec("INSERT INTO announce (payload,sig,time) VALUES (?,?,?)", payload, sig, time)
		}
		return err
	})
}

func (s SQLiteStoreCtx) GetProfile() (p spec.Profile, err error) {
	err = s.doTxn("GetProfile", func(tx *sql.Tx) error {
		row := tx.QueryRow("SELECT name,bio,lat,long,country,city,icon FROM profile LIMIT 1")
		e := row.Scan(&p.Name, &p.Bio, &p.Lat, &p.Lon, &p.Country, &p.City, &p.Icon)
		if e != nil {
			if errors.Is(e, sql.ErrNoRows) {
				return spec.ErrNotFound
			} else {
				return fmt.Errorf("query: %v", e)
			}
		}
		return nil
	})
	return
}

func (s SQLiteStoreCtx) SetProfile(p spec.Profile) error {
	return s.doTxn("SetProfile", func(tx *sql.Tx) error {
		res, err := tx.Exec("UPDATE profile SET name=?,bio=?,lat=?,long=?,country=?,city=?,icon=?", p.Name, p.Bio, p.Lat, p.Lon, p.Country, p.City, p.Icon)
		if err != nil {
			return err
		}
		num, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if num == 0 {
			_, err = tx.Exec("INSERT INTO profile (name,bio,lat,long,country,city,icon) VALUES (?,?,?,?,?,?,?)", p.Name, p.Bio, p.Lat, p.Lon, p.Country, p.City, p.Icon)
		}
		return err
	})
}

func (s SQLiteStoreCtx) GetProfileNodes() (nodeList [][]byte, err error) {
	err = s.doTxn("GetProfileNodes", func(tx *sql.Tx) error {
		rows, err := tx.Query("SELECT pubkey FROM nodes")
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var pub []byte
			err = rows.Scan(&pub)
			if err != nil {
				return dbErr(err, "GetProfileNodes: scanning row")
			}
			nodeList = append(nodeList, pub)
		}
		if err = rows.Err(); err != nil { // docs say this check is required!
			return dbErr(err, "GetProfileNodes: query")
		}
		return nil
	})
	return
}

func (s SQLiteStoreCtx) AddProfileNode(pubkey []byte) error {
	return s.doTxn("AddProfileNode", func(tx *sql.Tx) error {
		now := time.Now().Unix()
		res, err := tx.Exec("UPDATE nodes SET time=? WHERE pubkey=?", now, pubkey)
		if err != nil {
			return err
		}
		num, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if num == 0 {
			_, err = tx.Exec("INSERT INTO nodes (pubkey,time) VALUES (?,?)", pubkey, now)
		}
		return err
	})
}

// Trim expires records after N days.
//
// To take account of the possibility that this software has not
// been run in the last N days (which would result in immediately
// expiring all records) we use a system where:
//
// We keep a day counter that we increment once per day.
// All records, when updated, store the current day counter + N.
// Records expire once their stored day-count is < today.
//
// This causes expiry to lag by the number of offline days.
func (s SQLiteStoreCtx) Trim() (advanced bool, err error) {
	err = s.doTxn("Trim", func(tx *sql.Tx) error {
		// check if date has changed
		row := tx.QueryRow("SELECT dayc,last FROM config LIMIT 1")
		var dayc int64
		var last int64
		err := row.Scan(&dayc, &last)
		if err != nil {
			return fmt.Errorf("Trim: SELECT config: %v", err)
		}
		today := unixDayStamp()
		if last != today {
			// advance the day-count and save unix-daystamp
			dayc += 1
			advanced = true
			_, err := tx.Exec("UPDATE config SET dayc=?,last=?", dayc, today)
			if err != nil {
				return fmt.Errorf("Trim: UPDATE: %v", err)
			}
			// expire identities
			_, err = tx.Exec("DELETE FROM identity WHERE dayc < ?", dayc)
			if err != nil {
				return fmt.Errorf("Trim: DELETE: %v", err)
			}
		}
		return nil
	})
	return
}
