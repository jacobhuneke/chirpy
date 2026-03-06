package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func main() {
	apicfg := apiConfig{
		fileserverHits: atomic.Int32{},
	}

	serveMux := http.NewServeMux()
	handler := http.StripPrefix("/app", http.FileServer(http.Dir(".")))
	serveMux.Handle("/app/", apicfg.middlewareMetricsInc(handler))
	serveMux.HandleFunc("GET /healthz", handlerReadiness)
	serveMux.HandleFunc("GET /metrics", apicfg.handlerNumReqs)
	serveMux.HandleFunc("POST /reset", apicfg.handlerReqReset)

	server := http.Server{
		Addr:    ":8080",
		Handler: serveMux,
	}

	log.Fatal(server.ListenAndServe())
}

func handlerReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}

func (a *apiConfig) handlerNumReqs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	str := fmt.Sprintf("Hits: %v", a.fileserverHits.Load())
	w.Write([]byte(str))
}

func (a *apiConfig) handlerReqReset(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	a.fileserverHits.Store(0)
}
