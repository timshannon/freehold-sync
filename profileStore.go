// Copyright 2015 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import "bitbucket.org/tshannon/freehold-sync/syncer"

const dsName = "profiles.ds"

// profileStore is the structure of how profile
// information will be stored in a local datastore file
type profileStore struct {
	*syncer.Profile
	Ignore     []string `json:"ignore"`
	LocalPath  string   `json:"localPath"`
	RemotePath string   `json:"remotePath"`
}
