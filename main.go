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
	dbPLATFORM := os.Getenv("PLATFORM")
	fmt.Printf("url: %s\nplat: %s\n", dbURL, dbPLATFORM)
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
		platform: dbPLATFORM,
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
	ServeMux.HandleFunc("POST /api/users", cfg.Create_user_handler)

	server := http.Server{
		Handler: ServeMux,
		Addr:    ":8080",
	}
	server.ListenAndServe()
}
