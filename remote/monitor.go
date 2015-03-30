// Copyright 2015 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package remote

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"bitbucket.org/tshannon/freehold-sync/datastore"
	"bitbucket.org/tshannon/freehold-sync/log"
	"bitbucket.org/tshannon/freehold-sync/syncer"
)

const (
	bucket = datastore.BucketRemote
	//LogType is the log type for remote logging
	LogType = "remote"
)

var (
	changeHandler ChangeHandler
	watching      profileFiles
	stopWatching  = false
	stopped       chan int
	ignore        ignoreFiles //File changes to ignore because they are from this process
)

func init() {
	watching = profileFiles{
		files: make(map[string][]*syncer.Profile),
	}
	stopped = make(chan int)
	ignore = ignoreFiles{
		files: make(map[string]struct{}),
	}
}

type profileFiles struct {
	sync.RWMutex
	files map[string][]*syncer.Profile
}

func (p *profileFiles) add(profile *syncer.Profile, file *File) {
	p.RLock()

	if profiles, ok := p.files[file.ID()]; ok {
		for i := range profiles {
			if profiles[i].ID() == profile.ID() {
				p.RUnlock()
				// file + profile is already being watched
				return
			}
		}

		// already watching, but profile is new
		p.RUnlock()
		p.Lock()
		p.files[file.ID()] = append(profiles, profile)
		p.Unlock()
		return
	}
	p.RUnlock()
	// not currently watching file
	p.Lock()
	defer p.Unlock()

	p.files[file.ID()] = []*syncer.Profile{profile}

	return
}

func (p *profileFiles) has(profile *syncer.Profile, file *File) bool {
	p.RLock()
	defer p.RUnlock()
	if profiles, ok := p.files[file.ID()]; ok {
		for i := range profiles {
			if profiles[i].ID() == profile.ID() {
				return true
			}
		}
	}

	return false

}

func (p *profileFiles) profiles(f *File) []*syncer.Profile {
	p.RLock()
	defer p.RUnlock()
	if profiles, ok := p.files[f.ID()]; ok {
		return profiles
	}

	return nil
}

func (p *profileFiles) remove(profile *syncer.Profile, file *File) {
	//If profile is nil, remove all from file, and remove watch
	// if last profile is removed, remove watch

	p.RLock()
	if profiles, ok := p.files[file.ID()]; ok {
		p.RUnlock()
		p.Lock()
		defer p.Unlock()

		if profile == nil {
			delete(p.files, file.ID())
			return
		}

		for i := range profiles {
			if profiles[i].ID() == profile.ID() {
				//remove profile
				profiles = append(profiles[:i], profiles[i+1:]...)
				break
			}
		}
		if len(profiles) == 0 {
			delete(p.files, file.ID())
			//remove from DS if exists
			datastore.Delete(bucket, file.ID())

			return
		}
	}
	p.RUnlock()
	// not currently watching file
	return
}

func (p *profileFiles) dirWatchList() ([]*File, error) {
	p.RLock()
	defer p.RUnlock()

	result := make([]*File, len(p.files), len(p.files))
	i := 0
	for k, v := range p.files {
		// All profiles watching this folder will share the same client root
		uri, err := url.Parse(k)
		if err != nil {
			return nil, fmt.Errorf("Error parsing file watch url: %v", err)
		}

		result[i], err = New(v[0].Remote.(*File).client, uri.Path)
		if err != nil {
			return nil, fmt.Errorf("Error building remote dir watch list: %v", err)
		}
		i++
	}

	return result, nil
}

// ChangeHandler is the function called when a change occurs in a monitored folder
type ChangeHandler func(*syncer.Profile, syncer.Syncer)

// StartWatcher Starts remote file system monitoring
func StartWatcher(handler ChangeHandler, pollInterval time.Duration) error {
	changeHandler = handler

	// Loop every pollInterval
	// record what the current folder looks like
	// call changeHandler for any file that changed
	// set deleted boolean if file used to exist and no longer does

	go func() {
		var wg sync.WaitGroup
		for {
			watchList, err := watching.dirWatchList()
			if err != nil {
				log.New(fmt.Sprintf("Error getting watch list: %s", err.Error()), LogType)
			}
			for i := range watchList {
				wg.Add(1)
				go func(watchFile *File) {
					defer wg.Done()
					diff, err := watchFile.differences()
					profiles := watching.profiles(watchFile)
					if err != nil {
						log.New(fmt.Sprintf("Error getting differences for %s: %s", watchFile.ID(), err.Error()), LogType)
					}
					for d := range diff {
						for p := range profiles {
							changeHandler(profiles[p], diff[d])
						}

					}

				}(watchList[i])
			}
			wg.Wait()
			if stopWatching {
				stopped <- 1
				break
			}
			//TODO: Use timer so this poll can be interrupted immediately
			time.Sleep(pollInterval)
		}
	}()
	return nil
}

// StopWatcher stops the local file system monitoring
func StopWatcher() {
	//stop polling
	stopWatching = true
	<-stopped // wait for polling loop to stop
}

// Returns the differences between the local record of the folder and
// the current remote view of the folder.  Sets deleted if file used
// to exist
func (f *File) differences() ([]syncer.Syncer, error) {
	var diff []syncer.Syncer
	if !f.IsDir() {
		return nil, nil
	}

	remFiles, err := f.Children()
	if err != nil {
		return nil, err
	}

	var dsFiles []*File

	err = datastore.Get(bucket, f.ID(), &dsFiles)
	if err != nil && err != datastore.ErrNotFound {
		return nil, fmt.Errorf("Error reading remote DS file list for %s: Error: %s", f.ID(), err.Error())
	}

	for i := range dsFiles {
		if ignore.has(dsFiles[i].ID()) {
			continue
		}
		found := false
		for j := range remFiles {
			if remFiles[j].ID() == dsFiles[i].ID() {
				found = true
				//Dirs are always marked as different
				// to ensure they are being monitored see syncer.Profile.Sync
				if !remFiles[j].Modified().Equal(dsFiles[i].Modified()) || remFiles[j].IsDir() {
					diff = append(diff, remFiles[j])
				}
			}
		}
		if !found {
			//Exists in DS but not remote
			// file was deleted
			dsFiles[i].deleted = true
			diff = append(diff, dsFiles[i])
		}
	}

	for i := range remFiles {
		if ignore.has(remFiles[i].ID()) {
			continue
		}
		found := false
		for j := range dsFiles {
			if remFiles[i].ID() == dsFiles[j].ID() {
				found = true
			}
		}
		if !found {
			//Exists in Remote, but not DS
			// file is new

			diff = append(diff, remFiles[i])
		}
	}

	// insert current view of remote site into DS
	err = datastore.Put(bucket, f.ID(), remFiles)
	if err != nil {
		return nil, err
	}

	return diff, nil
}

func deleteRemoteFileFromDS(fileID string) error {
	var dsFiles []*File
	parent := filepath.Dir(strings.TrimRight(fileID, "/"))

	err := datastore.Get(bucket, parent, &dsFiles)
	if err == datastore.ErrNotFound {
		return nil //nothing to delete
	}
	if err != nil {
		return err
	}

	for i := range dsFiles {
		if dsFiles[i].ID() == fileID {
			//Remove file from list
			dsFiles = append(dsFiles[:i], dsFiles[i+1:]...)
			break
		}
	}

	return datastore.Put(bucket, parent, dsFiles)
}

type ignoreFiles struct {
	sync.RWMutex
	files map[string]struct{}
}

func (i *ignoreFiles) add(file string) {
	i.Lock()
	defer i.Unlock()
	i.files[file] = struct{}{}
}

func (i *ignoreFiles) remove(file string) {
	i.Lock()
	defer i.Unlock()
	delete(i.files, file)
}

func (i *ignoreFiles) has(file string) bool {
	i.RLock()
	defer i.RUnlock()
	_, ok := i.files[file]

	return ok
}
