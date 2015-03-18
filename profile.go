// Copyright 2015 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"errors"
	"net/http"
	"strings"
)

/*profile:
Post: Post new Sync Profile
Put: Update existing Sync Profile
*/
func profileGet(w http.ResponseWriter, r *http.Request) {
	input := &profileStore{}

	if errHandled(parseJSON(r, input), w) {
		return
	}

	if strings.TrimSpace(input.ID) == "" {
		all, err := allProfiles()
		if errHandled(err, w) {
			return
		}
		respondJsend(w, &jsend{
			Status: statusSuccess,
			Data:   all,
		})
		return
	}

	profile, err := getProfile(input.ID)
	if errHandled(err, w) {
		return
	}

	respondJsend(w, &jsend{
		Status: statusSuccess,
		Data:   profile,
	})
}

func profilePost(w http.ResponseWriter, r *http.Request) {
	input := &profileStore{}

	if errHandled(parseJSON(r, input), w) {
		return
	}

	profile, err := newProfile(input.Name, input.Direction, input.ConflictResolution, input.ConflictDuration, input.Active,
		input.LocalPath, input.RemotePath, input.Client)
	if errHandled(err, w) {
		return
	}

	respondJsend(w, &jsend{
		Status: statusSuccess,
		Data:   profile,
	})

}

func profilePut(w http.ResponseWriter, r *http.Request) {
	input := &profileStore{}

	if errHandled(parseJSON(r, input), w) {
		return
	}

	//For a general API, I wouldn't want this, but since I am the consumer and
	// the sender, it's ok for now, but I may change it later.
	// The risks is that json parsed zero values would override profile values
	// if they aren't specified.  Ideally you'd have a whole separate profileStoreInput struct
	// with pointers that you can check for null, but to be honest I'm being lazy and having
	// 3 different structures to describe a profile seems a bit overkill.  I may change
	// this later.

	if errHandled(input.update(), w) {
		return
	}

	respondJsend(w, &jsend{
		Status: statusSuccess,
	})

}

func profileStatusGet(w http.ResponseWriter, r *http.Request) {
	input := &profileStore{}

	if errHandled(parseJSON(r, input), w) {
		return
	}

	if strings.TrimSpace(input.ID) == "" {
		errHandled(errors.New("No ID specified. You must specify a profile ID when getting a status."), w)
		return
	}

	profile, err := getProfile(input.ID)
	if errHandled(err, w) {
		return
	}

	respondJsend(w, &jsend{
		Status: statusSuccess,
		Data:   profile.status(),
	})
}

func profileDelete(w http.ResponseWriter, r *http.Request) {
	input := &profileStore{}

	if errHandled(parseJSON(r, input), w) {
		return
	}

	if strings.TrimSpace(input.ID) == "" {
		errHandled(errors.New("No ID specified. You must specify a profile ID."), w)
		return
	}

	profile, err := getProfile(input.ID)
	if errHandled(err, w) {
		return
	}
	if errHandled(profile.delete(), w) {
		return
	}

	respondJsend(w, &jsend{
		Status: statusSuccess,
	})
}
