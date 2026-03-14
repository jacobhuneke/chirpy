package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
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
	err := a.db.DeleteRefreshTokens(r.Context())
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}
	err = a.db.DeleteChirps(r.Context())
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}
	err = a.db.DeleteUsers(r.Context())
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
	respondWithJSON(w, 201, UserC{user.ID, user.CreatedAt, user.UpdatedAt, user.Email, user.IsChirpyRed})
}

func (a *apiConfig) handlerChirps(w http.ResponseWriter, r *http.Request) {
	//sets input parameters, the chirp and the user
	type parameters struct {
		Body   string    `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}

	//reads request body into parameters
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}

	//validate the user
	token, err := auth.GetBearerToken(r.Header)
	if err != nil || token == "" {
		respondWithError(w, 401, "Unauthorized")
		return
	}
	id, err := auth.ValidateJWT(token, a.secret)
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		return
	}

	//list of banned words to check chirp for, formats body string for check
	bannedWords := []string{"kerfuffle", "sharbert", "fornax"}
	lowercaseBody := strings.ToLower(params.Body)
	splLowerBody := strings.Split(lowercaseBody, " ")
	splRegBody := strings.Split(params.Body, " ")

	//checks and censors
	for i, word := range splLowerBody {
		for _, banned := range bannedWords {
			if word == banned {
				splRegBody[i] = "****"
			}
		}
	}

	//checks if chirp is too long
	if len(params.Body) > 140 {
		errString := "Chirp is too long"
		respondWithError(w, 400, errString)
		return
	} else { //creates a new chirp in the database and returns it
		chirpParams := database.CreateChirpParams{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Body:      strings.Join(splRegBody, " "),
			UserID:    id,
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
	auth := r.URL.Query().Get("author_id")
	s := r.URL.Query().Get("sort")
	var chirps []database.Chirp
	var err error
	if auth != "" {
		user_id, err := uuid.Parse(auth)
		if err != nil {
			respondWithError(w, 400, err.Error())
			return
		}
		chirps, err = a.db.GetChirpsByAuthor(r.Context(), user_id)
		if err != nil {
			respondWithError(w, 400, err.Error())
			return
		}
	} else {
		chirps, err = a.db.GetChirps(r.Context())
		if err != nil {
			respondWithError(w, 400, err.Error())
			return
		}
	}
	if s != "" && s == "desc" {
		sort.Slice(chirps, func(i, j int) bool { return chirps[i].CreatedAt.After(chirps[j].CreatedAt) })
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
		Password           string `json:"password"`
		Email              string `json:"email"`
		Expires_in_seconds *int   `json:"expires_in_seconds,omitempty"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}

	user, err := a.db.GetUserByEmail(r.Context(), params.Email)
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		return
	}
	b, err := auth.CheckPasswordHash(params.Password, user.HashedPassword)

	if err != nil || b == false {
		respondWithError(w, 401, "Unauthorized")
		return
	}

	token, err := auth.MakeJWT(user.ID, a.secret, time.Hour)
	if err != nil {
		respondWithError(w, 401, err.Error())
		return
	}
	refresh_token := auth.MakeRefreshToken()
	rt_params := database.CreateRefreshTokenParams{
		Token:     refresh_token,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(1440 * time.Hour),
		RevokedAt: sql.NullTime{},
	}
	_, err = a.db.CreateRefreshToken(r.Context(), rt_params)
	if err != nil {
		respondWithError(w, 401, err.Error())
		return
	}

	respondWithJSON(w, 200, UserL{user.ID, user.CreatedAt, user.UpdatedAt, user.Email, token, refresh_token, user.IsChirpyRed})
}

func (a *apiConfig) handlerRefresh(w http.ResponseWriter, r *http.Request) {
	rt, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, err.Error())
		return
	}
	refresh_token, err := a.db.GetUserFromRefreshToken(r.Context(), rt)
	if err != nil {
		respondWithError(w, 401, "cannot get user")
		return
	}
	if refresh_token.RevokedAt.Valid || refresh_token.ExpiresAt.Before(time.Now()) {
		respondWithError(w, 401, "invalid token")
		return
	}
	token, err := auth.MakeJWT(refresh_token.UserID, a.secret, time.Duration(3600*time.Second))
	if err != nil {
		respondWithError(w, 401, "cannot make token")
		return
	}
	respondWithJSON(w, 200, struct {
		Token string `json:"token"`
	}{
		Token: token,
	})
}

func (a *apiConfig) handlerRevoke(w http.ResponseWriter, r *http.Request) {
	rt, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, err.Error())
		return
	}
	_, err = a.db.RevokeRefreshToken(r.Context(), rt)
	if err != nil {
		respondWithError(w, 401, "Cannot revoke")
		return
	}
	w.WriteHeader(204)
}
func (a *apiConfig) handlerUpdateLoginCreds(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, err.Error())
		return
	}
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err = decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 401, err.Error())
		return
	}
	userID, err := auth.ValidateJWT(token, a.secret)
	if err != nil {
		respondWithError(w, 401, err.Error())
		return
	}
	hashedPass, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, 401, err.Error())
		return
	}
	newCreds := database.UpdateCredsParams{
		ID:             userID,
		Email:          params.Email,
		HashedPassword: hashedPass,
	}
	u, err := a.db.UpdateCreds(r.Context(), newCreds)
	if err != nil {
		respondWithError(w, 401, err.Error())
		return
	}
	respondWithJSON(w, 200, UserC{u.ID, u.CreatedAt, u.UpdatedAt, u.Email, u.IsChirpyRed})
}

func (a *apiConfig) handlerDeleteChirp(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, err.Error())
		return
	}
	userID, err := auth.ValidateJWT(token, a.secret)
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		return
	}
	path := r.PathValue("chirpID")
	if path != "" {
		chirpid, err := uuid.Parse(path)
		if err != nil {
			respondWithError(w, 404, "couldnt get path uuid")
			return
		}
		chirp, err := a.db.GetChirp(r.Context(), chirpid)
		if err != nil {
			respondWithError(w, 404, "could not get chirp")
			return
		}
		if chirp.UserID != userID {
			respondWithError(w, 403, "Forbidden")
			return
		}
		err = a.db.DeleteChirpByID(r.Context(), chirpid)
		if err != nil {
			respondWithError(w, 404, "could not delete chirp")
			return
		}
		w.WriteHeader(204)
	}
	respondWithError(w, 403, "must pass a chirp id")
}

func (a *apiConfig) handlerUpgradeRed(w http.ResponseWriter, r *http.Request) {
	polkaKey, err := auth.GetAPIKey(r.Header)
	if err != nil {
		w.WriteHeader(401)
		return
	}
	if polkaKey != a.polka_key {
		w.WriteHeader(401)
		return
	}
	type parameters struct {
		Event string `json:"event"`
		Data  struct {
			UserID uuid.UUID `json:"user_id"`
		} `json:"data"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err = decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 401, err.Error())
		return
	}
	if params.Event != "user.upgraded" {
		w.WriteHeader(204)
		return
	}
	_, err = a.db.UpgradeRedByUserID(r.Context(), params.Data.UserID)
	if err != nil {
		respondWithError(w, 404, err.Error())
		return
	}
	w.WriteHeader(204)
}
