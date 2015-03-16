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

	"bitbucket.org/tshannon/config"
	"bitbucket.org/tshannon/freehold-sync/local"
	"bitbucket.org/tshannon/freehold-sync/log"
	"bitbucket.org/tshannon/freehold-sync/remote"
	"bitbucket.org/tshannon/freehold/data/store"
)

//TODO: Use https://godoc.org/gopkg.in/fsnotify.v1

var flagPort = 6080

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
	//TODO: Client timeouts
	//TODO: Remote Polling interval?

	fmt.Printf("Freehold is currently using the file %s for settings.\n", cfg.FileName())

	dataDir := filepath.Dir(cfg.FileName()) // where log and remote ds will be stored

	log.DSDir = dataDir

	s := &http.Server{
		Addr:    ":" + strconv.Itoa(port),
		Handler: rootHandler,
	}

	err = local.StartWatcher() //TODO: Change handler
	if err != nil {
		halt("Error starting up local file monitor: " + err.Error())
	}

	err = remote.StartWatcher() //TODO: Change handler
	if err != nil {
		halt("Error starting up remote file monitor: " + err.Error())
	}

	err = s.ListenAndServe()
	if err != nil {
		halt(err.Error())
	}

}

func halt(msg string) {
	store.Halt()
	local.StopWatcher()
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}
