// Copyright 2015 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package syncer

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
// Either Overwrite the older file, or Rename the older file
const (
	ConResOverwrite = iota
	ConResRename
)

// Syncer is used for comparing two files local or remote
// to determine which one should be overwritten based on
// the sync profile rules
type Syncer interface {
	ID() string                              // Unique ID for the file, usually includes the full path to the file
	Modified() time.Time                     // Last time the file was modified
	IsDir() bool                             // whether or not the file is a dir
	Exists() bool                            // Whether or not the file exists
	Deleted() bool                           // If the file doesn't exist was it deleted
	Delete() error                           // Deletes the file
	Rename() error                           // Renames the file in the case of a conflic.  The Interface implementation chooses how
	Open() (io.ReadCloser, error)            // Opens the file for reading
	Write(r io.ReadCloser, size int64) error // Writes from the reader to the Syncer
	Size() int64                             //size of the file
	CreateDir() error                        //Create a New Directory based on the non-existant syncer's name
	StartMonitor(*Profile) error             //Start Monitoring this syncer for changes (Dir's only), calls profile.Sync method on all changes, and initial startup
	StopMonitor(*Profile) error              //Stop Monitoring this syncer for changes (Dir's only)
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
	ID                 string           //Unique identifier for the profile
	Direction          int              //direction to sync files
	ConflictResolution int              //Method for handling when there is a sync conflict between two files
	ConflictDuration   time.Duration    //Duration between to file's modified times to determine if there is a conflict
	Ignore             []*regexp.Regexp //List of regular expressions of filepaths to ignore if they match

	Local  Syncer //Local starting point for syncing
	Remote Syncer // Remote starting point for syncing
}

// Start starts syncing the Profile
func (p *Profile) Start() error {
	return p.Sync(p.Local, p.Remote)
}

// Stop stops the profile from syncing
func (p *Profile) Stop() error {
	err := p.Local.StopMonitor(p)
	if err != nil {
		return err
	}
	return p.Remote.StopMonitor(p)
}

// Sync Compares the local and remove files and updates the appropriate one
func (p *Profile) Sync(local, remote Syncer) error {
	if !local.Exists() && !remote.Exists() {
		return nil
	}

	if p.ignore(local.ID()) || p.ignore(remote.ID()) {
		return nil
	}

	if local.IsDir() {
		err := local.StartMonitor(p) // may already exist, but we'll let the interface handle that
		if err != nil {
			return err
		}
		if !remote.IsDir() {
			// rename file, create dir
			err = remote.Rename()
			if err != nil {
				return err
			}
			return remote.CreateDir()
		}
	}

	if remote.IsDir() {
		err := remote.StartMonitor(p) // may already exist, but we'll let the interface handle that
		if err != nil {
			return err
		}
		if !local.IsDir() {
			// rename file, create dir
			err = local.Rename()
			if err != nil {
				return err
			}
			return local.CreateDir()
		}
	}

	if !local.Exists() {
		if local.Deleted() {
			if p.Direction != DirectionLocalOnly {
				return remote.Delete()
			}
			return nil
		}
		if p.Direction != DirectionRemoteOnly {
			//write local
			if remote.IsDir() {
				return local.CreateDir()
			}
			return p.copy(remote, local)
		}
		return nil
	}
	if !remote.Exists() {
		if remote.Deleted() {
			if p.Direction != DirectionRemoteOnly {
				return local.Delete()
			}
			return nil
		}
		if p.Direction != DirectionLocalOnly {
			//write remote
			if local.IsDir() {
				return remote.CreateDir()
			}
			return p.copy(local, remote)
		}
		return nil
	}

	if local.IsDir() || remote.IsDir() {
		//Handled by monitors
		return nil
	}

	//Both exist Check modified
	if local.Modified().Equal(remote.Modified()) {
		//Already in Sync
		return nil
	}

	var before, after Syncer

	if local.Modified().Before(remote.Modified()) {
		if p.Direction == DirectionRemoteOnly {
			return nil
		}
		before = local
		after = remote
	} else {
		//remote before local
		if p.Direction == DirectionLocalOnly {
			return nil
		}
		before = remote
		after = local
	}

	//check for conflict
	if p.isConflict(before.Modified(), after.Modified()) {
		//resolve conflict
		if p.ConflictResolution == ConResRename {
			before.Rename()
		}
	}

	return p.copy(after, before)
}

func (p *Profile) copy(source, dest Syncer) error {
	r, err := source.Open()
	defer r.Close()
	if err != nil {
		return err
	}

	return dest.Write(r, source.Size())
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
