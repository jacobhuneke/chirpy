package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func respondWithError(w http.ResponseWriter, code int, msg string) {
	// We send the error message in a JSON object: {"error": "message"}
	type errorRes struct {
		Error string `json:"error"`
	}
	respondWithJSON(w, code, errorRes{
		Error: msg,
	})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	// 1. Set the header so the client knows JSON is coming
	w.Header().Set("Content-Type", "application/json")

	// 2. Turn the Go struct into JSON bytes
	dat, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}

	// 3. Send the status code and the data
	w.WriteHeader(code)
	w.Write(dat)
}
