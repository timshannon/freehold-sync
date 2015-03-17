// Copyright 2015 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"errors"
	"net/http"
	"os/user"

	"bitbucket.org/tshannon/freehold-sync/local"
)

type dirListInput struct {
	DirPath *string `json:"dirPath"`
	Client  *client `json:"client"`
}

func localGet(w http.ResponseWriter, r *http.Request) {
	input := &dirListInput{}

	if errHandled(parseJSON(r, input), w) {
		return
	}

	usr, err := user.Current()
	if errHandled(err, w) {
		return
	}

	//Default / start browsing at user home dir
	dirPath := usr.HomeDir

	if input.DirPath != nil {
		dirPath = *input.DirPath
	}

	f, err := local.New(dirPath)
	if errHandled(err, w) {
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
