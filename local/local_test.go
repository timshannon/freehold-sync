package local

import (
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"
)

func TestFile(t *testing.T) {
	parent := "../local"
	f, err := New(parent)
	if err != nil {
		t.Fatal(err)
	}

	if !f.IsDir() {
		t.Fatal("Parent Directory not registering as Dir")
	}

	c, err := f.Children()
	if err != nil {
		t.Fatalf("Error getting children of file: %s", err)
	}

	found := false
	for i := range c {
		if c[i].ID() == filepath.Join(parent, "local_test.go") {
			found = true
			if c[i].IsDir() {
				t.Fatal("local_test.go is registering as a dir when it is a file")
			}

			if c[i].Modified().After(time.Now()) {
				t.Fatal("Modified date is after current time")
			}

			r, err := c[i].Data()
			defer r.Close()
			if err != nil {
				t.Fatalf("Error getting data from file: %s", err)
			}

			data, err := ioutil.ReadAll(r)
			if err != nil {
				t.Fatalf("Error reading from file: %s", err)
			}
			if len(data) == 0 {
				t.Fatalf("No data was read from the file")
			}

		}
	}

	if !found {
		t.Fatal("local_test.go not found as child of the current directory")
	}

}
