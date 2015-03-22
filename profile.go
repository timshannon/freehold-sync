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

	profile, err := newProfile(input.Name, input.Direction, input.ConflictResolution, input.ConflictDurationSeconds, input.Active,
		input.Ignore, input.LocalPath, input.RemotePath, input.Client)
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

	status, count := profile.status()

	respondJsend(w, &jsend{
		Status: statusSuccess,
		Data:   map[string]interface{}{"status": status, "count": count},
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
