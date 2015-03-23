// Copyright 2015 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package remote

import (
	"errors"
	"io"
	"path/filepath"
	"strings"
	"time"

	fh "bitbucket.org/tshannon/freehold-client"
	"bitbucket.org/tshannon/freehold-sync/datastore"
	"bitbucket.org/tshannon/freehold-sync/syncer"
)

// File is implements the syncer.Syncer interface
// for a file on the Remote machine
type File struct {
	*fh.File
	client  *fh.Client
	deleted bool
	exists  bool
}

// New Returns a File from the remote instance for use in syncing
func New(client *fh.Client, filePath string) (*File, error) {
	f := &File{
		exists: false,
		client: client,
	}

	file, err := client.GetFile(filePath)
	if fh.IsNotFound(err) {
		f.File = &fh.File{
			URL:  filePath,
			Name: filepath.Base(filePath),
		}
		return f, nil
	}
	if err != nil {
		return nil, err
	}

	f.File = file
	f.exists = true

	return f, nil
}

// Client is the freehold client used to retrieve this file
func (f *File) Client() *fh.Client {
	return f.client
}

// ID is the unique identifier for a remote file
func (f *File) ID() string {
	return f.URL
}

// Modified is the date the file was last modified
func (f *File) Modified() time.Time {
	if !f.IsDir() && f.exists {
		return f.ModifiedTime()
	}
	return time.Time{}
}

// Children returns the child files for this given File, will only return
// records if the file is a Dir
func (f *File) Children() ([]*File, error) {
	if !f.exists {
		return nil, nil
	}
	children, err := f.File.Children()
	if err != nil {
		return nil, err
	}
	syncers := make([]*File, len(children))

	for i := range children {
		syncers[i] = &File{
			File:    children[i],
			client:  f.Client(),
			deleted: false,
			exists:  true,
		}
	}
	return syncers, nil
}

// Open returns a ReadWriteCloser for reading, and writing data to the file
func (f *File) Open() (io.ReadCloser, error) {
	return f, nil
}

// Write writes from the reader to the Syncer
func (f *File) Write(r io.ReadCloser, size int64, modTime time.Time) error {
	if f.IsDir() {
		return errors.New("Can't write a directory with this method")
	}
	var err error
	if f.exists {
		err = f.File.Delete()
		if err != nil {
			return err
		}
	}
	dest := &fh.File{
		URL:   filepath.Dir(f.URL),
		Name:  filepath.Base(filepath.Dir(f.URL)),
		IsDir: true,
	}

	newFile, err := f.client.UploadFromReader(f.Name, r, size, modTime, dest)
	if err != nil {
		return err
	}

	f.File = newFile

	f.exists = true
	f.deleted = false
	return nil
}

// IsDir is whether or not the file is a directory
func (f *File) IsDir() bool {
	if !f.exists {
		return false
	}
	return f.File.IsDir
}

// Exists is whether or not the file exists
func (f *File) Exists() bool {
	return f.exists
}

// Delete deletes the file
func (f *File) Delete() error {
	if !f.exists {
		return nil
	}
	err := f.File.Delete()
	if err != nil {
		return err
	}

	var dsFiles []*File
	parent := filepath.Dir(strings.TrimRight(f.ID(), "/"))

	err = remoteDS.Get(parent, dsFiles)
	if err != nil && err != datastore.ErrNotFound {
		return nil // nothing to delete
	}

	for i := range dsFiles {
		if dsFiles[i].ID() == f.ID() {
			//Remove file from list
			dsFiles = append(dsFiles[:i], dsFiles[i+1:]...)
			break
		}
	}

	if f.IsDir() {
		//Remove watcher
		watching.remove(nil, f)
	}

	//TODO: recursively remove child watchers

	return remoteDS.Put(parent, dsFiles)
}

// Rename renames the file based on the filename and the time
// the rename function is called
func (f *File) Rename() error {
	if !f.Exists() {
		return errors.New("Can't Rename / Move a file which doesn't exist!")
	}
	if f.IsDir() {
		return errors.New("Can't call rename on a directory")
	}
	ext := filepath.Ext(f.URL)
	newName := strings.TrimSuffix(f.URL, ext)

	newName += time.Now().Format(time.Stamp) + ext

	return f.File.Move(newName)
}

// Size returns the size of the file
func (f *File) Size() int64 {
	if !f.exists {
		return 0
	}
	return f.File.Size
}

// Deleted - If the file doesn't exist was it deleted
func (f *File) Deleted() bool {
	return f.deleted
}

// CreateDir creates a New Directory based on the non-existant syncer's name
func (f *File) CreateDir() error {
	return f.client.NewFolder(f.URL)
}

// StartMonitor starts Monitoring this syncer for changes (Dir's only)
func (f *File) StartMonitor(p *syncer.Profile) error {
	if !f.IsDir() {
		return errors.New("Can't start monitoring a non-directory")
	}

	if watching.has(p, f) {
		return nil
	}

	// Start watching, and sync all children of this folder
	children, err := f.Children()
	if err != nil {
		return err
	}

	// Trigger initial change event to make sure all
	// child folders are monitored recursively and all
	// files are in sync
	for i := range children {
		changeHandler(p, children[i])
	}

	watching.add(p, f)
	return nil
}

// StopMonitor stops Monitoring this syncer for changes (Dir's only)
func (f *File) StopMonitor(p *syncer.Profile) error {

	if !f.IsDir() {
		return errors.New("Can't stop monitoring a non-directory")
	}

	if !watching.has(p, f) {
		return nil
	}

	// Recursively stop watching all children dirs
	children, err := f.Children()
	if err != nil {
		return err
	}

	for i := range children {
		if children[i].IsDir() {
			children[i].StopMonitor(p)
		}
	}

	watching.remove(p, f)
	return nil
}
