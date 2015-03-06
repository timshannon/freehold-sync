// Copyright 2015 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package local

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"bitbucket.org/tshannon/freehold-sync/sync"
)

// File is implements the sync.Syncer interface
// for a file on the local machine
type File struct {
	filepath string
	info     os.FileInfo
}

// New Returns a File from the local machine for use in syncing
func New(filePath string) (sync.Syncer, error) {

	info, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	return &File{
		filepath: filePath,
		info:     info,
	}, nil
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
func (f *File) Children() ([]sync.Syncer, error) {
	if !f.IsDir() {
		return []sync.Syncer{}, nil
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

	children := make([]sync.Syncer, 0, len(childNames))

	for i := range childNames {
		n, err := New(filepath.Join(f.ID(), childNames[i]))
		if err != nil {
			return nil, err
		}
		children = append(children, n)
	}

	return children, nil

}

// Data returns a ReadCloser for getting the data out of the file
func (f *File) Data() (io.ReadCloser, error) {
	file, err := os.Open(f.ID())

	if err != nil {
		return nil, err
	}
	return file, nil
}

// IsDir is whether or not the file is a directory
func (f *File) IsDir() bool {
	return f.info.IsDir()
}
