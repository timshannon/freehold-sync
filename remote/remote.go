// Copyright 2015 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package remote

import (
	"io"
	"path/filepath"
	"strings"
	"time"

	fh "bitbucket.org/tshannon/freehold-client"
	"bitbucket.org/tshannon/freehold-sync/sync"
)

// File is implements the sync.Syncer interface
// for a file on the Remote machine
type File struct {
	*fh.File
	client  *fh.Client
	deleted bool
	exists  bool
}

// New Returns a File from the remote instance for use in syncing
func New(client *fh.Client, filePath string) (sync.Syncer, error) {
	f := &File{
		exists: false,
		client: client,
	}

	file, err := client.GetFile(filePath)
	if fh.IsNotFound(err) {
		//TODO: Check if deleted based on local list of previously known files
		// this will have to do until we have server side file versioning
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
func (f *File) Children() ([]sync.Syncer, error) {
	if !f.exists {
		return nil, nil
	}
	children, err := f.File.Children()
	if err != nil {
		return nil, err
	}
	syncers := make([]sync.Syncer, len(children))

	for i := range children {
		syncers[i] = &File{
			File:    children[i],
			deleted: false,
		}
	}
	return syncers, nil
}

// Open returns a ReadWriteCloser for reading, and writing data to the file
func (f *File) Open() (io.ReadCloser, error) {
	return f, nil
}

// Write writes from the reader to the Syncer
func (f *File) Write(r io.ReadCloser, size int64) error {
	var err error
	if f.exists {
		return f.Update(r, size)
	}
	dest := &fh.File{
		URL:   filepath.Dir(f.URL),
		Name:  filepath.Base(filepath.Dir(f.URL)),
		IsDir: true,
	}
	newFile, err := f.client.UploadFromReader(f.Name, r, size, dest)
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

	//TODO: Delete from local DS
	return f.File.Delete()
}

// Rename renames the file based on the filename and the time
// the rename function is called
func (f *File) Rename() error {
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
