package main

import (
	"fmt"
	"net/http"
	"math/rand"
	"time"
	"strings"
	"github.com/gorilla/mux"
)

var (
	urlMap = make(map[string]string)
	baseURL = "http://localhost:8080/"
)

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/", homeHandler)
	r.HandleFunc("/shorten", shortenHandler)
	r.HandleFunc("/{shortURL}", redirectHandler)

	http.Handle("/", r)

	fmt.Println("Server started on :8080")
	http.ListenAndServe(":8080", nil)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Welcome to the URL shortener service!"))
}

func shortenHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	longURL := r.Form.Get("url")

	if longURL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	shortURL := generateShortURL()
	urlMap[shortURL] = longURL

	shortenedURL := baseURL + shortURL
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Shortened URL: " + shortenedURL))
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shortURL := vars["shortURL"]

	longURL, exists := urlMap[shortURL]
	if !exists {
		http.NotFound(w, r)
		return
	}

	http.Redirect(w, r, longURL, http.StatusSeeOther)
}

func generateShortURL() string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	rand.Seed(time.Now().UnixNano())
	var sb strings.Builder

	for i := 0; i < 6; i++ {
		sb.WriteByte(letters[rand.Intn(len(letters))])
	}

	return sb.String()
}
