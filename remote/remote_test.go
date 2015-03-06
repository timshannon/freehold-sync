package remote

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

var (
	mux      *http.ServeMux
	server   *httptest.Server
	username = "tester"
	token    = "testerToken"
	filePath = "/v1/file/testing/bootstrap.zip"
)

func startMockServer() {
	mux = http.NewServeMux()
	server = httptest.NewServer(mux)
}

func stopMockServer() {
	server.Close()
}

func TestFile(t *testing.T) {
	startMockServer()
	defer stopMockServer()

	//Setup Mock Handler
	mux.HandleFunc("/v1/property/file/testing",
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `{
    status: "success",
    data: 
        {   name: "family-pictures", 
            url: "/v1/file/testing/", 
            isDir: true,
            permissions: {owner: "tshannon", public: "",    friend: "r",    private: "rw"},
        }`)
		})

	client := NewClient(server.URL, username, token, false)

	f, err := client.File(filePath)
	if err != nil {
		t.Fatal(err)
	}

	if !f.IsDir() {
		t.Fatal("Parent Directory not registering as Dir")
	}
	/*
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
	*/
}

func TestRealFile(t *testing.T) {

	client := NewClient("https://tshannon.org", "tshannon", "eAhmtD1hJfzV4C3GvDSIOAkumN54vSgCOIOcrLb5w2A=", true)

	f, err := client.File(filePath)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(f.Modified())

	if !f.IsDir() {
		t.Fatal("Parent Directory not registering as Dir")
	}
	/*
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
	*/
}
