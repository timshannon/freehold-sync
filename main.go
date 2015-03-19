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
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"bitbucket.org/tshannon/config"
	"bitbucket.org/tshannon/freehold-sync/local"
	"bitbucket.org/tshannon/freehold-sync/log"
	"bitbucket.org/tshannon/freehold-sync/remote"
	"bitbucket.org/tshannon/freehold-sync/syncer"
	"bitbucket.org/tshannon/freehold/data/store"
)

var (
	flagPort    = 6080
	dataDir     = ""
	httpTimeout time.Duration
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

	dataDir = filepath.Dir(cfg.FileName()) // where log and remote ds will be stored

	log.DSDir = dataDir

	s := &http.Server{
		Addr:    ":" + strconv.Itoa(port),
		Handler: rootHandler,
	}

	err = local.StartWatcher(localChanges)
	if err != nil {
		halt("Error starting up local file monitor: " + err.Error())
	}

	err = remote.StartWatcher(remoteChanges, dataDir, remotePolling)
	if err != nil {
		halt("Error starting up remote file monitor: " + err.Error())
	}

	err = s.ListenAndServe()
	if err != nil {
		halt(err.Error())
	}

}

func localChanges(p *syncer.Profile, s syncer.Syncer) {
	syncing.start(p.ID())
	defer syncing.stop(p.ID())
	remotePath := strings.TrimPrefix(s.ID(), p.Local.ID()) // get path relative to profile
	r, err := remote.New(p.Remote.(*remote.File).Client(), remotePath)
	if err != nil {
		log.New(fmt.Sprintf("Error building remote syncer: %s", err.Error()), remote.LogType)
		return
	}
	p.Sync(s, r)
}

func remoteChanges(p *syncer.Profile, s syncer.Syncer) {
	syncing.start(p.ID())
	defer syncing.stop(p.ID())

	localPath := strings.TrimPrefix(s.ID(), p.Remote.ID()) // get path relative to profile
	l, err := local.New(localPath)
	if err != nil {
		log.New(fmt.Sprintf("Error building local syncer: %s", err.Error()), remote.LogType)
		return
	}
	p.Sync(l, s)
}

func halt(msg string) {
	store.Halt()
	local.StopWatcher()
	remote.StopWatcher()
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}
