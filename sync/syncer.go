// Copyright 2015 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

import "time"

// Syncer is used for comparing two files local or remote
// to determine which one should be overwritten based on
// the sync profile rules
type Syncer interface {
	Path() string
	Modified() *time.Time
	Children() []Syncer
	Data() []byte
}
