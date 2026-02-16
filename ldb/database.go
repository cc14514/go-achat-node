// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 liangchuan

package ldb

import (
	"bytes"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

// OpenFileLimit is retained for compatibility with older callers.
// It is not currently used by this package.
var OpenFileLimit = 64

type LDBDatabase struct {
	fn string
	db *leveldb.DB
}

// NewLDBDatabase opens (or creates) a LevelDB database at file.
// cache and handles are interpreted as coarse tuning knobs and default to
// small but reasonable values when set too low.
func NewLDBDatabase(file string, cache int, handles int) (*LDBDatabase, error) {
	if cache < 16 {
		cache = 16
	}
	if handles < 16 {
		handles = 16
	}

	opts := &opt.Options{
		OpenFilesCacheCapacity: handles,
		BlockCacheCapacity:     cache / 2 * opt.MiB,
		WriteBuffer:            cache / 4 * opt.MiB,
		Filter:                 filter.NewBloomFilter(10),
	}

	db, err := leveldb.OpenFile(file, opts)
	if err != nil {
		return nil, err
	}
	return &LDBDatabase{fn: file, db: db}, nil
}

var Kfilter = func(prefix, k []byte) bool {
	if k != nil && len(k) > len(prefix) {
		return bytes.Equal(k[:len(prefix)], prefix)
	}
	return false
}

// Path returns the path to the database directory.
func (db *LDBDatabase) Path() string { return db.fn }

func (db *LDBDatabase) Put(key []byte, value []byte) error { return db.db.Put(key, value, nil) }

func (db *LDBDatabase) Has(key []byte) (bool, error) { return db.db.Has(key, nil) }

func (db *LDBDatabase) Get(key []byte) ([]byte, error) { return db.db.Get(key, nil) }

func (db *LDBDatabase) Delete(key []byte) error { return db.db.Delete(key, nil) }

func (db *LDBDatabase) NewIterator() iterator.Iterator { return db.db.NewIterator(nil, nil) }

func (db *LDBDatabase) Close() { _ = db.db.Close() }

func (db *LDBDatabase) LDB() *leveldb.DB { return db.db }

func (db *LDBDatabase) NewBatch() Batch {
	return &ldbBatch{db: db.db, b: new(leveldb.Batch)}
}

type ldbBatch struct {
	db   *leveldb.DB
	b    *leveldb.Batch
	size int
}

func (b *ldbBatch) Put(key, value []byte) error {
	b.b.Put(key, value)
	b.size += len(value)
	return nil
}

func (b *ldbBatch) Write() error { return b.db.Write(b.b, nil) }

func (b *ldbBatch) ValueSize() int { return b.size }

type table struct {
	db     Database
	prefix string
}

// NewTable returns a Database object that prefixes all keys with a given
// string.
func NewTable(db Database, prefix string) Database {
	return &table{
		db:     db,
		prefix: prefix,
	}
}

func (dt *table) NewIterator() iterator.Iterator {
	return dt.db.NewIterator()
}

func (dt *table) Put(key []byte, value []byte) error {
	return dt.db.Put(append([]byte(dt.prefix), key...), value)
}

func (dt *table) Has(key []byte) (bool, error) {
	return dt.db.Has(append([]byte(dt.prefix), key...))
}

func (dt *table) Get(key []byte) ([]byte, error) {
	return dt.db.Get(append([]byte(dt.prefix), key...))
}

func (dt *table) Delete(key []byte) error {
	return dt.db.Delete(append([]byte(dt.prefix), key...))
}

func (dt *table) Close() {
	// Do nothing; don't close the underlying DB.
}

type tableBatch struct {
	batch  Batch
	prefix string
}

// NewTableBatch returns a Batch object which prefixes all keys with a given string.
func NewTableBatch(db Database, prefix string) Batch {
	return &tableBatch{db.NewBatch(), prefix}
}

func (dt *table) NewBatch() Batch {
	return &tableBatch{dt.db.NewBatch(), dt.prefix}
}

func (tb *tableBatch) Put(key, value []byte) error {
	return tb.batch.Put(append([]byte(tb.prefix), key...), value)
}

func (tb *tableBatch) Write() error {
	return tb.batch.Write()
}

func (tb *tableBatch) ValueSize() int {
	return tb.batch.ValueSize()
}
