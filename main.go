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
	"runtime"
	"strconv"
	"time"

	"bitbucket.org/tshannon/config"
	"bitbucket.org/tshannon/freehold-sync/datastore"
	"bitbucket.org/tshannon/freehold-sync/local"
	"bitbucket.org/tshannon/freehold-sync/log"
	"bitbucket.org/tshannon/freehold-sync/remote"
	"bitbucket.org/tshannon/freehold-sync/syncer"
	"bitbucket.org/tshannon/freehold-sync/trayhost"
)

var (
	flagPort     = 6080
	httpTimeout  time.Duration
	server       *http.Server
	retry        chan retrier
	flagSkipTray = true
)

func init() {
	flag.IntVar(&flagPort, "port", 6080, "Default Port to host freehold-sync webserver on.")
	flag.BoolVar(&flagSkipTray, "skipTray", false, "Whether or not to skip starting the system tray.")

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
	retry = make(chan retrier, 100)
}

func main() {
	flag.Parse()

	settingPaths := config.StandardFileLocations("freehold-sync/settings.json")
	fmt.Println("Freehold-Sync will use settings files in the following locations (in order of priority):")
	for i := range settingPaths {
		fmt.Println("\t", settingPaths[i])
	}
	cfg, err := config.LoadOrCreate(settingPaths...)
	if err != nil {
		halt(err.Error())
	}

	port := strconv.Itoa(cfg.Int("port", flagPort))
	remotePolling := time.Duration(cfg.Int("remotePollingSeconds", 30)) * time.Second
	httpTimeout = time.Duration(cfg.Int("httpTimeoutSeconds", 30)) * time.Second
	dataDir := filepath.Dir(cfg.FileName())

	fmt.Printf("Freehold-Sync is currently using the file %s for settings.\n", cfg.FileName())

	if flagSkipTray {
		startServer(port, dataDir, remotePolling)
	} else {
		runtime.LockOSThread()

		go func() {
			trayhost.SetURL("http://localhost:" + port)
			startServer(port, dataDir, remotePolling)
		}()

		trayhost.EnterLoop("Freehold-Sync", getIconData())
		//tray is exited
		halt("Freehold-Sync shutting down")
	}
}

func startServer(port, dataDir string, remotePolling time.Duration) {
	err := datastore.Open(filepath.Join(dataDir, "sync.ds"))
	if err != nil {
		halt(err.Error())
	}

	server := &http.Server{
		Addr:    ":" + port,
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
				log.New(fmt.Sprintf("Error starting profile: %s", err.Error()), "Both")
				continue
			}
			err = prf.Start()
			if err != nil {
				log.New(fmt.Sprintf("Error starting profile: %s", err.Error()), "Both")
				continue
			}
		}
	}

	err = server.ListenAndServe()
	if err != nil {
		halt(err.Error())
	}

}

func localChanges(p *syncer.Profile, s syncer.Syncer) {
	// get path relative to local profile
	rPath := path.Join(p.Remote.Path(p), filepath.ToSlash(s.Path(p)))

	r, err := remote.New(p.Remote.(*remote.File).Client(), rPath)
	if err != nil {
		log.New(fmt.Sprintf("Error building remote syncer for local syncer %s Error: %s", s.ID(), err.Error()), local.LogType)
		return
	}

	err = p.Sync(s, r)
	if err != nil {
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
	lPath := filepath.Join(p.Local.Path(p), s.Path(p))

	l, err := local.New(lPath)
	if err != nil {
		log.New(fmt.Sprintf("Error building local syncer for remote syncer %s Error: %s", s.ID(), err.Error()), remote.LogType)
		return
	}
	err = p.Sync(l, s)
	if err != nil {
		retry <- &syncRetry{
			profile:       p,
			local:         l,
			remote:        s,
			logType:       remote.LogType,
			originalError: err,
		}
	}
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
