// Copyright 2015 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package syncer

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"sync"
	"time"
)

var syncing syncingData // tracks which profiles and files are currently syncing

func init() {
	syncing = syncingData{
		profiles: make(map[string]int),
		files:    make(map[string]struct{}),
	}
}

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
	ID() string                                                 // Unique ID for the file, usually includes the full path to the file
	Path(p *Profile) string                                     // Relative path to the file based on the passed in Profile
	Modified() time.Time                                        // Last time the file was modified
	IsDir() bool                                                // whether or not the file is a dir
	Exists() bool                                               // Whether or not the file exists
	Deleted() bool                                              // If the file doesn't exist was it deleted
	Delete() error                                              // Deletes the file
	Rename() error                                              // Renames the file in the case of a conflict.
	Open() (io.ReadCloser, error)                               // Opens the file for reading
	Write(r io.ReadCloser, size int64, modTime time.Time) error // Writes from the reader to the Syncer, closes reader
	Size() int64                                                // Size of the file
	CreateDir() (Syncer, error)                                 // Create a New Directory based on the non-existant syncer's name
	StartMonitor(*Profile) error                                // Start Monitoring this syncer for changes (Dir's only)
	StopMonitor(*Profile) error                                 // Stop Monitoring this syncer for changes (Dir's only)
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
	Name               string           //Name of the profile
	Direction          int              //direction to sync files
	ConflictResolution int              //Method for handling when there is a sync conflict between two files
	ConflictDuration   time.Duration    //Duration between to file's modified times to determine if there is a conflict
	Ignore             []*regexp.Regexp //List of regular expressions of filepaths to ignore if they match

	Local  Syncer //Local starting point for syncing
	Remote Syncer // Remote starting point for syncing
}

// ID uniquely identifies a profile.  Is a combination of
// Local ID + Remote ID which ensures that the same profile isn't monitored / synced twice
func (p *Profile) ID() string {
	return p.Local.ID() + "_" + p.Remote.ID()
}

// Start starts syncing the Profile
func (p *Profile) Start() error {
	if p.Local == nil {
		return errors.New("Local sync starting point not set.")
	}
	if p.Remote == nil {
		return errors.New("Remote sync starting point not set.")
	}
	go func() {
		p.Sync(p.Local, p.Remote)
	}()
	return nil
}

// Stop stops the profile from syncing
func (p *Profile) Stop() error {
	err := p.Local.StopMonitor(p)
	if err != nil {
		return err
	}
	return p.Remote.StopMonitor(p)
}

//TODO: remove after testing
func (p *Profile) logDebug(msg string) {
	fmt.Printf("\t %s\n", msg)
}

// Sync Compares the local and remove files and updates the appropriate one
func (p *Profile) Sync(local, remote Syncer) error {
	// if file is already syncing in another thread drop out earily
	// first come first sync
	fmt.Printf("Start syncing %s with %s\n", local.ID(), remote.ID())
	defer fmt.Printf("Stop syncing %s with %s\n", local.ID(), remote.ID())

	if syncing.is(local) {
		p.logDebug("local dropping out early because it's already syncing elsewhere")
		return nil
	}
	if syncing.is(remote) {
		p.logDebug("remote dropping out early because it's already syncing elsewhere")
		return nil
	}

	syncing.start(p, local, remote)
	defer syncing.stop(p, local, remote)

	if !local.Exists() && !remote.Exists() {
		return nil
	}

	if p.ignore(local.ID()) || p.ignore(remote.ID()) {
		return nil
	}

	if local.IsDir() && local.Exists() {

		if remote.Exists() && !remote.IsDir() {
			// rename file, create dir
			err := remote.Rename()
			if err != nil {
				return err
			}
			dir, err := remote.CreateDir()
			if err != nil {
				return err
			}
			return dir.StartMonitor(p)
		}
	}

	if remote.IsDir() && remote.Exists() {
		if local.Exists() && !local.IsDir() {
			// rename file, create dir
			err := local.Rename()
			if err != nil {
				return err
			}
			dir, err := local.CreateDir()
			if err != nil {
				return err
			}
			return dir.StartMonitor(p)
		}
	}

	if !local.Exists() {
		p.logDebug("Local doesn't exist")
		if local.Deleted() {
			if p.Direction != DirectionLocalOnly {
				p.logDebug("Local was deleted, deleting remote")
				return remote.Delete()
			}
			return nil
		}
		if p.Direction != DirectionRemoteOnly {
			p.logDebug("Writing Local")
			//write local
			if remote.IsDir() {
				dir, err := local.CreateDir()
				if err != nil {
					return err
				}
				err = dir.StartMonitor(p)
				if err != nil {
					return err
				}
				return remote.StartMonitor(p)
			}
			return p.copy(remote, local)
		}
		return nil
	}
	if !remote.Exists() {
		p.logDebug("Remote doesn't exist")
		if remote.Deleted() {
			if p.Direction != DirectionRemoteOnly {
				p.logDebug("Remote was deleted, deleting local")
				return local.Delete()
			}
			return nil
		}
		if p.Direction != DirectionLocalOnly {
			p.logDebug("Writing Remote")
			//write remote
			if local.IsDir() {
				dir, err := remote.CreateDir()
				if err != nil {
					return err
				}
				err = dir.StartMonitor(p)
				if err != nil {
					return err
				}
				return local.StartMonitor(p)
			}

			return p.copy(local, remote)
		}
		return nil
	}

	if (local.IsDir() && local.Exists()) && (remote.IsDir() && remote.Exists()) {
		// Only start monitoring if local and remote folders are both exist
		err := local.StartMonitor(p) // may already exist, but we'll let the interface handle that
		if err != nil {
			return err
		}

		err = remote.StartMonitor(p) // may already exist, but we'll let the interface handle that
		if err != nil {
			return err
		}

	}

	if local.IsDir() || remote.IsDir() {
		//Handled by monitors
		return nil
	}

	//Both exist Check modified
	if remote.Modified().Equal(local.Modified()) {
		//Already in Sync
		return nil
	}

	var before, after Syncer

	if local.Modified().Before(remote.Modified()) {
		if p.Direction == DirectionRemoteOnly {
			return nil
		}

		p.logDebug("Remote will overwrite Local")
		before = local
		after = remote
	} else {
		//remote before local

		p.logDebug("Local will overwrite Remote")
		if p.Direction == DirectionLocalOnly {
			return nil
		}
		before = remote
		after = local
	}

	//check for conflict
	if p.isConflict(before.Modified(), after.Modified()) {
		p.logDebug("Conflict found")
		//resolve conflict
		if p.ConflictResolution == ConResRename {
			p.logDebug("Conflict rename")
			before.Rename()
		}
	}

	p.logDebug(fmt.Sprintf("Overwriting before with after: before: %s after %s", before.ID(), after.ID()))
	return p.copy(after, before)
}

func (p *Profile) copy(source, dest Syncer) error {
	r, err := source.Open()
	if err != nil {
		return err
	}

	err = dest.Write(r, source.Size(), source.Modified())
	return err
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

type syncingData struct {
	sync.RWMutex
	profiles map[string]int
	files    map[string]struct{}
}

func (sd *syncingData) start(p *Profile, local, remote Syncer) {
	sd.Lock()
	defer sd.Unlock()
	count := sd.profiles[p.ID()]
	count++
	sd.profiles[p.ID()] = count
	sd.files[local.ID()] = struct{}{}
	sd.files[remote.ID()] = struct{}{}
}

func (sd *syncingData) stop(p *Profile, local, remote Syncer) {
	sd.Lock()
	defer sd.Unlock()
	count := sd.profiles[p.ID()]
	if count > 0 {
		count--
		sd.profiles[p.ID()] = count
	}
	delete(sd.files, local.ID())
	delete(sd.files, remote.ID())
}

func (sd *syncingData) count(profileID string) int {
	sd.RLock()
	defer sd.RUnlock()
	return sd.profiles[profileID]
}

// is is whether or not the passed in file is currently syncing
func (sd *syncingData) is(s Syncer) bool {
	sd.RLock()
	defer sd.RUnlock()
	_, ok := sd.files[s.ID()]
	return ok
}

// ProfileSyncCount returns the number of files currently
// sycing on the passed in profile
func ProfileSyncCount(profileID string) int {
	return syncing.count(profileID)
}
