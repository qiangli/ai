package db

import (
	"os"
	"path/filepath"

	bolt "go.etcd.io/bbolt"
)

// https://github.com/etcd-io/bbolt?tab=readme-ov-file

type KVStore struct {
	base string
	db   *bolt.DB
}

func NewKVStore(base string) *KVStore {
	return &KVStore{base: base}
}

func (r *KVStore) Open(name string) error {
	if err := os.MkdirAll(r.base, 0755); err != nil {
		return err
	}
	db, err := bolt.Open(filepath.Join(r.base, name), 0600, nil)
	if err != nil {
		return err
	}
	r.db = db
	return nil
}

func (r *KVStore) Close() error {
	return r.db.Close()
}

func (r *KVStore) Create(bucket string) error {
	err := r.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucket))
		return err
	})
	return err
}

func (r *KVStore) Drop(bucket string) error {
	err := r.db.Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket([]byte(bucket))
	})
	return err
}

func (r *KVStore) Put(bucket, key string, value string) error {
	err := r.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		err := b.Put([]byte(key), []byte(value))
		return err
	})
	return err
}

func (r *KVStore) Get(bucket, key string) (*string, error) {
	var val []byte

	// The Get() function does not return an error
	r.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		v := b.Get([]byte(key))
		val = v
		return nil
	})
	// not found
	if val == nil {
		return nil, nil
	}
	result := string(val)
	return &result, nil
}

func (r *KVStore) Delete(bucket, key string) error {
	err := r.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		return b.Delete([]byte(key))
	})
	return err
}
