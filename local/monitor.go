// Copyright 2015 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package local

import (
	"bitbucket.org/tshannon/freehold-sync/sync"
	"gopkg.in/fsnotify.v1"
)

var (
	watcher       *fsnotify.Watcher
	changeHandler ChangeHandler
)

// ChangeHandler is the function called when a change occurs in a monitored folder
type ChangeHandler func(*sync.Profile, sync.Syncer)

// StartWatcher Starts local file system monitoring
func StartWatcher(handler ChangeHandler) error {
	var err error
	changeHandler = handler
	watcher, err = fsnotify.NewWatcher()

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Write == fsnotify.Write ||
					event.Op&fsnotify.Create == fsnotify.Create {
					//TODO: Call changeHandler, need profile from filename

				}
				if event.Op&fsnotify.Rename == fsnotify.Rename ||
					event.Op&fsnotify.Remove == fsnotify.Remove {
					//TODO: Call changeHandler with deleted file

				}

			case err := <-watcher.Errors:
				//TODO: Error Logging along with a sync log?
			}
		}
	}()
	return err
}

// StopWatcher stops the local file system monitoring
func StopWatcher() error {
	return watcher.Close()
}
