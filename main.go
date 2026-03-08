package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"

	"github.com/jacobhuneke/chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	queries        database.Queries
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}

	dbQueries := database.New(db)

	apicfg := apiConfig{
		fileserverHits: atomic.Int32{},
		queries:        *dbQueries,
	}

	serveMux := http.NewServeMux()
	handler := http.StripPrefix("/app", http.FileServer(http.Dir(".")))
	serveMux.Handle("/app/", apicfg.middlewareMetricsInc(handler))
	serveMux.HandleFunc("GET /api/healthz", handlerReadiness)
	serveMux.HandleFunc("GET /admin/metrics", apicfg.handlerNumReqs)
	serveMux.HandleFunc("POST /admin/reset", apicfg.handlerReqReset)
	serveMux.HandleFunc("POST /api/validate_chirp", handlerValidate)

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
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	str := fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, a.fileserverHits.Load())
	w.Write([]byte(str))
}

func (a *apiConfig) handlerReqReset(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	a.fileserverHits.Store(0)
}

func handlerValidate(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}
	type returnVals struct {
		CleanedBody string `json:"cleaned_body"`
	}
	bannedWords := []string{"kerfuffle", "sharbert", "fornax"}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}

	lowercaseBody := strings.ToLower(params.Body)
	splLowerBody := strings.Split(lowercaseBody, " ")
	splRegBody := strings.Split(params.Body, " ")

	for i, word := range splLowerBody {
		for _, banned := range bannedWords {
			if word == banned {
				splRegBody[i] = "****"
			}
		}
	}

	if len(params.Body) > 140 {
		errString := "Chirp is too long"
		respondWithError(w, 400, errString)
	} else {
		respondWithJSON(w, 200, returnVals{
			CleanedBody: strings.Join(splRegBody, " "),
		})
	}
}

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
