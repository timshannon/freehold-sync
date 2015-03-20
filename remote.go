// Copyright 2015 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"errors"
	"net/http"
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
}

func remoteRootGet(w http.ResponseWriter, r *http.Request) {
	respondJsend(w, &jsend{
		Status: statusSuccess,
		Data:   "/v1/file/",
	})
}

func remoteGet(w http.ResponseWriter, r *http.Request) {
	input := &dirListInput{}

	if errHandled(parseJSON(r, input), w) {
		return
	}

	dirPath := "/v1/file/"

	if input.DirPath != nil {
		dirPath = *input.DirPath
	}

	if strings.TrimSpace(dirPath) == "" {
		dirPath = "/v1/file/"
	}

	if input.Client.URL == nil || input.Client.User == nil || input.Client.Password == nil {
		errHandled(errors.New("Invalid input to retrieve a remote file.  You must provide a url, username, and password/token."), w)
		return
	}

	c, err := fh.NewFromClient(&http.Client{Timeout: httpTimeout}, *input.Client.URL, *input.Client.User, *input.Client.Password)
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
			dirList = append(dirList, children[i].ID())
		}
	}

	respondJsend(w, &jsend{
		Status: statusSuccess,
		Data:   dirList,
	})

}

func tokenGet(w http.ResponseWriter, r *http.Request) {
	input := &tokenInput{}
	if errHandled(parseJSON(r, input), w) {
		return
	}

	if input.Client.URL == nil || input.Client.User == nil || input.Client.Password == nil {
		errHandled(errors.New("Invalid input to generate a token.  You must provide a url, username, and password."), w)
		return
	}

	c, err := fh.NewFromClient(&http.Client{Timeout: httpTimeout}, *input.Client.URL, *input.Client.User, *input.Client.Password)
	if errHandled(err, w) {
		return
	}

	name := "Freehold-Sync"
	if input.Name != nil {
		name += ":" + *input.Name
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
