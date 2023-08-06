package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/mux"
)

var (
	urlMap   = make(map[string]string)
	baseURL  = "http://localhost:8080/"
	dataFile = "url_data.json"
	mu       sync.Mutex
)

const (
	letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

type URLData struct {
	ShortURL string `json:"short_url"`
	LongURL  string `json:"long_url"`
}

func main() {
	loadURLData()

	r := mux.NewRouter()
	r.HandleFunc("/", homeHandler)
	r.HandleFunc("/shorten", shortenHandler)
	r.HandleFunc("/{shortURL}", redirectHandler)

	http.Handle("/", r)

	fmt.Println("Server started on :8080")

	// 創建一個關閉通道，用於在伺服器關閉時通知
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		if err := http.ListenAndServe(":8080", nil); err != nil {
			fmt.Println("Error starting server:", err)
			shutdown <- syscall.SIGTERM
		}
	}()

	<-shutdown    // 等待關閉通知
	saveURLData() // 在伺服器關閉前保存資料
}

func loadURLData() {
	if _, err := os.Stat(dataFile); os.IsNotExist(err) {
		return
	}

	fileContent, err := os.ReadFile(dataFile)
	if err != nil {
		fmt.Println("Error reading data file:", err)
		return
	}

	var savedURLs []URLData
	err = json.Unmarshal(fileContent, &savedURLs)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		return
	}

	for _, urlData := range savedURLs {
		urlMap[urlData.ShortURL] = urlData.LongURL
	}
}

func saveURLData() {
	mu.Lock()
	defer mu.Unlock()

	var savedURLs []URLData
	for shortURL, longURL := range urlMap {
		savedURLs = append(savedURLs, URLData{ShortURL: shortURL, LongURL: longURL})
	}

	data, err := json.MarshalIndent(savedURLs, "", "    ")
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return
	}

	err = os.WriteFile(dataFile, data, 0644)
	if err != nil {
		fmt.Println("Error writing data file:", err)
	}
}
func log(r *http.Request, StatusCode int) {
	requestTime := time.Now().Format("2006-01-02 15:04:05")

	fmt.Printf(`%s - %s - "%s %s %s" - %d\n`, r.RemoteAddr, requestTime, r.Method, r.URL.Path, r.Proto, StatusCode)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Welcome to the URL shortener service!"))
	log(r, http.StatusOK)
}

func shortenHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	longURL := r.Form.Get("url")

	if longURL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		log(r, http.StatusBadRequest)
		return
	}

	if !strings.HasPrefix(longURL, "http://") && !strings.HasPrefix(longURL, "https://") {
		http.Error(w, "URL must start with http:// or https://", http.StatusBadRequest)
		log(r, http.StatusBadRequest)
		return
	}

	shortURL := generateShortURL()
	urlMap[shortURL] = longURL

	shortenedURL := baseURL + shortURL
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Shortened URL: " + shortenedURL))
	log(r, http.StatusOK)
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shortURL := vars["shortURL"]

	longURL, exists := urlMap[shortURL]
	if !exists {
		http.NotFound(w, r)
		log(r, http.StatusNotFound)
		return
	}

	http.Redirect(w, r, longURL, http.StatusSeeOther)
	log(r, http.StatusTemporaryRedirect)
}

func generateShortURL() string {
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)

	var sb strings.Builder

	for i := 0; i < 6; i++ {
		sb.WriteByte(letters[r.Intn(len(letters))])
	}

	return sb.String()
}
