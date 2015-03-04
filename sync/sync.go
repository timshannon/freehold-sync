// Copyright 2015 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package sync

import (
	"regexp"
	"time"
)

const (
	DirectionBoth       = iota //Sync all files both ways
	DirectionRemoteOnly        //only sync files up to the remote location, but not down to local
	DirectionLocalOnly         //only sync files to the local location, but not up to the remote
)

const (
	ConResOverwrite = iota
	ConResMove
)

// Syncer is used for comparing two files local or remote
// to determine which one should be overwritten based on
// the sync profile rules
type Syncer interface {
	Id() string
	Modified() *time.Time
	Children() []Syncer
	Data() []byte
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
