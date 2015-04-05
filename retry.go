// Copyright 2015 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"fmt"

	"bitbucket.org/tshannon/freehold-sync/local"
	"bitbucket.org/tshannon/freehold-sync/log"
	"bitbucket.org/tshannon/freehold-sync/remote"
	"bitbucket.org/tshannon/freehold-sync/syncer"
)

// retrier is for retrying errors
type retrier interface {
	retry() error
}

type syncRetry struct {
	profile       *syncer.Profile
	local, remote syncer.Syncer
	logType       string
	originalError error
}

func (s *syncRetry) retry() error {
	//Set deleted
	l, err := local.New(s.local.ID())
	if err != nil {
		log.New(fmt.Sprintf("Error building local syncer %s for retying error: %s", l.ID(), err.Error()), local.LogType)
	}
	l.SetDeleted(s.local.Deleted())
	r, err := remote.New(s.remote.(*remote.File).Client(), s.remote.(*remote.File).URL)
	if err != nil {
		log.New(fmt.Sprintf("Error building remote syncer %s for retying error: %s", r.ID(), err.Error()), remote.LogType)
	}
	r.SetDeleted(s.remote.Deleted())
	return s.profile.Sync(l, r)
}

func retryPoll() {
	go func() {
		// while there are errors to retry, wait until the profiles are idle / not actively syncing, and
		// re-run the errors.  If they fail again, then log them.  This should clear up any order of operation issues
		// that my pop up due to user activity
		for r := range retry {
			fmt.Println("Retrying errors")
			err := r.retry()
			if err != nil {
				retry <- r
			}
		}
	}()
}
