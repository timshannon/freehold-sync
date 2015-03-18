// Copyright 2015 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	fh "bitbucket.org/tshannon/freehold-client"

	"bitbucket.org/tshannon/freehold-sync/datastore"
	"bitbucket.org/tshannon/freehold-sync/local"
	"bitbucket.org/tshannon/freehold-sync/remote"
	"bitbucket.org/tshannon/freehold-sync/syncer"
)

const dsName = "profiles.ds"

// profileStore is the structure of how profile
// information will be stored in a local datastore file
type profileStore struct {
	*syncer.Profile
	Ignore     []string `json:"ignore"`
	LocalPath  string   `json:"localPath"`
	RemotePath string   `json:"remotePath"`
	ID         string   `json:"id"`
	Active     bool     `json:"active"`
	Client     *client  `json:"client"`
}

func newProfile(name string, direction, conflictResolution int, conflictDuration time.Duration, active bool,
	localPath, remotePath string, remoteClient *client) (*profileStore, error) {
	ps := &profileStore{
		Profile: &syncer.Profile{
			Name:               name,
			Direction:          direction,
			ConflictResolution: conflictResolution,
			ConflictDuration:   conflictDuration,
		},
		Active:     active,
		LocalPath:  localPath,
		RemotePath: remotePath,
		Client:     remoteClient,
	}

	err := ps.prep()
	if err != nil {
		return nil, err
	}

	_, err = getProfile(ps.ID)
	if err != nil && err != datastore.ErrNotFound {
		return nil, err
	}

	if err != datastore.ErrNotFound {
		return nil, errors.New("A profile syncing these two locations already exist!")
	}

	err = ps.update()
	if err != nil {
		return nil, err
	}
	return ps, nil
}

func getProfile(id string) (*profileStore, error) {
	ds, err := datastore.Open(filepath.Join(dataDir, dsName))
	if err != nil {
		return nil, err
	}
	ps := &profileStore{}
	err = ds.Get(id, ps)
	if err != nil {
		return nil, err
	}

	return ps, nil
}

func allProfiles() ([]*profileStore, error) {
	ds, err := datastore.Open(filepath.Join(dataDir, dsName))
	if err != nil {
		return nil, err
	}

	min, err := ds.Min()
	if err != nil {
		return nil, err
	}
	max, err := ds.Max()
	if err != nil {
		return nil, err
	}

	iter, err := ds.Iter(min, max)
	if err != nil {
		return nil, err
	}

	var all []*profileStore

	for iter.Next() {
		if iter.Err() != nil {
			return nil, iter.Err()
		}

		p := &profileStore{}
		err = json.Unmarshal(iter.Value(), p)
		if err != nil {
			return nil, err
		}
		all = append(all, p)
	}

	return all, nil
}

func (p *profileStore) prep() error {
	if strings.TrimSpace(p.LocalPath) == "" {
		return errors.New("Local path not set")
	}
	if strings.TrimSpace(p.RemotePath) == "" {
		return errors.New("Remote path not set")
	}

	if strings.TrimSpace(p.Name) == "" {
		return errors.New("No Name specified for this Sync Profile")
	}

	if p.Direction != syncer.DirectionBoth ||
		p.Direction != syncer.DirectionLocalOnly ||
		p.Direction != syncer.DirectionRemoteOnly {

		return errors.New("Invalid sync profile direction")
	}

	if p.ConflictResolution != syncer.ConResOverwrite ||
		p.ConflictResolution != syncer.ConResRename {
		return errors.New("Invalid sync profile conflict resolution")
	}

	//validate regex
	for i := range p.Ignore {
		rx, err := regexp.Compile(p.Ignore[i])
		if err != nil {
			return fmt.Errorf("Invalid Regular expression: %s", err)
		}

		p.Profile.Ignore = append(p.Profile.Ignore, rx)
	}

	lFile, err := local.New(p.LocalPath)
	if err != nil {
		return fmt.Errorf("Error accessing the local sync path: %s", err)
	}
	p.Profile.Local = lFile

	c, err := fh.NewFromClient(&http.Client{Timeout: httpTimeout}, *p.Client.URL, *p.Client.User, *p.Client.Password)
	if err != nil {
		return err
	}
	rFile, err := remote.New(c, p.RemotePath)
	if err != nil {
		return fmt.Errorf("Error accessing the remote sync path: %s", err)
	}

	p.Profile.Remote = rFile

	p.ID = p.Profile.ID()

	return nil
}

func (p *profileStore) update() error {
	err := p.prep()
	if err != nil {
		return err
	}

	err = p.Profile.Stop()
	if err != nil {
		return err
	}

	ds, err := datastore.Open(filepath.Join(dataDir, dsName))
	if err != nil {
		return err
	}
	err = ds.Put(p.Profile.ID(), p)
	if err != nil {
		return err
	}

	if p.Active {
		return p.Profile.Start()
	}
	return nil
}

func (p *profileStore) status() string {
	if p.Active {
		//if active and not syncing
		// status = "Synchronized"
		//if active and syncing
		// status = "Syncing
	}
	//not active

	//if not active and syncing
	// status = "Stopping"
	//if not active and not syncing
	// status = "Stopped"
	return "TODO"
}

func (p *profileStore) delete() error {
	ds, err := datastore.Open(filepath.Join(dataDir, dsName))
	if err != nil {
		return err
	}

	p.prep()
	if p.Profile.Local != nil && p.Profile.Remote != nil {
		p.Profile.Stop()
	}

	return ds.Delete(p.ID)
}
