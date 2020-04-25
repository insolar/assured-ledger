// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package object

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"

	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/store"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
)

// RecordDB is a DB storage implementation. It saves records to disk and does not allow removal.
type RecordDB struct {
	// batchLock sync.Mutex

	db *store.BadgerDB
}

type recordKey insolar.ID

func (k recordKey) Scope() store.Scope {
	return store.ScopeRecord
}

func (k recordKey) DebugString() string {
	id := insolar.ID(k)
	return "recordKey. " + id.DebugString()
}

func (k recordKey) ID() []byte {
	id := insolar.ID(k)
	return id.AsBytes()
}

func newRecordKey(raw []byte) recordKey {
	pulse := insolar.NewPulseNumber(raw)
	hash := raw[pulse.Size():]

	return recordKey(*insolar.NewID(pulse, hash))
}

const (
	recordPositionKeyPrefix          = 0x01
	lastKnownRecordPositionKeyPrefix = 0x02
)

type recordPositionKey struct {
	pn     insolar.PulseNumber
	number uint32
}

func newRecordPositionKey(pn insolar.PulseNumber, number uint32) recordPositionKey {
	return recordPositionKey{pn: pn, number: number}
}

func (k recordPositionKey) Scope() store.Scope {
	return store.ScopeRecordPosition
}

func (k recordPositionKey) ID() []byte {
	parsedNum := make([]byte, 4)
	binary.BigEndian.PutUint32(parsedNum, k.number)
	return bytes.Join([][]byte{{recordPositionKeyPrefix}, k.pn.Bytes(), parsedNum}, nil)
}

func newRecordPositionKeyFromBytes(raw []byte) recordPositionKey {
	k := recordPositionKey{}

	k.pn = insolar.NewPulseNumber(raw[1:])
	k.number = binary.BigEndian.Uint32(raw[(k.pn.Size() + 1):])

	return k
}

func (k recordPositionKey) String() string {
	return fmt.Sprintf("recordPositionKey. pulse: %d, number: %d", k.pn, k.number)
}

type lastKnownRecordPositionKey struct {
	pn insolar.PulseNumber
}

func newLastKnownRecordPositionKey(raw []byte) lastKnownRecordPositionKey {
	k := lastKnownRecordPositionKey{}
	k.pn = insolar.NewPulseNumber(raw[1:])
	return k
}

func (k lastKnownRecordPositionKey) String() string {
	return fmt.Sprintf("lastKnownRecordPositionKey. pulse: %d", k.pn)
}

func (k lastKnownRecordPositionKey) Scope() store.Scope {
	return store.ScopeRecordPosition
}

func (k lastKnownRecordPositionKey) ID() []byte {
	return bytes.Join([][]byte{{lastKnownRecordPositionKeyPrefix}, k.pn.Bytes()}, nil)
}

// NewRecordDB creates new DB storage instance.
func NewRecordDB(db *store.BadgerDB) *RecordDB {
	return &RecordDB{db: db}
}

// setRecord is a helper method for getting last known position of record to db in scope of txn and pulse.
func getLastKnownPosition(txn *badger.Txn, pn insolar.PulseNumber) (uint32, error) {
	key := lastKnownRecordPositionKey{pn: pn}

	fullKey := append(key.Scope().Bytes(), key.ID()...)

	item, err := txn.Get(fullKey)
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return 0, ErrNotFound
		}
		return 0, err
	}

	buff, err := item.ValueCopy(nil)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint32(buff), nil
}

func (r *RecordDB) truncateRecordsHead(ctx context.Context, from insolar.PulseNumber) error {
	keyFrom := recordKey(*insolar.NewID(from, nil))
	it := store.NewReadIterator(r.db.Backend(), keyFrom, false)
	defer it.Close()

	var hasKeys bool
	for it.Next() {
		hasKeys = true
		key := newRecordKey(it.Key())

		err := r.db.Delete(key)
		if err != nil {
			return errors.Wrapf(err, "can't delete key: %s", key.DebugString())
		}

		inslogger.FromContext(ctx).Debugf("Erased key: %s", key.DebugString())
	}

	if !hasKeys {
		inslogger.FromContext(ctx).Infof("No records. Nothing done. Start key: %s", from.String())
	}

	return nil
}

func (r *RecordDB) truncatePositionRecordHead(ctx context.Context, from store.Key, prefix byte) error {
	it := store.NewReadIterator(r.db.Backend(), from, false)
	defer it.Close()

	var hasKeys bool
	for it.Next() {
		hasKeys = true
		if it.Key()[0] != prefix {
			continue
		}
		key := makePositionKey(it.Key())

		err := r.db.Delete(key)
		if err != nil {
			return errors.Wrapf(err, "can't delete key: %s", key)
		}

		inslogger.FromContext(ctx).Debugf("Erased key: %s", key)
	}

	if !hasKeys {
		inslogger.FromContext(ctx).Infof("No records. Nothing done. Start key: %s", from)
	}

	return nil
}

func makePositionKey(raw []byte) store.Key {
	switch raw[0] {
	case recordPositionKeyPrefix:
		return newRecordPositionKeyFromBytes(raw)
	case lastKnownRecordPositionKeyPrefix:
		return newLastKnownRecordPositionKey(raw)
	default:
		panic("unknown prefix: " + string(raw[0]))
	}
}

// TruncateHead remove all records after lastPulse
func (r *RecordDB) TruncateHead(ctx context.Context, from insolar.PulseNumber) error {

	if err := r.truncateRecordsHead(ctx, from); err != nil {
		return errors.Wrap(err, "failed to truncate records head")
	}

	if err := r.truncatePositionRecordHead(ctx, recordPositionKey{pn: from}, recordPositionKeyPrefix); err != nil {
		return errors.Wrap(err, "failed to truncate record positions head")
	}

	if err := r.truncatePositionRecordHead(ctx, lastKnownRecordPositionKey{pn: from}, lastKnownRecordPositionKeyPrefix); err != nil {
		return errors.Wrap(err, "failed to truncate last known record positions head")
	}

	return nil
}

// LastKnownPosition returns last known position of record in Pulse.
func (r *RecordDB) LastKnownPosition(pn insolar.PulseNumber) (uint32, error) {
	var position uint32
	var err error

	err = r.db.Backend().View(func(txn *badger.Txn) error {
		position, err = getLastKnownPosition(txn, pn)
		return err
	})

	return position, err
}

// AtPosition returns record ID for a specific pulse and a position
func (r *RecordDB) AtPosition(pn insolar.PulseNumber, position uint32) (insolar.ID, error) {
	var recID insolar.ID
	err := r.db.Backend().View(func(txn *badger.Txn) error {
		lastKnownPosition, err := getLastKnownPosition(txn, pn)
		if err != nil {
			return err
		}

		if position > lastKnownPosition {
			return ErrNotFound
		}
		positionKey := newRecordPositionKey(pn, position)
		fullKey := append(positionKey.Scope().Bytes(), positionKey.ID()...)

		item, err := txn.Get(fullKey)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return ErrNotFound
			}
			return err
		}
		rawID, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}

		recID = *insolar.NewIDFromBytes(rawID)
		return nil
	})
	return recID, err
}
