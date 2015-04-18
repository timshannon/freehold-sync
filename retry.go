// Copyright 2015 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"fmt"
	"time"

	"bitbucket.org/tshannon/freehold-sync/local"
	"bitbucket.org/tshannon/freehold-sync/log"
	"bitbucket.org/tshannon/freehold-sync/remote"
	"bitbucket.org/tshannon/freehold-sync/syncer"
)

// retrier is for retrying errors
type retrier interface {
	//profile() *syncer.Profile
	retry() error
}

func retryPoll() {
	//TODO: Update Sync Status if there are retries to run
	go func() {
		// while there are errors to retry, wait until the profiles are idle / not actively syncing, and
		// re-run the errors.  If they fail again, then log them.  This should clear up any order of operation issues
		// that my pop up due to user activity
		for r := range retry {
			remote.PauseWatcher()
			err := r.retry()
			if err != nil {
				retry <- r
			}
			remote.ResumeWatcher()
		}
	}()
}

type syncRetry struct {
	profile       *syncer.Profile
	local, remote syncer.Syncer
	logType       string
	originalError error
	retryCount    int
}

func (s *syncRetry) retry() error {
	time.Sleep(5 * time.Second)
	//Set deleted
	l, err := local.New(s.local.ID())
	if err != nil {
		log.New(fmt.Sprintf("Error building local syncer %s for retying error: %s", s.local.ID(), err.Error()), local.LogType)
	}
	l.SetDeleted(s.local.Deleted())
	r, err := remote.New(s.remote.(*remote.File).Client(), s.remote.(*remote.File).URL)
	if err != nil {
		log.New(fmt.Sprintf("Error building remote syncer %s for retying error: %s", s.remote.ID(), err.Error()), remote.LogType)
	}
	r.SetDeleted(s.remote.Deleted())

	err = s.profile.Sync(l, r)
	if err != nil {
		s.retryCount++
		if s.retryCount >= 3 {
			//after 3 attempts log error and don't retry again
			log.New(fmt.Sprintf("Error with syncing %s and %s retrying.  Error: %s\n", r.ID(), l.ID(), err), s.logType)
			return nil
		}
	}
	return err
}
