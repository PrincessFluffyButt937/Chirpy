package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/PrincessFluffyButt937/Chirpy/internal/database"

	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	queries        *database.Queries
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

func (cfg *apiConfig) MetricsReset() http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		cfg.fileserverHits.Store(0)
		res.Header().Set("Content-Type", "text/plain; charset=utf-8")
		res.WriteHeader(200)
		res.Write([]byte("Server Hits reset to 0."))
	})
}

func main() {
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
		queries: database.New(db),
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
	ServeMux.HandleFunc("POST /api/validate_chirp", ValidateChirp)

	server := http.Server{
		Handler: ServeMux,
		Addr:    ":8080",
	}
	server.ListenAndServe()
}
