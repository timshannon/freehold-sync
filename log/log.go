// Copyright 2015 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package log

import (
	"log"
	"log/syslog"
	"path/filepath"
	"time"

	"bitbucket.org/tshannon/freehold-sync/datastore"
)

const dsFile = "log.ds"

var (
	//DSFolder is the location of the log DS File
	DSFolder string
	// WriteToSysLog is whether or not the logs will also be written to the system log
	WriteToSysLog bool
)

// Log is a log entry
type Log struct {
	When string `json:"when"`
	Log  string `json:"log"`
}

// New inserts a new log entry
func New(entry string) {
	ds, err := datastore.Open(filepath.Join(DSFolder, dsFile))
	if err != nil {
		syslogError("Error can't log entry to freehold-sync log. Entry: " +
			entry + " error: " + err.Error())
		return

	}
	when := time.Now().Format(time.RFC3339)

	log := &Log{
		When: when,
		Log:  entry,
	}

	err = ds.Put(when, log)
	if err != nil {
		syslogError("Error can't log entry to freehold-sync log. Entry: " +
			entry + " error: " + err.Error())
		return
	}

	if WriteToSysLog {
		syslogError(entry)
	}
}

func syslogError(err string) {
	lWriter, lerr := syslog.New(syslog.LOG_ERR, "freehold-sync")
	if lerr != nil {
		log.Fatal("Error writing to syslog: %v", err)
	}

	lWriter.Err(err)
}
