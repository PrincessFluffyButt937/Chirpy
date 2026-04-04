package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
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

func HealthStatus(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	res.WriteHeader(200)
	res.Write([]byte("OK"))
}

var cfg apiConfig

func main() {
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
