package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

type Error_json struct {
	Err string `json:"error"`
}

type Valid_chirp struct {
	Valid        bool   `json:"valid"`
	Cleaned_body string `json:"cleaned_body"`
}

type Chirp struct {
	Body string `json:"body"`
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

func ValidateChirp(res http.ResponseWriter, req *http.Request) {
	chirp_body := Chirp{}
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
	valid := Valid_chirp{
		Valid:        true,
		Cleaned_body: CleanChirp(chirp_body.Body),
	}
	JsonResponce(res, 200, valid)
}

func HealthStatus(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	res.WriteHeader(200)
	res.Write([]byte("OK"))
}
