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
	"fmt"
	"log"
	"log/syslog"
	"path/filepath"
	"time"

	"bitbucket.org/tshannon/freehold-sync/datastore"
)

const (
	dsFile   = "log.ds"
	maxRows  = 1000
	pageSize = 25
)

var (
	//DSDir is the location of the log DS File
	DSDir string
)

// Log is a log entry
type Log struct {
	When string `json:"when"`
	Type string `json:"type"`
	Log  string `json:"log"`
}

// New inserts a new log entry
func New(entry, Type string) {

	syslogError(fmt.Sprintf("Type: %s  Entry: %s", Type, entry))

	ds, err := datastore.Open(filepath.Join(DSDir, dsFile))
	if err != nil {
		syslogError("Error can't log entry to freehold-sync log. Entry: " +
			entry + " error: " + err.Error())
		return

	}
	when := time.Now().Format(time.RFC3339)

	log := &Log{
		When: when,
		Type: Type,
		Log:  entry,
	}

	err = ds.Put(when+"_"+Type, log)
	if err != nil {
		syslogError("Error can't log entry to freehold-sync log. Entry: " +
			entry + " error: " + err.Error())
		return
	}

	count, err := logCount()
	if err != nil {
		syslogError("Error can't get log count from freehold-sync log. Error: " + err.Error())
		return
	}

	if count > maxRows {
		// delete oldest records until count is equal with max rows
		for ; count > maxRows; count-- {
			min, err := ds.Min()
			if err != nil {
				syslogError("Error can't delete old logs from freehold-sync log. Error: " + err.Error())
				return
			}

			err = ds.Delete(min)
			if err != nil {
				syslogError("Error can't delete old logs from freehold-sync log. Error: " + err.Error())
				return
			}
		}
	}

}

func logCount() (int, error) {
	ds, err := datastore.Open(filepath.Join(DSDir, dsFile))
	if err != nil {
		return 0, err
	}

	min, err := ds.Min()
	if err != nil {
		return 0, err
	}
	max, err := ds.Max()
	if err != nil {
		return 0, err
	}
	iter, err := ds.Iter(min, max)
	if err != nil {
		return 0, err
	}

	count := 0
	for iter.Next() {
		if iter.Err() != nil {
			return 0, iter.Err()
		}

		count++
	}
	return count, nil

}

// Get retrieves the logs for a given type / page
// if type is "" then return all logs of all types
func Get(Type string, page int) ([]*Log, error) {
	ds, err := datastore.Open(filepath.Join(DSDir, dsFile))
	if err != nil {
		return nil, err
	}

	min, err := ds.Min()
	if err != nil {
		return nil, err
	}
	max, err := ds.Max()
	if err != nil {
		return nil, err
	}

	skip := page * pageSize
	iter, err := ds.Iter(max, min)
	if err != nil {
		return nil, err
	}

	logs := make([]*Log, 0, pageSize)

	for iter.Next() {
		if iter.Err() != nil {
			return nil, iter.Err()
		}

		l := &Log{}
		err = json.Unmarshal(iter.Value(), l)
		if err != nil {
			return nil, err
		}
		if Type != "" && l.Type != Type {
			continue
		}

		if skip <= 0 {
			logs = append(logs, l)
			if len(logs) >= pageSize {
				return logs, nil
			}
		} else {
			skip--
		}
	}

	return logs, nil
}

func syslogError(err string) {
	lWriter, lerr := syslog.New(syslog.LOG_ERR, "freehold-sync")
	if lerr != nil {
		log.Fatal("Error writing to syslog: %v", err)
	}

	lWriter.Err(err)
}
