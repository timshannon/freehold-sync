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

// LogType is the log type for local syncing
const LogType = "local"

var (
	watcher       *fsnotify.Watcher
	changeHandler ChangeHandler
	watching      profileFiles // folders being watched for changes
	ignore        ignoreFiles  //File changes to ignore because they are from this process
	changes       changeMap    //queued up changes to a given file, makes sure excessive calls to sync don't happen
)

func init() {
	watching = profileFiles{
		files: make(map[string][]*syncer.Profile),
	}
	ignore = ignoreFiles{
		files: make(map[string]struct{}),
	}
	changes = changeMap{
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
				file, err := New(event.Name)
				if err != nil {
					log.New(err.Error(), LogType)
					continue
				}
				if ignore.has(file.ID()) {
					continue
				}
				if event.Op == fsnotify.Rename || event.Op == fsnotify.Remove {
					file.deleted = true
				}

				queueChange(file)

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
	}
	return nil
}

type changeMap struct {
	sync.RWMutex
	files map[string]struct{}
}

func (c *changeMap) add(f *File) {
	c.Lock()
	defer c.Unlock()
	c.files[f.ID()] = struct{}{}
}

func (c *changeMap) remove(f *File) {
	c.Lock()
	defer c.Unlock()
	delete(c.files, f.ID())
}

func (c *changeMap) has(f *File) bool {
	c.RLock()
	defer c.RUnlock()
	_, ok := c.files[f.ID()]
	return ok
}

// queueChange queues up a change and waits for the file to stop
// changing to before sending the changeHandler signal
// Subsequent queued events for the same file will group together
// into one change event until the change handler is called
func queueChange(f *File) {
	if !changes.has(f) {
		changes.add(f)
		go func() {
			defer changes.remove(f)
			f.waitInUse() // wait for the file to stop changing
			profiles := watching.profiles(f)
			for i := range profiles {
				changeHandler(profiles[i], f)
			}
		}()
	}

}
