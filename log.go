// Copyright 2015 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"net/http"

	"bitbucket.org/tshannon/freehold-sync/log"
)

type logInput struct {
	Type string `json:"type"`
	Page int    `json:"page"`
}

func logGet(w http.ResponseWriter, r *http.Request) {

	input := &logInput{}
	if errHandled(parseJSON(r, input), w) {
		return
	}

	logs, err := log.Get(input.Type, input.Page)
	if errHandled(err, w) {
		return
	}

	respondJsend(w, &jsend{
		Status: statusSuccess,
		Data:   logs,
	})

}
