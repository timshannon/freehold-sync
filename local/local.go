// Copyright 2015 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package local

import (
	"io"
	"os"
	"time"

	"bitbucket.org/tshannon/freehold-sync/sync"
)

type File struct {
	filepath string
	info     os.FileInfo
}

func New(filepath string) (sync.Syncer, error) {

	info, err := os.Stat(filepath)
	if err != nil {
		return nil, err
	}

	return &File{
		filepath: filepath,
		info:     info,
	}, nil
}

func (f *File) Id() string {
	return f.filepath
}

func (f *File) Modified() time.Time {
	return f.info.ModTime()

}

func (f *File) Children() ([]sync.Syncer, error) {
	if !f.IsDir() {
		return []sync.Syncer{}, nil
	}

	file, err := os.Open(f.Id())
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
		n, err := New(childNames[i])
		if err != nil {
			return nil, err
		}
		children = append(children, n)
	}

	return children, nil

}

func (f *File) Data() (io.ReadCloser, error) {
	file, err := os.Open(f.Id())

	if err != nil {
		return nil, err
	}
	return file, nil
}

func (f *File) IsDir() bool {
	return f.info.IsDir()
}
