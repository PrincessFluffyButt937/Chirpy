package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
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
		ID:        db_user.ID,
		CreatedAt: db_user.CreatedAt,
		UpdatedAt: db_user.UpdatedAt,
		Email:     db_user.Email,
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
	user_json := User{
		ID:        db_user.ID,
		CreatedAt: db_user.CreatedAt,
		UpdatedAt: db_user.UpdatedAt,
		Email:     db_user.Email,
	}
	JsonResponce(res, 200, user_json)
}

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

	now := time.Now()
	chirp_param := database.CreateChirpParams{
		ID:        uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
		Body:      chirp_body.Body,
		UserID:    chirp_body.User_id,
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
	chirps, err := cfg.db.GetChirps(req.Context())
	if err != nil {
		msg := fmt.Sprintf("DB_GetChirps: %s", err)
		ErrorResponce(res, 500, msg)
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
	JsonResponce(res, 200, Chirp_L)
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
