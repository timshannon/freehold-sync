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
	"sync"
	"time"

	"bitbucket.org/tshannon/freehold-sync/datastore"
	"bitbucket.org/tshannon/freehold-sync/local"
	"bitbucket.org/tshannon/freehold-sync/remote"
	"bitbucket.org/tshannon/freehold-sync/syncer"
)

const dsName = "profiles.ds"

var syncing profilesSyncing

// profileStore is the structure of how profile
// information will be stored in a local datastore file
type profileStore struct {
	*syncer.Profile
	Ignore                  []string `json:"ignore"`
	ConflictDurationSeconds int      `json:"conflictDurationSeconds"`
	LocalPath               string   `json:"localPath"`
	RemotePath              string   `json:"remotePath"`
	ID                      string   `json:"id"`
	Active                  bool     `json:"active"`
	Client                  *client  `json:"client"`
}

//could expand this to track individual files, but we'll just
// have a sync count for now
type profilesSyncing struct {
	sync.RWMutex
	profile map[string]int
}

func (ps *profilesSyncing) start(ID string) {
	ps.Lock()
	defer ps.Unlock()
	count := ps.profile[ID]
	count++
	ps.profile[ID] = count
}

func (ps *profilesSyncing) stop(ID string) {
	ps.Lock()
	defer ps.Unlock()
	count := ps.profile[ID]
	if count > 0 {
		count--
		ps.profile[ID] = count
	}
}

func (ps *profilesSyncing) count(ID string) int {
	ps.RLock()
	defer ps.RUnlock()
	return ps.profile[ID]
}

func init() {
	syncing = profilesSyncing{
		profile: make(map[string]int),
	}
}

func newProfile(name string, direction, conflictResolution, conflictDurationSeconds int, active bool, ignore []string,
	localPath, remotePath string, remoteClient *client) (*profileStore, error) {
	ps := &profileStore{
		Profile: &syncer.Profile{
			Name:               name,
			Direction:          direction,
			ConflictResolution: conflictResolution,
		},
		Ignore:                  ignore,
		Active:                  active,
		LocalPath:               localPath,
		RemotePath:              remotePath,
		Client:                  remoteClient,
		ConflictDurationSeconds: conflictDurationSeconds,
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
	if strings.TrimSpace(p.Name) == "" {
		return errors.New("No Name specified for this Sync Profile")
	}
	if strings.TrimSpace(p.LocalPath) == "" {
		return errors.New("Local path not set")
	}
	if strings.TrimSpace(p.RemotePath) == "" {
		return errors.New("Remote path not set")
	}

	if p.Direction != syncer.DirectionBoth &&
		p.Direction != syncer.DirectionLocalOnly &&
		p.Direction != syncer.DirectionRemoteOnly {

		return errors.New("Invalid sync profile direction")
	}

	if p.ConflictResolution != syncer.ConResOverwrite &&
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
	if !lFile.Exists() {
		return fmt.Errorf("Local sync path does not exist!")
	}
	p.Profile.Local = lFile

	c, err := remoteClient(p.Client)
	if err != nil {
		return err
	}
	rFile, err := remote.New(c, p.RemotePath)
	if err != nil {
		return fmt.Errorf("Error accessing the remote sync path: %s", err)
	}

	if !rFile.Exists() {
		return fmt.Errorf("Remote sync path does not exist!")
	}

	p.Profile.Remote = rFile

	p.ID = p.Profile.ID()
	p.Profile.ConflictDuration = time.Duration(p.ConflictDurationSeconds) * time.Second

	return nil
}

func (p *profileStore) update() error {
	oldID := p.ID
	err := p.prep()
	if err != nil {
		return err
	}

	if oldID != "" && oldID != p.Profile.ID() {
		//ID changed, check if an existing profile
		// is already syncing these paths
		_, err = getProfile(p.Profile.ID())
		if err != datastore.ErrNotFound {
			return errors.New("A profile syncing these two locations already exist!")
		}
		// delete old profile
		err = deleteProfile(oldID)
		if err != nil {
			return err
		}
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

func (p *profileStore) status() (int, string) {
	count := syncing.count(p.ID)
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
	p.prep()
	if p.Profile.Local != nil && p.Profile.Remote != nil {
		p.Profile.Stop()
	}
	return deleteProfile(p.ID)
}
