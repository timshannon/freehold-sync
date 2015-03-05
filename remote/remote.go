// Copyright 2015 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package remote

import (
	"io"
	"net/http"
	"time"

	"bitbucket.org/tshannon/freehold-sync/sync"
)

type File struct {
	url    string
	client *http.Client
}

func New(url string) (sync.Syncer, error) {
}

func (f *File) Id() string {
	return f.url
}

func (f *File) Modified() time.Time {
}

func (f *File) Children() ([]sync.Syncer, error) {
}

func (f *File) Data() (io.ReadCloser, error) {
}

func (f *File) IsDir() bool {
}
