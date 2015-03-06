package main

import "net/http"

func errHandled(err error, w http.ResponseWriter) bool {
	if err == nil {
		return false
	}

	respondJsend(w, &jsend{
		Status:  statusError,
		Message: err.Error(),
	})

	return true
}
