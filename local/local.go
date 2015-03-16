// Copyright 2015 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package local

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"bitbucket.org/tshannon/freehold-sync/syncer"
)

// File is implements the syncer.Syncer interface
// for a file on the local machine
type File struct {
	filepath string
	info     os.FileInfo
	exists   bool
	deleted  bool
}

// New Returns a File from the local machine for use in syncing
func New(filePath string) (*File, error) {
	f := &File{
		filepath: filePath,
		exists:   true,
	}

	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			f.exists = false
		} else {
			//shouldn't happen
			panic(err)
		}
	} else {
		f.info = info
	}

	return f, nil
}

// ID is the unique identifier for a local file
func (f *File) ID() string {
	return f.filepath
}

// Modified is the date the file was last modified
func (f *File) Modified() time.Time {
	if !f.IsDir() {
		return f.info.ModTime()
	}
	return time.Time{}

}

// Children returns the child files for this given File, will only return
// records if the file is a Dir
func (f *File) Children() ([]syncer.Syncer, error) {
	if !f.IsDir() {
		return []syncer.Syncer{}, nil
	}

	file, err := os.Open(f.ID())
	defer file.Close()

	if err != nil {
		return nil, err
	}

	childNames, err := file.Readdirnames(0)
	if err != nil {
		return nil, err
	}

	children := make([]syncer.Syncer, 0, len(childNames))

	for i := range childNames {
		n, err := New(filepath.Join(f.ID(), childNames[i]))
		if err != nil {
			return nil, err
		}
		children = append(children, n)
	}

	return children, nil

}

// Open returns a readwritecloser for reading and writing to the file
func (f *File) Open() (io.ReadCloser, error) {
	var file *os.File
	var err error
	if !f.exists {
		return nil, os.ErrNotExist
	}
	file, err = os.Open(f.ID())

	if err != nil {
		return nil, err
	}
	return file, nil
}

// Write writes from the reader to the Syncer
func (f *File) Write(r io.ReadCloser, size int64) error {
	var wf *os.File
	var err error
	if f.exists {
		wf, err = os.Open(f.ID())

	} else {
		wf, err = os.Create(f.ID())
	}
	if err != nil {
		return err
	}

	written, err := io.Copy(wf, r)
	if err != nil {
		return nil
	}
	if written != size {
		return io.ErrShortWrite
	}
	return nil
}

// IsDir is whether or not the file is a directory
func (f *File) IsDir() bool {
	return f.info.IsDir()
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

	if f.IsDir() {
		//Remove watcher
		err := watching.remove(nil, f)
		if err != nil {
			return err
		}
	}

	return os.Remove(f.filepath)
}

// Rename renames the file based on the filename and the time
// the rename function is called
func (f *File) Rename() error {
	ext := filepath.Ext(f.filepath)
	newName := strings.TrimSuffix(f.filepath, ext)

	newName += time.Now().Format(time.Stamp) + ext

	return os.Rename(f.filepath, newName)
}

// Size returns the size of the file
func (f *File) Size() int64 {
	if !f.exists {
		return 0
	}
	return f.info.Size()
}

// Deleted - If the file doesn't exist was it deleted
func (f *File) Deleted() bool {
	return f.deleted
}

// CreateDir creates a New Directory based on the non-existant syncer's name
func (f *File) CreateDir() error {
	if f.exists {
		return errors.New("Can't create directory, name already exists")
	}
	return os.Mkdir(filepath.Base(f.filepath), 0777)
}

// StartMonitor starts Monitoring this syncer for changes (Dir's only), calls profile.Sync method on all changes, and initial startup
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

	return watching.add(p, f)
}

// StopMonitor stops Monitoring this syncer for changes
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

	return watching.remove(p, f)
}
