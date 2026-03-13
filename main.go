package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jacobhuneke/chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbURL          string
	platform       string
	db             database.Queries
	secret         string
}
type UserC struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

type UserL struct {
	ID           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Email        string    `json:"email"`
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
}

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
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
	platform := os.Getenv("PLATFORM")
	secret := os.Getenv("SECRET")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}

	dbQueries := database.New(db)

	apicfg := apiConfig{
		fileserverHits: atomic.Int32{},
		dbURL:          dbURL,
		platform:       platform,
		db:             *dbQueries,
		secret:         secret,
	}

	serveMux := http.NewServeMux()
	handler := http.StripPrefix("/app", http.FileServer(http.Dir(".")))
	serveMux.Handle("/app/", apicfg.middlewareMetricsInc(handler))
	serveMux.HandleFunc("GET /api/healthz", handlerReadiness)
	serveMux.HandleFunc("GET /admin/metrics", apicfg.handlerNumReqs)
	serveMux.HandleFunc("POST /admin/reset", apicfg.handlerReqReset)
	serveMux.HandleFunc("POST /api/users", apicfg.handlerCreateUser)
	serveMux.HandleFunc("POST /api/chirps", apicfg.handlerChirps)
	serveMux.HandleFunc("GET /api/chirps", apicfg.handlerGetChirps)
	serveMux.HandleFunc("GET /api/chirps/{chirpID}", apicfg.handlerGetChirp)
	serveMux.HandleFunc("POST /api/login", apicfg.handlerLogin)
	serveMux.HandleFunc("POST /api/refresh", apicfg.handlerRefresh)
	serveMux.HandleFunc("POST /api/revoke", apicfg.handlerRevoke)
	serveMux.HandleFunc("PUT /api/users", apicfg.handlerUpdateLoginCreds)

	server := http.Server{
		Addr:    ":8080",
		Handler: serveMux,
	}

	log.Fatal(server.ListenAndServe())
}
