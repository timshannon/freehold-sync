// Copyright 2015 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package remote

import (
	"errors"
	"fmt"
	"io"
	"net/url"
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
	file         *fh.File
	client       *fh.Client
	Name         string    `json:"name"`
	FullURL      string    `json:"fullUrl"`
	URL          string    `json:"path"`
	ModifiedTime time.Time `json:"modified"`
	deleted      bool
	exists       bool
}

// New Returns a File from the remote instance for use in syncing
func New(client *fh.Client, filePath string) (*File, error) {
	if client == nil {
		return nil, errors.New("Can't retrieve a file with a nil client")
	}

	f := newEmptyFile(client, filePath)

	file, err := client.GetFile(filePath)
	if fh.IsNotFound(err) {
		//Check if deleted
		in, err := f.inRemoteDS()
		if err != nil {
			return nil, err
		}
		//If file no longer exists, but
		// is in the remote DS list, then it
		// was deleted
		f.deleted = in
		return f, nil
	}
	if err != nil {
		return nil, err
	}

	f = newFromFile(client, file)

	return f, nil
}

func newFromFile(client *fh.Client, file *fh.File) *File {
	eURL, _ := url.QueryUnescape(file.FullURL())
	f := &File{
		exists:       true,
		deleted:      false,
		client:       client,
		Name:         file.Name,
		URL:          file.URL,
		FullURL:      eURL,
		ModifiedTime: file.ModifiedTime(),
		file:         file,
	}
	return f
}

func newEmptyFile(client *fh.Client, filePath string) *File {
	uri := client.RootURL()
	uri.Path = filePath
	eURL, _ := url.QueryUnescape(uri.String())

	f := &File{
		exists:  false,
		client:  client,
		Name:    filepath.Base(filePath),
		URL:     filePath,
		FullURL: eURL,
	}
	return f
}

// Client is the freehold client used to retrieve this file
func (f *File) Client() *fh.Client {
	return f.client
}

// ID is the unique identifier for a remote file
// the full URL of the file
func (f *File) ID() string {
	return f.FullURL
}

// Path is the path relative to the passed in profile
// if path is root to the profile, then return the full path
// without the domain
func (f *File) Path(p *syncer.Profile) string {
	if f.ID() == p.Remote.ID() {
		return f.URL
	}
	return strings.TrimPrefix(f.URL, p.Remote.Path(p))
}

// Modified is the date the file was last modified
func (f *File) Modified() time.Time {
	if !f.IsDir() && f.exists {
		return f.ModifiedTime
	}
	return time.Time{}
}

// Children returns the child files for this given File, will only return
// records if the file is a Dir
func (f *File) Children() ([]*File, error) {
	if !f.exists {
		return nil, nil
	}
	children, err := f.file.Children()
	if err != nil {
		return nil, err
	}
	syncers := make([]*File, len(children))

	for i := range children {
		syncers[i] = newFromFile(f.Client(), children[i])
	}
	return syncers, nil
}

// Open returns a ReadWriteCloser for reading, and writing data to the file
func (f *File) Open() (io.ReadCloser, error) {
	return f, nil
}

// Read reads the data out of the remote file
func (f *File) Read(p []byte) (n int, err error) {
	if !f.exists {
		return 0, fmt.Errorf("Can't read file %s , because it doesn't exist.")
	}
	return f.file.Read(p)
}

// Close closes an open file reader
func (f *File) Close() error {
	if !f.exists {
		return fmt.Errorf("Can't close file %s , because it doesn't exist.")
	}
	return f.file.Close()
}

// Write writes from the reader to the Syncer
func (f *File) Write(r io.ReadCloser, size int64, modTime time.Time) error {
	if f.IsDir() {
		return errors.New("Can't write a directory with this method")
	}

	//ignore  events for this change
	ignore.add(f.ID())
	defer ignore.remove(f.ID())
	var err error
	if f.exists {
		err = f.file.Delete()
		if err != nil && !fh.IsNotFound(err) {
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

	f.file = newFile

	f.exists = true
	f.deleted = false
	return r.Close()
}

// IsDir is whether or not the file is a directory
func (f *File) IsDir() bool {
	if !f.exists {
		return false
	}
	return f.file.IsDir
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

	//ignore  events for this change
	ignore.add(f.ID())
	defer ignore.remove(f.ID())

	if f.IsDir() {
		//Remove monitor
		err := f.stopWatcherRecursive(nil)
		if err != nil {
			return err
		}
	} else {
		err := deleteRemoteFileFromDS(f.ID())
		if err != nil {
			return err
		}
	}

	err := f.file.Delete()
	if err != nil && !fh.IsNotFound(err) {
		return err
	}
	return nil
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

	//ignore  events for this change
	ignore.add(f.ID())
	defer ignore.remove(f.ID())
	ext := filepath.Ext(f.file.URL)
	newName := strings.TrimSuffix(f.file.URL, ext)

	newName += time.Now().Format(time.Stamp) + ext

	return f.file.Move(newName)
}

// Size returns the size of the file
func (f *File) Size() int64 {
	if !f.exists {
		return 0
	}
	return f.file.Size
}

// Deleted - If the file doesn't exist was it deleted
func (f *File) Deleted() bool {
	return f.deleted
}

// SetDeleted is for setting the deleted value
func (f *File) SetDeleted(deleted bool) {
	f.deleted = deleted
}

// CreateDir creates a New Directory based on the non-existant syncer's name
func (f *File) CreateDir() (syncer.Syncer, error) {
	//ignore  events for this change
	ignore.add(f.ID())
	defer ignore.remove(f.ID())

	err := f.client.NewFolder(f.URL)
	if err != nil {
		if !strings.Contains(err.Error(), "Folder already exists") {
			return nil, err
		}
	}

	return New(f.client, f.URL)
}

// StartMonitor starts Monitoring this syncer for changes (Dir's only)
func (f *File) StartMonitor(p *syncer.Profile) error {
	if !f.IsDir() {
		return errors.New("Can't start monitoring a non-directory")
	}

	if watching.has(p, f) {
		return nil
	}

	// Start watching, and check for current differences
	// if folder hasn't been watched yet, then all
	// files will be checked
	diff, err := f.differences()
	if err != nil {
		return err
	}

	// Trigger initial change event to make sure all
	// child folders are monitored recursively and all
	// files are in sync
	for i := range diff {
		go func(s syncer.Syncer) {
			changeHandler(p, s)
		}(diff[i])
	}

	watching.add(p, f)
	return nil
}

// StopMonitor stops Monitoring this syncer for changes (Dir's only)
func (f *File) StopMonitor(p *syncer.Profile) error {
	// Recursively stop watching all children dirs
	return f.stopWatcherRecursive(p)
}

func (f *File) stopWatcherRecursive(p *syncer.Profile) error {
	// Recursively stop watching all children dirs
	children, err := f.Children()
	if err != nil {
		return err
	}

	for i := range children {
		if children[i].IsDir() {
			err = children[i].stopWatcherRecursive(p)
			if err != nil {
				return err
			}
		}
	}
	watching.remove(p, f)
	deleteRemoteFileFromDS(f.ID())
	f.removeFromRemoteDS()
	return nil
}

func (f *File) removeFromRemoteDS() error {
	err := datastore.Delete(bucket, f.ID())
	if err == datastore.ErrNotFound {
		return nil
	}
	return err
}

func (f *File) inRemoteDS() (bool, error) {
	dsFiles := make([]*File, 0, 1)
	parent := filepath.Dir(strings.TrimRight(f.ID(), "/"))

	err := datastore.Get(bucket, parent, &dsFiles)
	if err == datastore.ErrNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	for i := range dsFiles {
		if dsFiles[i].ID() == f.ID() {
			return true, nil
		}
	}

	return false, nil
}
