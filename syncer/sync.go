// Copyright 2015 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package syncer

import (
	"errors"
	"io"
	"regexp"
	"sync"
	"time"
)

var syncing syncingData // tracks which profiles are currently syncing

//right now changes are happening single threaded. In order to make multiple changes concurrently
// we'll need to build a dependency graph; i.e. you can't run deletes on a directory you are write a file to, etc
// might be more trouble than it's worth
// or maybe a simple:
// 	if file isn't currently syncing, and none of files parents are currently syncing
// 	then sync immediately?

func init() {
	syncing = syncingData{
		profiles: make(map[string]int),
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

const (
	changeTypeWrite = iota
	changeTypeDelete
	changeTypeRename
	changeTypeCreateDir
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

	changes chan *changeItem // collects all changes as they come in and runs them in the order they arrive
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

	p.changes = make(chan *changeItem, 200)
	go func() {
		p.Sync(p.Local, p.Remote)
	}()
	go func() {
		for change := range p.changes {
			change.runChange()
		}
	}()

	return nil
}

// Stop stops the profile from syncing
func (p *Profile) Stop() error {
	err := p.Local.StopMonitor(p)
	if err != nil {
		return err
	}
	err = p.Remote.StopMonitor(p)
	if err != nil {
		return err
	}

	if p.changes != nil {
		close(p.changes)
	}
	return nil
}

// Sync Compares the local and remove files and updates the appropriate one
func (p *Profile) Sync(local, remote Syncer) error {
	syncing.start(p)
	defer syncing.stop(p)

	if !local.Exists() && !remote.Exists() {
		return nil
	}

	if p.ignore(local.ID()) || p.ignore(remote.ID()) {
		return nil
	}

	var err error

	if local.IsDir() && local.Exists() {

		if remote.Exists() && !remote.IsDir() {
			// rename file, create dir
			err = <-p.rename(remote)
			if err != nil {
				return err
			}

			return <-p.createDir(local, remote)
		}
	}

	if remote.IsDir() && remote.Exists() {
		if local.Exists() && !local.IsDir() {
			err = <-p.rename(local)
			if err != nil {
				return err
			}
			return <-p.createDir(remote, local)
		}
	}

	if !local.Exists() {
		if local.Deleted() {
			if p.Direction != DirectionLocalOnly {
				return <-p.delete(remote)
			}
			return nil
		}
		if p.Direction != DirectionRemoteOnly {
			//write local
			if remote.IsDir() {
				return <-p.createDir(remote, local)
			}
			return <-p.write(remote, local)
		}
		return nil
	}

	if !remote.Exists() {
		if remote.Deleted() {
			if p.Direction != DirectionRemoteOnly {
				return <-p.delete(local)
			}
			return nil
		}
		if p.Direction != DirectionLocalOnly {
			//write remote
			if local.IsDir() {
				return <-p.createDir(local, remote)
			}
			return <-p.write(local, remote)
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
		//Handled by monitors, nothing to sync
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
			return <-p.rename(before)
		}
	}

	return <-p.write(after, before)
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

func (p *Profile) rename(s Syncer) chan error {
	return queueChange(p, nil, s, changeTypeRename)
}

func (p *Profile) createDir(from, to Syncer) chan error {
	return queueChange(p, from, to, changeTypeCreateDir)
}
func (p *Profile) delete(s Syncer) chan error {
	return queueChange(p, nil, s, changeTypeDelete)
}
func (p *Profile) write(from, to Syncer) chan error {
	return queueChange(p, from, to, changeTypeWrite)
}

type syncingData struct {
	sync.RWMutex
	profiles map[string]int
}

func (sd *syncingData) start(p *Profile) {
	sd.Lock()
	defer sd.Unlock()
	count := sd.profiles[p.ID()]
	count++
	sd.profiles[p.ID()] = count
}

func (sd *syncingData) stop(p *Profile) {
	sd.Lock()
	defer sd.Unlock()
	count := sd.profiles[p.ID()]
	if count > 0 {
		count--
		sd.profiles[p.ID()] = count
	}
}

func (sd *syncingData) count(profileID string) int {
	sd.RLock()
	defer sd.RUnlock()
	return sd.profiles[profileID]
}

// ProfileSyncCount returns the number of files currently
// sycing on the passed in profile
func ProfileSyncCount(profileID string) int {
	return syncing.count(profileID)
}

type changeItem struct {
	changeType int
	from, to   Syncer
	profile    *Profile
	done       chan error
}

func (c *changeItem) runChange() {
	switch c.changeType {
	case changeTypeCreateDir:
		dir, err := c.to.CreateDir()
		if err != nil {
			c.done <- err
			return
		}
		err = dir.StartMonitor(c.profile)
		if err != nil {
			c.done <- err
			return
		}
		c.done <- c.from.StartMonitor(c.profile)

	case changeTypeDelete:
		c.done <- c.to.Delete()
	case changeTypeRename:
		c.done <- c.to.Rename()
	case changeTypeWrite:
		r, err := c.from.Open()
		if err != nil {
			c.done <- err
			return
		}
		c.done <- c.to.Write(r, c.from.Size(), c.from.Modified())
	}
}

func queueChange(p *Profile, from, to Syncer, changeType int) chan error {
	done := make(chan error)
	p.changes <- &changeItem{
		changeType: changeType,
		from:       from,
		to:         to,
		profile:    p,
		done:       done,
	}
	return done
}
