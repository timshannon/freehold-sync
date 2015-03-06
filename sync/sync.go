// Copyright 2015 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package sync

import (
	"io"
	"regexp"
	"time"
)

// Direction determines which way a sync will move files
//	DirectionBoth: Sync all files both ways
//	DirectionRemoteOnly: Only sync files up to the remote location, but not down to local
//	DirectionLocalOnly: Only sync files to the local location, but not up to the remote
const (
	DirectionBoth = iota
	DirectionRemoteOnly
	DirectionLocalOnly
)

// ConRes determines the method for Conflict Resolution
// When two files are found to be in conflict (modified within
// a set period of each other), this method is used to resolve it
// Either Overwrite the older file, or Move the older file
const (
	ConResOverwrite = iota
	ConResMove
)

// Syncer is used for comparing two files local or remote
// to determine which one should be overwritten based on
// the sync profile rules
type Syncer interface {
	ID() string
	Modified() time.Time
	Children() ([]Syncer, error)
	Data() (io.ReadCloser, error)
	IsDir() bool
}

// Profile is a profile for syncing folders between a local and
// remote site
// Conflict resolution happens when two files both have modified dates
// within the range of the specified ConflictDuration
// If two files have the same modified date, then there is no conflict, they
// are seen as the same
// For example:
// 	if the conflictDuration is 30 seconds and file1 was modified once
//	at the remote site and once locally within 30 seconds of each other
//	the conflict resolution option is used, wheter the the oldest file is
//	overwritten, or if the older file is moved
// If there is no conflict and the file's modified dates don't match, the
// older file is overwritten
type Profile struct {
	Local              Syncer
	Remote             Syncer
	Direction          int              //direction to sync files
	ConflictResolution int              //Method for handling when there is a sync conflict between two files
	ConflictDuration   time.Duration    //Duration between to file's modified times to determine if there is a conflict
	Ignore             []*regexp.Regexp //List of regular expressions of filepaths to ignore if they match
}
