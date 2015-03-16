// Copyright 2015 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package local

import (
	"path/filepath"
	"sync"

	"bitbucket.org/tshannon/freehold-sync/log"
	"bitbucket.org/tshannon/freehold-sync/syncer"
	"gopkg.in/fsnotify.v1"
)

var (
	watcher       *fsnotify.Watcher
	changeHandler ChangeHandler
	watching      profileFiles
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

func (p *profileFiles) add(profile *syncer.Profile, file *File) error {
	p.RLock()
	if profiles, ok := p.files[file.ID()]; ok {
		for i := range profiles {
			if profiles[i].ID == profile.ID {
				p.Unlock()
				// file + profile is already being watched
				return nil
			}
		}

		// already watching, but profile is new
		p.Unlock()
		p.Lock()
		p.files[file.ID()] = append(profiles, profile)
		p.Unlock()
		return nil
	}
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
	defer p.Unlock()
	if profiles, ok := p.files[file.ID()]; ok {
		for i := range profiles {
			if profiles[i].ID == profile.ID {
				return true
			}
		}
	}

	return false

}

func (p *profileFiles) profiles(f *File) []*syncer.Profile {
	p.RLock()
	defer p.Unlock()
	parent := filepath.Dir(f.ID())
	if profiles, ok := p.files[parent]; ok {
		return profiles
	}

	return nil
}

func (p *profileFiles) remove(profile *syncer.Profile, file *File) error {
	//If profile is nil, remove all from file, and remove watch
	// if last profile is removed, remove watch

	//TODO: Method for removing monitor when profile is removed

	if profiles, ok := p.files[file.ID()]; ok {
		p.Lock()
		defer p.Unlock()

		if profile == nil {
			delete(p.files, file.ID())
			return watcher.Remove(file.ID())
		}

		for i := range profiles {
			if profiles[i].ID == profile.ID {
				//remove profile
				profiles = append(profiles[:i], profiles[i+1:]...)
			}
		}
		if len(profiles) == 0 {
			delete(p.files, file.ID())
			return watcher.Remove(file.ID())
		}
	}
	// not currently watching file
	return nil
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
					if err != nil {
						log.New(err.Error())
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
				log.New(err.Error())
			}
		}
	}()
	return err
}

// StopWatcher stops the local file system monitoring
func StopWatcher() error {
	return watcher.Close()
}
