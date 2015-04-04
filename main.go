// Copyright 2015 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"bitbucket.org/tshannon/config"
	"bitbucket.org/tshannon/freehold-sync/datastore"
	"bitbucket.org/tshannon/freehold-sync/local"
	"bitbucket.org/tshannon/freehold-sync/log"
	"bitbucket.org/tshannon/freehold-sync/remote"
	"bitbucket.org/tshannon/freehold-sync/syncer"
)

var (
	flagPort    = 6080
	httpTimeout time.Duration
	server      *http.Server
	retry       chan *syncRetry // errors to retry when not syncing is idle
)

//TODO: System Tray: https://github.com/cratonica/trayhost

func init() {
	flag.IntVar(&flagPort, "port", 6080, "Default Port to host freehold-sync webserver on.")

	//Capture program shutdown, to make sure everything shuts down nicely
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			if sig == os.Interrupt {
				halt("Freehold-Sync shutting down")
			}
		}
	}()
	retry = make(chan *syncRetry, 100)
}

func main() {
	flag.Parse()

	settingPaths := config.StandardFileLocations("freehold-sync/settings.json")
	fmt.Println("Freehold-sync will use settings files in the following locations (in order of priority):")
	for i := range settingPaths {
		fmt.Println("\t", settingPaths[i])
	}
	cfg, err := config.LoadOrCreate(settingPaths...)
	if err != nil {
		halt(err.Error())
	}

	port := cfg.Int("port", flagPort)
	remotePolling := time.Duration(cfg.Int("remotePollingSeconds", 30)) * time.Second
	httpTimeout = time.Duration(cfg.Int("httpTimeoutSeconds", 30)) * time.Second

	fmt.Printf("Freehold is currently using the file %s for settings.\n", cfg.FileName())

	dataDir := filepath.Dir(cfg.FileName())
	err = datastore.Open(filepath.Join(dataDir, "sync.ds"))
	if err != nil {
		halt(err.Error())
	}

	server := &http.Server{
		Addr:    ":" + strconv.Itoa(port),
		Handler: rootHandler,
	}

	err = local.StartWatcher(localChanges)
	if err != nil {
		halt("Error starting up local file monitor: " + err.Error())
	}

	err = remote.StartWatcher(remoteChanges, remotePolling)
	if err != nil {
		halt("Error starting up remote file monitor: " + err.Error())
	}

	all, err := allProfiles()
	if err != nil {
		halt(err.Error())
	}

	retryPoll()

	for i := range all {
		if all[i].Active {
			prf, err := all[i].makeProfile()
			if err != nil {
				log.New(err.Error(), "Both")
			}
			err = prf.Start()
			if err != nil {
				log.New(err.Error(), "Both")
			}
		}
	}

	err = server.ListenAndServe()
	if err != nil {
		halt(err.Error())
	}

}

//TODO: capture sync errors and retry them when no profiles are synchronizing
// If they fail again, then log them as errors

func localChanges(p *syncer.Profile, s syncer.Syncer) {
	// get path relative to local profile
	rPath := path.Join(p.Remote.Path(p), s.Path(p))

	r, err := remote.New(p.Remote.(*remote.File).Client(), rPath)
	if err != nil {
		log.New(fmt.Sprintf("Error building remote syncer for local syncer %s Error: %s", s.ID(), err.Error()), local.LogType)
		return
	}
	err = p.Sync(s, r)
	if err != nil {
		fmt.Printf("Error with %s to %s retrying.  Error: %s\n", s.ID(), r.ID(), err)
		retry <- &syncRetry{
			profile:       p,
			local:         s,
			remote:        r,
			logType:       local.LogType,
			originalError: err,
		}
	}
}

func remoteChanges(p *syncer.Profile, s syncer.Syncer) {
	// get path relative to remote profile
	lPath := path.Join(p.Local.Path(p), s.Path(p))

	l, err := local.New(lPath)
	if err != nil {
		log.New(fmt.Sprintf("Error building local syncer for remote syncer %s Error: %s", s.ID(), err.Error()), remote.LogType)
		return
	}
	err = p.Sync(l, s)
	if err != nil {
		fmt.Printf("Error with %s to %s retrying.  Error: %s\n", s.ID(), l.ID(), err)
		retry <- &syncRetry{
			profile:       p,
			local:         l,
			remote:        s,
			logType:       remote.LogType,
			originalError: err,
		}
	}
}

type syncRetry struct {
	profile       *syncer.Profile
	local, remote syncer.Syncer
	logType       string
	originalError error
}

func retryPoll() {
	go func() {
		// while there are errors to retry, wait until the profiles are idle / not actively syncing, and
		// re-run the errors.  If they fail again, then log them.  This should clear up any order of operation issues
		// that my pop up due to user activity
		for i := range retry {
			for syncing := syncer.ProfileSyncCount(i.profile.ID()); syncing > 0; {
				time.Sleep(10 * time.Second)
			}
			fmt.Println("Retrying errors")
			//Set deleted
			l, err := local.New(i.local.ID())
			if err != nil {
				log.New(fmt.Sprintf("Error building local syncer %s for retying error: %s", l.ID(), err.Error()), local.LogType)
			}
			l.SetDeleted(i.local.Deleted())
			r, err := remote.New(i.remote.(*remote.File).Client(), i.remote.(*remote.File).URL)
			if err != nil {
				log.New(fmt.Sprintf("Error building remote syncer %s for retying error: %s", r.ID(), err.Error()), remote.LogType)
			}
			r.SetDeleted(i.remote.Deleted())
			err = i.profile.Sync(l, r)
			if err != nil {
				log.New(fmt.Sprintf("Error retrying sync error.  Local: %s Remote %s Original Error: %s Retry Error: %s", l.ID(), r.ID(), i.originalError, err), i.logType)
			}
		}
	}()
}

func halt(msg string) {
	time.Sleep(1 * time.Second)
	fmt.Fprintln(os.Stderr, msg)
	datastore.Close()
	close(retry)
	local.StopWatcher()
	remote.StopWatcher()
	os.Exit(1)
}
