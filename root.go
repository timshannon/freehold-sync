// Copyright 2015 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"net/http"
	"path"
)

func rootGet(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		//send index.html
		data, err := Asset("web/index.html")
		if errHandled(err, w) {
			return
		}

		w.Write(data)
		return
	}

	assetPath := path.Join("web", r.URL.Path)
	data, err := Asset(assetPath)

	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Write(data)
}
