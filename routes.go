// Copyright 2015 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import "net/http"

var rootHandler *http.ServeMux

func init() {
	setupRoutes()
}

/*
	/: Root - Serve main page
	/profile:
		Get: Retrieve Sync Profiles
		Post: Post new Sync Profile
		Put: Update existing Sync Profile
	/profile/status:
		Get: Retrieve sync status of a specific sync profile
	/local:
		Get: Get local file Directory listings for Sync profile selection
	/remote:
		Get: Get remote file directory listings
	/remote/token:
		Get: Get token from user / password
	/log:
		Get: Get logs
*/

func setupRoutes() {
	rootHandler = http.NewServeMux()

	rootHandler.Handle("/", &methodHandler{
		get: rootGet,
	})

	//Logs
	rootHandler.Handle("/log/", &methodHandler{
		get: logGet,
	})

	//Local
	rootHandler.Handle("/local/", &methodHandler{
		get: localGet,
	})

	//Remote
	rootHandler.Handle("/remote/", &methodHandler{
		get: remoteGet,
	})
	rootHandler.Handle("/remote/token/", &methodHandler{
		get: tokenGet,
	})

	//Profiles
	rootHandler.Handle("/profile/", &methodHandler{
		get:    profileGet,
		post:   profilePost,
		put:    profilePut,
		delete: profileDelete,
	})

	rootHandler.Handle("/profile/status/", &methodHandler{
		get: profileStatusGet,
	})
}

type methodHandler struct {
	get    http.HandlerFunc
	post   http.HandlerFunc
	put    http.HandlerFunc
	delete http.HandlerFunc
}

func (m *methodHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if m.get == nil {
		m.get = http.NotFound
	}
	if m.post == nil {
		m.post = http.NotFound
	}
	if m.put == nil {
		m.put = http.NotFound
	}
	if m.delete == nil {
		m.delete = http.NotFound
	}
	switch r.Method {
	case "GET":
		m.get(w, r)
		return
	case "POST":
		m.post(w, r)
		return
	case "PUT":
		m.put(w, r)
		return
	case "DELETE":
		m.delete(w, r)
		return
	default:
		http.NotFound(w, r)
		return
	}
}
