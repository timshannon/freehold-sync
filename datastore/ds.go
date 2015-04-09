// Copyright 2015 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Package datastore manages opening and closing the bolt datastore
// as well as allows a central package for getting buckets to work with
package datastore

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/boltdb/bolt"
)

var ds *bolt.DB

// Supported Buckets
const (
	BucketProfile = "profiles"
	BucketLog     = "log"
	BucketRemote  = "remote"
)

// ErrNotFound is returned when a value isn't found for the passed in key
var ErrNotFound = errors.New("Value not found")

// Open opens a the bolt datastore
func Open(filename string) error {
	db, err := bolt.Open(filename, 0666, &bolt.Options{Timeout: 1 * time.Minute})

	if err != nil {
		return err
	}
	ds = db
	return ds.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(BucketProfile))
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists([]byte(BucketLog))
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists([]byte(BucketRemote))
		if err != nil {
			return err
		}

		return nil
	})
}

// Close closes the bolt datastore
func Close() error {
	if ds != nil {
		return ds.Close()
	}
	return nil
}

// Get gets a value from the DS for the passed in key
func Get(bucket string, key interface{}, result interface{}) error {
	return ds.View(func(tx *bolt.Tx) error {
		dsKey, err := json.Marshal(key)
		if err != nil {
			return err
		}

		dsValue := tx.Bucket([]byte(bucket)).Get(dsKey)

		if dsValue == nil {
			return ErrNotFound
		}

		return json.Unmarshal(dsValue, result)
	})
}

// Put puts a new value in the DS at the given key
func Put(bucket string, key interface{}, value interface{}) error {
	return ds.Update(func(tx *bolt.Tx) error {
		dsKey, err := json.Marshal(key)
		if err != nil {
			return err
		}

		dsValue, err := json.Marshal(value)
		if err != nil {
			return err
		}

		return tx.Bucket([]byte(bucket)).Put(dsKey, dsValue)
	})
}

// Delete removes the value from the DS for the given key
func Delete(bucket string, key interface{}) error {
	return ds.Update(func(tx *bolt.Tx) error {
		dsKey, err := json.Marshal(key)
		if err != nil {
			return err
		}

		return tx.Bucket([]byte(bucket)).Delete(dsKey)
	})
}

// DB returns the underlying bolt DB
func DB() *bolt.DB {
	return ds
}
