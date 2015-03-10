// Copyright 2015 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package sync

import (
	"errors"
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
	IsDir() bool
	Exists() bool
	Deleted() bool
	Delete() error
	Move(newLocation string) error
	Open() io.ReadWriteCloser
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
	Direction          int              //direction to sync files
	ConflictResolution int              //Method for handling when there is a sync conflict between two files
	ConflictDuration   time.Duration    //Duration between to file's modified times to determine if there is a conflict
	Ignore             []*regexp.Regexp //List of regular expressions of filepaths to ignore if they match
}

// Sync Compares the local and remove files and updates the appropriate one
func (p *Profile) Sync(local, remote Syncer) error {
	if p.ignore(local.ID()) || p.ignore(remote.ID()) {
		return nil
	}

	if !local.Exists() && !remote.Exists() {
		return nil
	}

	if !local.Exists() {
		if local.Deleted() && p.Direction != DirectionLocalOnly {
			return remote.Delete()
		}
	}
	if !remote.Exists() {
		if remote.Deleted() && p.Direction != DirectionRemoteOnly {
			return local.Delete()
		}
	}

	//Both exist Check modified
	if local.Modified().Equal(remote.Modified()) {
		return nil
	}

	var before, after Syncer

	if local.Modified().Before(remote.Modified()) {
		before = local
		after = remote
	} else {
		before = remote
		after = local
	}

	//check for conflict
	if p.isConflict(before, after) {
		//resolve conflict
		if p.ConflictResolution == ConResMove {
			//TODO
		}
		//overwrite, continue as normal
	}

	//overwrite before with after

	return errors.New("TODO: Check modified")

}

func (p *Profile) isConflict(before, after time.Time) bool {
	if !before.Before(after) {
		panic("Invalid conflict times")
	}
	if p.ConflictDuration >= after.Sub(before) {
		return true
	}
	return false
}

func (p *Profile) ignore(id string) bool {
	for i := range p.Ignore {
		if p.Ignore[i].MatchString(id) {
			return true
		}
	}
	return false
}
