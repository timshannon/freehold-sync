// Copyright 2015 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"bitbucket.org/tshannon/freehold-sync/datastore"
	"bitbucket.org/tshannon/freehold-sync/local"
	"bitbucket.org/tshannon/freehold-sync/remote"
	"bitbucket.org/tshannon/freehold-sync/syncer"
)

const dsName = "profiles.ds"

// profileStore is the structure of how profile
// information will be stored in a local datastore file
type profileStore struct {
	Name                    string   `json:"name"`
	Direction               int      `json:"direction"`
	ConflictResolution      int      `json:"conflictResolution"`
	Ignore                  []string `json:"ignore"`
	ConflictDurationSeconds int      `json:"conflictDurationSeconds"`
	LocalPath               string   `json:"localPath"`
	RemotePath              string   `json:"remotePath"`
	ID                      string   `json:"id"`
	Active                  bool     `json:"active"`
	Client                  *client  `json:"client"`
}

func newProfile(name string, direction, conflictResolution, conflictDurationSeconds int, active bool, ignore []string,
	localPath, remotePath string, remoteClient *client) (*profileStore, error) {
	ps := &profileStore{
		ConflictResolution:      conflictResolution,
		Direction:               direction,
		Name:                    name,
		Ignore:                  ignore,
		Active:                  active,
		LocalPath:               localPath,
		RemotePath:              remotePath,
		Client:                  remoteClient,
		ConflictDurationSeconds: conflictDurationSeconds,
	}

	_, err := ps.makeProfile()
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

func (p *profileStore) makeProfile() (*syncer.Profile, error) {
	if strings.TrimSpace(p.Name) == "" {
		return nil, errors.New("No Name specified for this Sync Profile")
	}
	if strings.TrimSpace(p.LocalPath) == "" {
		return nil, errors.New("Local path not set")
	}
	if strings.TrimSpace(p.RemotePath) == "" {
		return nil, errors.New("Remote path not set")
	}

	if p.Direction != syncer.DirectionBoth &&
		p.Direction != syncer.DirectionLocalOnly &&
		p.Direction != syncer.DirectionRemoteOnly {

		return nil, errors.New("Invalid sync profile direction")
	}

	if p.ConflictResolution != syncer.ConResOverwrite &&
		p.ConflictResolution != syncer.ConResRename {
		return nil, errors.New("Invalid sync profile conflict resolution")
	}

	var ignore []*regexp.Regexp

	//validate regex
	for i := range p.Ignore {
		rx, err := regexp.Compile(p.Ignore[i])
		if err != nil {
			return nil, fmt.Errorf("Invalid Regular expression: %s", err)
		}

		ignore = append(ignore, rx)
	}

	lFile, err := local.New(p.LocalPath)
	if err != nil {
		return nil, fmt.Errorf("Error accessing the local sync path: %s", err)
	}
	if !lFile.Exists() {
		return nil, fmt.Errorf("Local sync path does not exist!")
	}

	c, err := remoteClient(p.Client)
	if err != nil {
		return nil, err
	}
	rFile, err := remote.New(c, p.RemotePath)
	if err != nil {
		return nil, fmt.Errorf("Error accessing the remote sync path: %s", err)
	}

	if !rFile.Exists() {
		return nil, fmt.Errorf("Remote sync path does not exist!")
	}

	profile := &syncer.Profile{
		Name:               p.Name,
		Direction:          p.Direction,
		ConflictResolution: p.ConflictResolution,
		ConflictDuration:   time.Duration(p.ConflictDurationSeconds) * time.Second,
		Ignore:             ignore,
		Local:              lFile,
		Remote:             rFile,
	}

	p.ID = profile.ID()
	return profile, nil
}

func (p *profileStore) update() error {
	oldID := p.ID
	profile, err := p.makeProfile()
	if err != nil {
		return err
	}

	if oldID != "" && oldID != profile.ID() {
		//ID changed, check if an existing profile
		// is already syncing these paths
		_, err = getProfile(profile.ID())
		if err != datastore.ErrNotFound {
			return errors.New("A profile syncing these two locations already exist!")
		}
		// delete old profile
		err = deleteProfile(oldID)
		if err != nil {
			return err
		}
	}

	err = profile.Stop()
	if err != nil {
		return err
	}

	ds, err := datastore.Open(filepath.Join(dataDir, dsName))
	if err != nil {
		return err
	}
	err = ds.Put(p.ID, p)
	if err != nil {
		return err
	}

	if p.Active {
		return profile.Start()
	}
	return nil
}

func (p *profileStore) status() (int, string) {
	count := syncer.ProfileSyncCount(p.ID)
	if p.Active {
		if count > 0 {
			return count, "Syncing"
		}
		return 0, "Synchronized"
	}
	//not active
	if count > 0 {
		return count, "Stopping"
	}
	return 0, "Stopped"

}

func deleteProfile(ID string) error {
	ds, err := datastore.Open(filepath.Join(dataDir, dsName))
	if err != nil {
		return err
	}

	return ds.Delete(ID)
}

func (p *profileStore) delete() error {
	profile, _ := p.makeProfile()
	if profile != nil {
		profile.Stop()
	}
	return deleteProfile(p.ID)
}
