package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jacobhuneke/chirpy/internal/auth"
	"github.com/jacobhuneke/chirpy/internal/database"
)

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
	if a.platform != "dev" {
		respondWithError(w, 403, "Forbidden")
		return
	}

	err := a.db.DeleteUsers(r.Context())
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	a.fileserverHits.Store(0)

}

func (a *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}

	hashed, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, 500, err.Error())
		return
	}

	userParams := database.CreateUserParams{
		ID:             uuid.New(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		Email:          params.Email,
		HashedPassword: hashed,
	}

	user, err := a.db.CreateUser(r.Context(), userParams)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}
	respondWithJSON(w, 201, User{user.ID, user.CreatedAt, user.UpdatedAt, user.Email})
}

func (a *apiConfig) handlerChirps(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body   string    `json:"body"`
		UserID uuid.UUID `json:"user_id"`
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
		return
	} else {
		chirpParams := database.CreateChirpParams{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Body:      strings.Join(splRegBody, " "),
			UserID:    params.UserID,
		}
		params.Body = strings.Join(splRegBody, " ")
		chirp, err := a.db.CreateChirp(r.Context(), chirpParams)
		if err != nil {
			respondWithError(w, 400, err.Error())
			return
		}
		respondWithJSON(w, 201, Chirp{chirp.ID, chirp.CreatedAt, chirp.UpdatedAt, chirp.Body, chirp.UserID})
	}
}

func (a *apiConfig) handlerGetChirps(w http.ResponseWriter, r *http.Request) {
	chirps, err := a.db.GetChirps(r.Context())
	if err != nil {
		respondWithError(w, 400, err.Error())
		return
	}
	jsonchirps := []Chirp{}
	for _, chirp := range chirps {
		jsonchirps = append(jsonchirps, Chirp{chirp.ID, chirp.CreatedAt, chirp.UpdatedAt, chirp.Body, chirp.UserID})
	}
	respondWithJSON(w, 200, jsonchirps)
}

func (a *apiConfig) handlerGetChirp(w http.ResponseWriter, r *http.Request) {
	path := r.PathValue("chirpID")
	if path != "" {
		id, err := uuid.Parse(path)
		if err != nil {
			respondWithError(w, 404, "couldnt get path uuid")
			return
		}
		chirp, err := a.db.GetChirp(r.Context(), id)
		if err != nil {
			respondWithError(w, 404, "could not get chirp")
			return
		}
		respondWithJSON(w, 200, Chirp{chirp.ID, chirp.CreatedAt, chirp.UpdatedAt, chirp.Body, chirp.UserID})
	} else {
		respondWithError(w, 404, "could not get path")
		return
	}

}

func (a *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}
	user, err := a.db.GetUser(r.Context(), params.Email)
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		return
	}
	b, err := auth.CheckPasswordHash(params.Password, user.HashedPassword)

	if err != nil || b == false {
		respondWithError(w, 401, "Unauthorized")
		return
	}
	respondWithJSON(w, 200, User{user.ID, user.CreatedAt, user.UpdatedAt, user.Email})
}
