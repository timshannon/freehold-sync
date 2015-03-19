// Copyright 2015 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"bytes"
	"net/http"
	"path"
	"time"
)

func rootGet(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		//send index.html
		writeAsset(w, r, "web/index.html")
		return
	}

	writeAsset(w, r, path.Join("web", r.URL.Path))
}

func writeAsset(w http.ResponseWriter, r *http.Request, asset string) {
	data, err := Asset(asset)

	if err != nil {
		http.NotFound(w, r)
		return
	}

	br := bytes.NewReader(data)

	http.ServeContent(w, r, r.URL.Path, time.Time{}, br)
}
