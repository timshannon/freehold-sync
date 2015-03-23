// Copyright 2015 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package remote

import (
	"fmt"
	"net/url"
	"path/filepath"
	"sync"
	"time"

	"bitbucket.org/tshannon/freehold-sync/datastore"
	"bitbucket.org/tshannon/freehold-sync/log"
	"bitbucket.org/tshannon/freehold-sync/syncer"
)

const (
	remoteDSName = "remote.ds"
	//LogType is the log type for remote logging
	LogType = "remote"
)

var (
	changeHandler ChangeHandler
	watching      profileFiles
	remoteDS      *datastore.DS
	stopWatching  = false
)

func init() {
	watching = profileFiles{
		files: make(map[string][]*syncer.Profile),
	}
}

type profileFiles struct {
	sync.RWMutex
	files map[string][]*syncer.Profile
}

func (p *profileFiles) add(profile *syncer.Profile, file *File) {
	p.RLock()

	if profiles, ok := p.files[file.FullURL()]; ok {
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
		p.files[file.FullURL()] = append(profiles, profile)
		p.Unlock()
		return
	}
	p.RUnlock()
	// not currently watching file
	p.Lock()
	defer p.Unlock()

	p.files[file.FullURL()] = []*syncer.Profile{profile}

	return
}

func (p *profileFiles) has(profile *syncer.Profile, file *File) bool {
	p.RLock()
	defer p.RUnlock()
	if profiles, ok := p.files[file.FullURL()]; ok {
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
	if profiles, ok := p.files[f.FullURL()]; ok {
		return profiles
	}

	return nil
}

func (p *profileFiles) remove(profile *syncer.Profile, file *File) {
	//If profile is nil, remove all from file, and remove watch
	// if last profile is removed, remove watch

	p.RLock()
	if profiles, ok := p.files[file.FullURL()]; ok {
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
func StartWatcher(handler ChangeHandler, dsDir string, pollInterval time.Duration) error {
	var err error
	changeHandler = handler

	remoteDS, err = datastore.Open(filepath.Join(dsDir, remoteDSName))
	if err != nil {
		return err
	}
	// Loop every pollInterval
	// record what the current folder looks like
	// call changeHandler for any file that changed
	// set deleted boolean if file used to exist and no longer does

	go func() {
		var wg sync.WaitGroup
		for {
			wg.Add(1)
			watchList, err := watching.dirWatchList()
			if err != nil {
				log.New(fmt.Sprintf("Error getting watch list: %s", err.Error()), LogType)
			}
			for i := range watchList {

				profiles := watching.profiles(watchList[i])
				go func() {
					defer wg.Done()
					diff, err := watchList[i].differences()
					if err != nil {
						log.New(fmt.Sprintf("Error getting differences for %s: %s", watchList[i].ID(), err.Error()), LogType)
					}
					for d := range diff {
						for p := range profiles {
							changeHandler(profiles[p], diff[d])
						}

					}

				}()
			}

			if stopWatching {
				break
			}
			wg.Wait()
			time.Sleep(pollInterval)
		}
	}()
	return nil
}

// StopWatcher stops the local file system monitoring
func StopWatcher() {
	//stop polling
	stopWatching = true
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

	err = remoteDS.Get(f.ID(), dsFiles)
	if err != nil && err != datastore.ErrNotFound {
		return nil, fmt.Errorf("Error reading remote DS file list for %s", f.ID())
	}

	for i := range dsFiles {
		found := false
		for j := range remFiles {
			if remFiles[i].ID() == dsFiles[i].ID() {
				found = true
				if !remFiles[j].Modified().Equal(dsFiles[i].Modified()) {
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
	err = remoteDS.Put(f.ID(), remFiles)
	if err != nil {
		return nil, err
	}
	return diff, nil
}
