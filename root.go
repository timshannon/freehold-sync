// Copyright 2015 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import "net/http"

func rootGet(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(webpage))
}
