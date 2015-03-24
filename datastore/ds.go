// Copyright 2015 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package datastore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"

	"bitbucket.org/tshannon/freehold/data/store"
)

// ErrNotFound is returned when a value isn't found for the passed in key
var ErrNotFound = errors.New("Value not found")

// DS is a wrapper of the store interface with a few
// handy things added for managing datastores
type DS struct {
	store.Storer
}

// Open opens a datastore
func Open(filename string) (*DS, error) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		err := os.MkdirAll(path.Dir(filename), 0777)
		if err != nil {
			return nil, errors.New("Error creating datastore folder at: " + path.Dir(filename) + ":" + err.Error())
		}
		err = store.Create(filename)
		if err != nil {
			return nil, errors.New("Error creating datastore " + filename + ": " + err.Error())
		}
	}
	ds, err := store.Open(filename)

	if err != nil {
		return nil, errors.New("Error opening datastore " + filename + ": " + err.Error())
	}
	return &DS{ds}, nil
}

// Get gets a value from the DS for the passed in key
func (c *DS) Get(key interface{}, result interface{}) error {
	dsKey, err := json.Marshal(key)
	if err != nil {
		return err
	}

	fmt.Println("dskey: ", string(dsKey))
	dsValue, err := c.Storer.Get(dsKey)
	if err != nil {
		return err
	}

	if dsValue == nil {
		return ErrNotFound
	}

	return json.Unmarshal(dsValue, result)
}

// Put puts a new value in the DS at the given key
func (c *DS) Put(key interface{}, value interface{}) error {
	dsKey, err := json.Marshal(key)
	if err != nil {
		return err
	}

	dsValue, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return c.Storer.Put(dsKey, dsValue)
}

// Delete removes the value from the DS for the given key
func (c *DS) Delete(key interface{}) error {
	dsKey, err := json.Marshal(key)
	if err != nil {
		return err
	}

	return c.Storer.Delete(dsKey)
}
