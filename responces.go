package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/PrincessFluffyButt937/Chirpy/internal/auth"
	"github.com/PrincessFluffyButt937/Chirpy/internal/database"
	"github.com/google/uuid"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	platform       string
	secret         string
	polka_key      string
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(res, req)
	})

}
func (cfg *apiConfig) MetricsTotal() http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.Header().Set("Content-Type", "text/html; charset=utf-8")
		res.WriteHeader(200)
		res.Write([]byte(fmt.Sprintf(`
<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, cfg.fileserverHits.Load())))
	})

}

type Status struct {
	Status string `json:"status"`
}

func (cfg *apiConfig) MetricsReset() http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if cfg.platform != "dev" {
			ErrorResponce(res, 403, "Insufficient authorization level.")
			return
		}
		cfg.fileserverHits.Store(0)
		if err := cfg.db.DeleteUsers(req.Context()); err != nil {
			msg := fmt.Sprintf("DB_DeleteUsers: %s", err)
			ErrorResponce(res, 500, msg)
			return
		}
		body := Status{
			Status: "Ok",
		}
		JsonResponce(res, 200, body)
	})
}

// User auth

type User struct {
	ID            uuid.UUID `json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Email         string    `json:"email"`
	Token         string    `json:"token"`
	Refresh_Token string    `json:"refresh_token"`
	IsChirpyRed   bool      `json:"is_chirpy_red"`
}

type Create_user struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (cfg *apiConfig) Create_user_handler(res http.ResponseWriter, req *http.Request) {
	user := Create_user{}
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&user); err != nil {
		msg := fmt.Sprintf("Decoder_Err_Create_user_handler: %s", err)
		ErrorResponce(res, 400, msg)
		return
	}
	hashed_password, err := auth.HashPassword(user.Password)
	if err != nil {
		msg := fmt.Sprintf("HashPassword: %s", err)
		ErrorResponce(res, 400, msg)
		return
	}

	now := time.Now()
	new_user := database.CreateUserParams{
		ID:             uuid.New(),
		CreatedAt:      now,
		UpdatedAt:      now,
		Email:          user.Email,
		HashedPassword: hashed_password,
	}
	db_user, err := cfg.db.CreateUser(req.Context(), new_user)
	if err != nil {
		msg := fmt.Sprintf("DB_CreateUser: %s", err)
		ErrorResponce(res, 500, msg)
		return
	}
	Created_db_User := User{
		ID:          db_user.ID,
		CreatedAt:   db_user.CreatedAt,
		UpdatedAt:   db_user.UpdatedAt,
		Email:       db_user.Email,
		IsChirpyRed: db_user.IsChirpyRed,
	}
	JsonResponce(res, 201, Created_db_User)
}

func (cfg *apiConfig) Login_user_handler(res http.ResponseWriter, req *http.Request) {
	user := Create_user{}
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&user); err != nil {
		msg := fmt.Sprintf("Login_user_handler: %s", err)
		ErrorResponce(res, 400, msg)
		return
	}
	db_user, err := cfg.db.GetUserByEmail(req.Context(), user.Email)
	if err != nil {
		msg := fmt.Sprintf("DB_GetUserByEmail: %s", err)
		ErrorResponce(res, 400, msg)
		return
	}
	match, err := auth.CheckPasswordHash(user.Password, db_user.HashedPassword)
	if err != nil {
		msg := fmt.Sprintf("CheckPasswordHash: %s", err)
		ErrorResponce(res, 401, msg)
		return
	}
	if !match {
		ErrorResponce(res, 401, "Password is incorrect")
		return
	}
	token, err := auth.MakeJWT(db_user.ID, cfg.secret, time.Duration(1)*time.Hour)
	if err != nil {
		ErrorResponce(res, 400, err.Error())
		return
	}
	ref_token := auth.MakeRefreshToken()
	now := time.Now()

	refresh_token_data := database.CreateRefreshTokenParams{
		Token:     ref_token,
		CreatedAt: now,
		UpdatedAt: now,
		UserID:    db_user.ID,
		ExpiresAt: now.Add(1440 * time.Hour),
		RevokedAt: sql.NullTime{Valid: false},
	}
	if err := cfg.db.CreateRefreshToken(req.Context(), refresh_token_data); err != nil {
		ErrorResponce(res, 400, err.Error())
		return
	}

	user_json := User{
		ID:            db_user.ID,
		CreatedAt:     db_user.CreatedAt,
		UpdatedAt:     db_user.UpdatedAt,
		Email:         db_user.Email,
		Token:         token,
		Refresh_Token: ref_token,
		IsChirpyRed:   db_user.IsChirpyRed,
	}
	JsonResponce(res, 200, user_json)
}

type Access_Token struct {
	Token string `json:"token"`
}

func (cfg *apiConfig) Refresh_access_token_handler(res http.ResponseWriter, req *http.Request) {
	ref_token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		ErrorResponce(res, 401, err.Error())
		return
	}
	db_ref_token, err := cfg.db.GetRefreshToken(req.Context(), ref_token)
	if err != nil {
		ErrorResponce(res, 401, err.Error())
		return
	}
	if time.Now().After(db_ref_token.ExpiresAt) {
		ErrorResponce(res, 401, "Token expired.")
		return
	}
	if db_ref_token.RevokedAt.Valid {
		ErrorResponce(res, 401, "Token revoked.")
		return
	}
	token, err := auth.MakeJWT(db_ref_token.UserID, cfg.secret, time.Duration(1)*time.Hour)
	if err != nil {
		ErrorResponce(res, 400, err.Error())
		return
	}
	Access_token_json := Access_Token{
		Token: token,
	}
	JsonResponce(res, 200, Access_token_json)
}

func (cfg *apiConfig) Revoke_refresh_token_handler(res http.ResponseWriter, req *http.Request) {
	ref_token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		ErrorResponce(res, 401, err.Error())
		return
	}
	db_ref_token, err := cfg.db.GetRefreshToken(req.Context(), ref_token)
	if err != nil {
		ErrorResponce(res, 401, err.Error())
		return
	}
	now := time.Now()
	rewoke_params := database.RevokeRefreshTokenParams{
		UpdatedAt: now,
		RevokedAt: sql.NullTime{Time: now, Valid: true},
		Token:     db_ref_token.Token,
	}
	if err := cfg.db.RevokeRefreshToken(req.Context(), rewoke_params); err != nil {
		ErrorResponce(res, 500, err.Error())
		return
	}
	res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	res.WriteHeader(204)
}

func (cfg *apiConfig) Update_user_email_password_handler(res http.ResponseWriter, req *http.Request) {
	jwt_token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		ErrorResponce(res, 401, err.Error())
		return
	}
	user := Create_user{}
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&user); err != nil {
		ErrorResponce(res, 400, err.Error())
		return
	}
	token_UserID, err := auth.ValidateJWT(jwt_token, cfg.secret)
	if err != nil {
		ErrorResponce(res, 401, err.Error())
		return
	}
	new_hashed_password, err := auth.HashPassword(user.Password)
	if err != nil {
		ErrorResponce(res, 400, err.Error())
		return
	}
	new_user_data := database.UpdateUserEmailPasswordParams{
		Email:          user.Email,
		HashedPassword: new_hashed_password,
		UpdatedAt:      time.Now(),
		ID:             token_UserID,
	}
	if err := cfg.db.UpdateUserEmailPassword(req.Context(), new_user_data); err != nil {
		ErrorResponce(res, 400, err.Error())
		return
	}
	db_user, err := cfg.db.GetUserByEmail(req.Context(), user.Email)
	if err != nil {
		ErrorResponce(res, 500, err.Error())
		return
	}
	json_struct := User{
		ID:          db_user.ID,
		CreatedAt:   db_user.CreatedAt,
		UpdatedAt:   db_user.UpdatedAt,
		Email:       db_user.Email,
		Token:       jwt_token,
		IsChirpyRed: db_user.IsChirpyRed,
	}
	JsonResponce(res, 200, json_struct)
}

// Chirps

type New_chirp struct {
	Body    string    `json:"body"`
	User_id uuid.UUID `json:"user_id"`
}

type Created_chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

func (cfg *apiConfig) New_Chirp_handler(res http.ResponseWriter, req *http.Request) {
	chirp_body := New_chirp{}
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&chirp_body); err != nil {
		msg := fmt.Sprintf("Decoder_Err_ValidateChirp: %s", err)
		ErrorResponce(res, 400, msg)
		return
	}
	if len(chirp_body.Body) > 140 {
		ErrorResponce(res, 400, "Chirp is too long")
		return
	}
	token_string, err := auth.GetBearerToken(req.Header)
	if err != nil {
		ErrorResponce(res, 401, err.Error())
		return
	}
	UserID_from_token, err := auth.ValidateJWT(token_string, cfg.secret)
	if err != nil {
		ErrorResponce(res, 401, err.Error())
		return
	}

	now := time.Now()
	chirp_param := database.CreateChirpParams{
		ID:        uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
		Body:      chirp_body.Body,
		UserID:    UserID_from_token,
	}
	db_chirp, err := cfg.db.CreateChirp(req.Context(), chirp_param)
	if err != nil {
		msg := fmt.Sprintf("DB_CreateChirp: %s", err)
		ErrorResponce(res, 400, msg)
		return
	}
	db_out := Created_chirp{
		ID:        db_chirp.ID,
		CreatedAt: db_chirp.CreatedAt,
		UpdatedAt: db_chirp.UpdatedAt,
		Body:      db_chirp.Body,
		UserID:    db_chirp.UserID,
	}
	JsonResponce(res, 201, db_out)
}

func (cfg *apiConfig) Get_Chirps_handler(res http.ResponseWriter, req *http.Request) {
	req_user_id := req.URL.Query().Get("author_id")
	req_sort := req.URL.Query().Get("sort")
	if req_user_id == "" {
		chirps, err := cfg.db.GetChirps(req.Context())
		if err != nil {
			ErrorResponce(res, 500, err.Error())
			return
		}
		Chirp_L := make([]Created_chirp, len(chirps))

		for i, chirp := range chirps {
			temp := Created_chirp{
				ID:        chirp.ID,
				CreatedAt: chirp.CreatedAt,
				UpdatedAt: chirp.UpdatedAt,
				Body:      chirp.Body,
				UserID:    chirp.UserID,
			}
			Chirp_L[i] = temp
		}
		if req_sort == "desc" {
			sort.Slice(Chirp_L, func(i, j int) bool { return Chirp_L[i].CreatedAt.After(Chirp_L[j].CreatedAt) })
		}

		JsonResponce(res, 200, Chirp_L)
	} else {
		parsed_user_id, err := uuid.Parse(req_user_id)
		if err != nil {
			ErrorResponce(res, 400, err.Error())
			return
		}
		chirps, err := cfg.db.GetChirpsByUserID(req.Context(), parsed_user_id)
		if err != nil {
			ErrorResponce(res, 400, err.Error())
			return
		}
		Chirp_L := make([]Created_chirp, len(chirps))

		for i, chirp := range chirps {
			temp := Created_chirp{
				ID:        chirp.ID,
				CreatedAt: chirp.CreatedAt,
				UpdatedAt: chirp.UpdatedAt,
				Body:      chirp.Body,
				UserID:    chirp.UserID,
			}
			Chirp_L[i] = temp
		}
		if req_sort == "desc" {
			sort.Slice(Chirp_L, func(i, j int) bool { return Chirp_L[i].CreatedAt.After(Chirp_L[j].CreatedAt) })
		}
		JsonResponce(res, 200, Chirp_L)
	}
}

func (cfg *apiConfig) Get_Chirp_By_ID_handler(res http.ResponseWriter, req *http.Request) {
	ID := req.PathValue("chirpID")
	UUID, err := uuid.Parse(ID)
	if err != nil {
		msg := fmt.Sprintf("uuid.Parse: %s", err)
		ErrorResponce(res, 404, msg)
		return
	}
	chirp, err := cfg.db.GetChirpByID(req.Context(), UUID)
	if err != nil {
		msg := fmt.Sprintf("DB_GetChirpByID: %s", err)
		ErrorResponce(res, 404, msg)
		return
	}
	return_chirp := Created_chirp{
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserID:    chirp.UserID,
	}
	JsonResponce(res, 200, return_chirp)
}

func (cfg *apiConfig) Delete_Chirp_By_ID_handler(res http.ResponseWriter, req *http.Request) {
	ID := req.PathValue("chirpID")
	chirp_uuid, err := uuid.Parse(ID)
	if err != nil {
		msg := fmt.Sprintf("uuid.Parse: %s", err)
		ErrorResponce(res, 404, msg)
		return
	}
	access_token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		ErrorResponce(res, 401, err.Error())
		return
	}
	user_id, err := auth.ValidateJWT(access_token, cfg.secret)
	if err != nil {
		ErrorResponce(res, 401, err.Error())
		return
	}
	db_chirp, err := cfg.db.GetChirpByID(req.Context(), chirp_uuid)
	if err != nil {
		ErrorResponce(res, 404, err.Error())
		return
	}
	if db_chirp.UserID != user_id {
		ErrorResponce(res, 403, "You are not authorized to delete this chirp.")
		return
	}
	if err := cfg.db.DeleteChirpByID(req.Context(), db_chirp.ID); err != nil {
		ErrorResponce(res, 500, err.Error())
		return
	}
	res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	res.WriteHeader(204)
}

// webhooks

type Webhook struct {
	Event string `json:"event"`
	Data  struct {
		User_ID string `json:"user_id"`
	} `json:"data"`
}

func (cfg *apiConfig) Update_User_Chirp_Red(res http.ResponseWriter, req *http.Request) {
	api_key, err := auth.GetApiKey(req.Header)
	if err != nil {
		ErrorResponce(res, 401, err.Error())
		return
	}
	if api_key != cfg.polka_key {
		ErrorResponce(res, 403, "Invalid ApiKey.")
		return
	}
	hook := Webhook{}
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&hook); err != nil {
		ErrorResponce(res, 400, err.Error())
		return
	}
	if hook.Event != "user.upgraded" {
		res.Header().Set("Content-Type", "text/plain; charset=utf-8")
		res.WriteHeader(204)
		return
	}
	user_uuid, err := uuid.Parse(hook.Data.User_ID)
	if err != nil {
		ErrorResponce(res, 400, err.Error())
		return
	}
	new_user_data := database.UpdateUserIsChirpyRedParams{
		IsChirpyRed: true,
		UpdatedAt:   time.Now(),
		ID:          user_uuid,
	}
	if err := cfg.db.UpdateUserIsChirpyRed(req.Context(), new_user_data); err != nil {
		ErrorResponce(res, 404, err.Error())
		return
	}
	res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	res.WriteHeader(204)
}

// json responces

type Error_json struct {
	Err string `json:"error"`
}

func ErrorResponce(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	data := Error_json{
		Err: msg,
	}
	m_data, err := json.Marshal(&data)
	if err != nil {
		log.Printf("ErrorResponce Marshal err: %s\n", err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(code)
	w.Write(m_data)
}

func JsonResponce(w http.ResponseWriter, code int, content interface{}) {
	w.Header().Set("Content-Type", "application/json")
	m_data, err := json.Marshal(content)
	if err != nil {
		log.Printf("JsonResponce Marshal err: %s\n", err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(code)
	w.Write(m_data)
}

func CleanChirp(s string) string {
	Filter := []string{
		"kerfuffle",
		"sharbert",
		"fornax",
	}
	Chirp_slice := strings.Split(s, " ")
	for i := 0; i < len(Chirp_slice); i++ {
		for _, bad_word := range Filter {
			lower := strings.ToLower(Chirp_slice[i])
			if lower == bad_word {
				Chirp_slice[i] = "****"
			}
		}
	}
	return strings.Join(Chirp_slice, " ")
}

func HealthStatus(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	res.WriteHeader(200)
	res.Write([]byte("OK"))
}
