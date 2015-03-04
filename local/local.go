// Copyright 2015 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package local

import (
	"os"

	"bitbucket.org/tshannon/freehold-sync/sync"
)

type File struct {
	filepath string
	info     os.FileInfo
}

func New(filepath string) (sync.Syncer, error) {

}
