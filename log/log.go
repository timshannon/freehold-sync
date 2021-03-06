// Copyright 2015 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Package log will log last 1000 items in a datstore file
// an the system log.  New records will push old records out
// of the datastore, but they will remain in the system log for
// the users system to manage
package log

import (
	"encoding/json"
	"time"

	"github.com/boltdb/bolt"

	"bitbucket.org/tshannon/freehold-sync/datastore"
)

const (
	bucket   = datastore.BucketLog
	maxRows  = 10000
	pageSize = 25
)

// Log is a log entry
type Log struct {
	When string `json:"when"`
	Type string `json:"type"`
	Log  string `json:"log"`
}

// New inserts a new log entry
func New(entry, Type string) {
	when := time.Now().Format(time.RFC3339)

	log := &Log{
		When: when,
		Type: Type,
		Log:  entry,
	}

	err := datastore.Put(bucket, when+"_"+Type, log)
	if err != nil {
		panic("Error can't log entry to freehold-sync log. Entry: " +
			entry + " error: " + err.Error())
		return
	}

	err = trimOldLogs()
	if err != nil {
		panic("Error can't trim old log entries: " +
			entry + " error: " + err.Error())
		return

	}
}

type key []byte

func trimOldLogs() error {
	return datastore.DB().Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		c := b.Cursor()
		count := 0

		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			count++
			if count > maxRows {
				err := c.Delete()
				if err != nil {
					return err
				}
			}
		}

		return nil
	})
}

// Get retrieves the logs for a given type / page
// if type is "" then return all logs of all types
func Get(Type string, page int) ([]*Log, error) {
	skip := page * pageSize
	logs := make([]*Log, 0, pageSize)

	err := datastore.DB().View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		c := b.Cursor()

		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			l := &Log{}
			err := json.Unmarshal(v, l)
			if err != nil {
				return err
			}
			if Type != "" && l.Type != Type {
				continue
			}

			if skip <= 0 {
				logs = append(logs, l)
				if len(logs) >= pageSize {
					break
				}
			} else {
				skip--
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return logs, nil
}
