// Copyright 2015 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package remote

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"bitbucket.org/tshannon/freehold-sync/sync"
)

//TODO: Move to freehold-client

// File is implements the sync.Syncer interface
// for a remote file at a freehold instance
type File struct {
	client      *FreeholdClient
	url         string
	propertyURL string
	modified    time.Time
	isDir       bool
}

// FreeholdClient handles the credentials and
// access to a given Freehold Instance
type FreeholdClient struct {
	rootURL       string
	username      string
	token         string
	skipSSLVerify bool
	client        *http.Client
}

// NewClient Returns a new FreeholdClient for access to
// a given freehold instance
func NewClient(rootURL, username, token string, skipSSLVerify bool) *FreeholdClient {
	tlsCfg := &tls.Config{InsecureSkipVerify: skipSSLVerify}
	return &FreeholdClient{
		rootURL:  rootURL,
		username: username,
		token:    token,
		client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsCfg,
			},
		},
	}
}

func (c *FreeholdClient) newRequest(method, url string) (*http.Request, error) {
	req, err := http.NewRequest(method, url, nil)

	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(c.username, c.token)
	return req, nil
}

// File Returns a File from a freehold instance for use in syncing
func (c *FreeholdClient) File(urlPath string) (sync.Syncer, error) {
	urlPath = strings.TrimSuffix(urlPath, "/")

	uri, err := url.Parse(c.rootURL)
	if err != nil {
		return nil, err
	}

	uri.Path = propertyPath(urlPath)

	f := &File{}

	req, err := c.newRequest("GET", uri.String())
	if err != nil {
		return nil, err
	}

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Request for file %s returned a status %d", f.propertyURL, res.StatusCode)
	}

	decoder := json.NewDecoder(res.Body)
	defer res.Body.Close()

	response := &fhResponse{}
	err = decoder.Decode(response)
	if err != nil {
		return nil, err
	}

	f.propertyURL = uri.String()
	uri.Path = urlPath
	f.url = uri.String()
	f.client = c

	f.modified, err = time.Parse(time.RFC3339, f.ModifiedDate)
	return f, nil
}

// ID returns the unique Identifier for a remote file, in this case the URL
func (f *File) ID() string {
	return f.url
}

// Modified returns the last time the file was modified
func (f *File) Modified() time.Time {
	return f.modified
}

// Children returns the child files for this given File, will only return
// records if the file is a Dir
func (f *File) Children() ([]sync.Syncer, error) {
	//TODO
	return nil, nil
}

// Data returns a ReadCloser for getting the data out of the file
func (f *File) Data() (io.ReadCloser, error) {
	//TODO
	return nil, nil
}

// IsDir is whether or not the file is a directory
func (f *File) IsDir() bool {
	return f.Dir
}

func propertyPath(url string) string {
	//Does not support Application paths
	if strings.Index(url, "/") == 0 {
		url = url[1:]
	}

	s := strings.SplitN(url, "/", 2)

	return path.Join(s[0], "properties", s[1])
}
