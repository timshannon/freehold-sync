package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"

	"bitbucket.org/tshannon/freehold/fail"
)

const (
	statusSuccess = "success"
	statusError   = "error"
	statusFail    = "fail"
)

type jsend struct {
	Status   string      `json:"status"`
	Data     interface{} `json:"data,omitempty"`
	Message  string      `json:"message,omitempty"`
	Failures []error     `json:"failures,omitempty"`
}

//respondJsend marshalls the input into a json byte array
// and writes it to the reponse with appropriate header
func respondJsend(w http.ResponseWriter, response *jsend) {
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Type", "application/json")

	if len(response.Failures) > 0 && response.Message == "" {
		response.Message = "One or more item has failed. Check the individual failures for details."
	}

	result, err := json.Marshal(response)
	if err != nil {
		result, _ = json.Marshal(&jsend{
			Status:  statusError,
			Message: "An internal error occurred, and we'll look into it.",
		})
	}

	switch response.Status {
	case statusFail:
		w.WriteHeader(http.StatusBadRequest)
	case statusError:
		w.WriteHeader(http.StatusInternalServerError)
	}

	w.Write(result)
}

func parseJSON(r *http.Request, result interface{}) error {
	buff, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	if len(buff) == 0 {
		if r.Method != "GET" {
			return nil
		}
		//Browsers will send GET request body as URL parms
		// We'll support either, but the request body will
		// take precedence
		v, err := url.QueryUnescape(r.URL.RawQuery)
		if err != nil {
			return fail.NewFromErr(err, r.URL.RawQuery)
		}

		buff = []byte(v)
	}

	if len(buff) == 0 {
		return nil
	}

	err = json.Unmarshal(buff, result)
	if err != nil {
		return err
	}
	return nil
}
