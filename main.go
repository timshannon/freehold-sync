// Copyright 2014 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"

	"bitbucket.org/tshannon/config"
	"bitbucket.org/tshannon/freehold/data/store"
)

var flagPort int = 6080

func init() {
	flag.IntVar(&flagPort, "port", 6080, "Default Port to host freehold-sync webserver on.")

	//Capture program shutdown, to make sure everything shuts down nicely
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			if sig == os.Interrupt {
				halt("Freehold shutting down")
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

	fmt.Printf("Freehold is currently using the file %s for settings.\n", cfg.FileName())

}

func halt(msg string) {
	store.Halt()
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}
