// Copyright 2015 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package local

import (
	"fmt"
	"path/filepath"
	"sync"

	"bitbucket.org/tshannon/freehold-sync/log"
	"bitbucket.org/tshannon/freehold-sync/syncer"
	"gopkg.in/fsnotify.v1"
)

// LogType is the log type for local syncing
const LogType = "local"

var (
	watcher       *fsnotify.Watcher
	changeHandler ChangeHandler
	watching      profileFiles // folders being watched for changes
	ignore        ignoreFiles  //File changes to ignore because they are from this process
)

func init() {
	watching = profileFiles{
		files: make(map[string][]*syncer.Profile),
	}
	ignore = ignoreFiles{
		files: make(map[string]struct{}),
	}
}

type profileFiles struct {
	sync.RWMutex
	files map[string][]*syncer.Profile
}

func (p *profileFiles) add(profile *syncer.Profile, file *File) error {
	p.RLock()
	if profiles, ok := p.files[file.ID()]; ok {
		for i := range profiles {
			if profiles[i].ID() == profile.ID() {
				p.RUnlock()
				// file + profile is already being watched
				return nil
			}
		}

		// already watching, but profile is new
		p.RUnlock()
		p.Lock()
		p.files[file.ID()] = append(profiles, profile)
		p.Unlock()
		return nil
	}
	p.RUnlock()
	// not currently watching file
	p.Lock()
	defer p.Unlock()

	err := watcher.Add(file.ID())
	if err != nil {
		return err
	}

	p.files[file.ID()] = []*syncer.Profile{profile}

	return nil
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
	parent := filepath.Dir(f.ID())
	if profiles, ok := p.files[parent]; ok {
		return profiles
	}

	return nil
}

func (p *profileFiles) remove(profile *syncer.Profile, file *File) error {
	//If profile is nil, remove all from file, and remove watch
	// if last profile is removed, remove watch

	p.RLock()
	if profiles, ok := p.files[file.ID()]; ok {
		p.RUnlock()
		p.Lock()
		defer p.Unlock()

		if profile == nil {
			delete(p.files, file.ID())
			return watcher.Remove(file.ID())
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
			return watcher.Remove(file.ID())
		}
	}
	p.RUnlock()
	// not currently watching file
	return nil
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

// ChangeHandler is the function called when a change occurs in a monitored folder
type ChangeHandler func(*syncer.Profile, syncer.Syncer)

// StartWatcher Starts local file system monitoring
func StartWatcher(handler ChangeHandler) error {
	var err error
	changeHandler = handler
	watcher, err = fsnotify.NewWatcher()

	go func() {
		for {
			select {
			case event := <-watcher.Events:

				if event.Op != fsnotify.Chmod {
					file, err := New(event.Name)
					if ignore.has(file.ID()) {
						fmt.Println("Skipping ", file.ID())
						return
					}
					if err != nil {
						log.New(err.Error(), LogType)
						continue
					}
					if event.Op == fsnotify.Rename || event.Op == fsnotify.Remove {
						file.deleted = true
					}

					profiles := watching.profiles(file)
					for i := range profiles {
						changeHandler(profiles[i], file)
					}

				}

			case err := <-watcher.Errors:
				log.New(err.Error(), LogType)
			}
		}
	}()
	return err
}

// StopWatcher stops the local file system monitoring
func StopWatcher() error {
	watching.RLock()
	defer watching.RUnlock()
	if len(watching.files) > 0 {
		//nil error if nothing is being watched
		return watcher.Close()

		return nil
	}
	return nil
}
