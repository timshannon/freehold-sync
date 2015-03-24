// Copyright 2015 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	fh "bitbucket.org/tshannon/freehold-client"
	"bitbucket.org/tshannon/freehold-sync/remote"
)

type tokenInput struct {
	Name   *string `json:"name"`
	Client *client `json:"client"`
}

type client struct {
	URL      *string `json:"url"`
	User     *string `json:"user"`
	Password *string `json:"password"`
	Token    *string `json:"token"`
}

func remoteRootGet(w http.ResponseWriter, r *http.Request) {
	defaultPath := "/v1/file/"
	input := &dirListInput{}

	if errHandled(parseJSON(r, input), w) {
		return
	}

	c, err := remoteClient(input.Client)
	if errHandled(err, w) {
		return
	}

	_, err = remote.New(c, defaultPath)
	if errHandled(err, w) {
		return
	}

	respondJsend(w, &jsend{
		Status: statusSuccess,
		Data:   defaultPath,
	})
}

func remoteClient(input *client) (*fh.Client, error) {

	if input == nil || input.URL == nil || input.User == nil {
		return nil, errors.New("Invalid input to retrieve a remote file.  You must provide a url, username, and password/token.")
	}

	pass := ""

	if input.Password != nil && *input.Password != "" {
		pass = *input.Password
	} else if input.Token != nil && *input.Token != "" {
		pass = *input.Token
	}

	if input.Password == nil && input.Token == nil {
		return nil, errors.New("Invalid input to retrieve a remote file.  You must provide a password or a token.")
	}

	c, err := fh.NewFromClient(&http.Client{Timeout: httpTimeout}, *input.URL, *input.User, pass)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func remoteGet(w http.ResponseWriter, r *http.Request) {
	input := &dirListInput{}

	if errHandled(parseJSON(r, input), w) {
		return
	}

	dirPath := ""
	if input.DirPath != nil {
		dirPath = *input.DirPath
	}

	if strings.TrimSpace(dirPath) == "" {
		errHandled(errors.New("Invalid path!"), w)
		return
	}

	c, err := remoteClient(input.Client)
	if errHandled(err, w) {
		return
	}

	f, err := remote.New(c, dirPath)
	if errHandled(err, w) {
		return
	}

	if !f.Exists() {
		errHandled(errors.New("Path does not exist!"), w)
		return
	}

	if !f.IsDir() {
		errHandled(errors.New("Path is not a directory!"), w)
		return
	}

	children, err := f.Children()
	dirList := make([]string, 0, len(children))
	for i := range children {
		if children[i].IsDir() {
			uri, err := url.Parse(children[i].ID())
			if errHandled(err, w) {
				return
			}
			dirList = append(dirList, uri.Path)
		}
	}

	respondJsend(w, &jsend{
		Status: statusSuccess,
		Data:   dirList,
	})
}

func tokenPost(w http.ResponseWriter, r *http.Request) {
	input := &tokenInput{}
	if errHandled(parseJSON(r, input), w) {
		return
	}

	c, err := remoteClient(input.Client)
	if errHandled(err, w) {
		return
	}

	name := "Freehold-Sync"
	if input.Name != nil {
		name += ": " + *input.Name
	}

	t, err := c.NewToken(name, "", "", time.Time{})
	if errHandled(err, w) {
		return
	}

	respondJsend(w, &jsend{
		Status: statusSuccess,
		Data:   t,
	})
}
