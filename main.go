package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/PrincessFluffyButt937/Chirpy/internal/database"
	"github.com/joho/godotenv"

	_ "github.com/lib/pq"
)

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Printf("DB err: %s", err)
		fmt.Printf("Database connection failed: %s\n", err)
		fmt.Println("Server startup aborted.")
		db.Close()
		return
	}
	defer db.Close()
	cfg := apiConfig{
		db:       database.New(db),
		platform: os.Getenv("PLATFORM"),
		secret:   os.Getenv("SECRET_STRING"),
	}
	cfg.fileserverHits.Store(0)
	ServeMux := http.NewServeMux()
	FileSys := http.Dir(".")
	FileServerHandler := http.FileServer(FileSys)
	StrFileServerHandler := http.StripPrefix("/app", FileServerHandler)
	ServeMux.Handle("/app/", cfg.middlewareMetricsInc(StrFileServerHandler))
	ServeMux.HandleFunc("GET /api/healthz", HealthStatus)
	ServeMux.Handle("GET /admin/metrics", cfg.MetricsTotal())
	ServeMux.Handle("POST /admin/reset", cfg.MetricsReset())
	ServeMux.HandleFunc("POST /api/chirps", cfg.New_Chirp_handler)
	ServeMux.HandleFunc("GET /api/chirps", cfg.Get_Chirps_handler)
	ServeMux.HandleFunc("GET /api/chirps/{chirpID}", cfg.Get_Chirp_By_ID_handler)
	ServeMux.HandleFunc("DELETE /api/chirps/{chirpID}", cfg.Delete_Chirp_By_ID_handler)
	ServeMux.HandleFunc("POST /api/users", cfg.Create_user_handler)
	ServeMux.HandleFunc("PUT /api/users", cfg.Update_user_email_password_handler)
	ServeMux.HandleFunc("POST /api/login", cfg.Login_user_handler)
	ServeMux.HandleFunc("POST /api/refresh", cfg.Refresh_access_token_handler)
	ServeMux.HandleFunc("POST /api/revoke", cfg.Revoke_refresh_token_handler)

	server := http.Server{
		Handler: ServeMux,
		Addr:    ":8080",
	}
	server.ListenAndServe()
}
