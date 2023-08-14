package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"sync"
)

type URLData struct {
	TargetURL   string `json:"url"`
	Count       int    `json:"count"`
	Title       string `json:"title"`
	Description string `json:"description"`
	ImageURL    string `json:"image"`
}

const (
	DATA_PATH  = "urls.json"
	SHORT_KEYS = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

var (
	SHORT_LEN = len(SHORT_KEYS)
	// short -> long
	// [k: ShortURL as string]: LongURL as URLData struct
	urlCache = map[string]URLData{}
	// long -> short
	// [k: LongURL as string]: ShortURL as string
	longURLCache = map[string]string{}
	fileLock     sync.Mutex
	// URL validation regex
	reURL = regexp.MustCompile(`(https?://)([\S]+\.)?([^\s/]+\.[^\s/]{2,}/?)([\S]+)?`)
)

func init() {
	updateCacheURLData()
}

func updateCacheURLData() (err error) {
	fileLock.Lock()
	defer fileLock.Unlock()

	fileContent, err := os.ReadFile(DATA_PATH)
	if err != nil && !os.IsNotExist(err) {
		fmt.Println("Error reading data file:", err)
		return
	}

	if err = json.Unmarshal(fileContent, &urlCache); err != nil {
		fmt.Println("Error parsing JSON:", err)
		return
	}

	// Update longURLCache
	for short, long := range urlCache {
		longURLCache[long.TargetURL] = short
	}

	return
}

func saveCacheURLData() (err error) {
	fileLock.Lock()
	defer fileLock.Unlock()

	data, err := json.Marshal(urlCache)
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return
	}

	if err = os.WriteFile(DATA_PATH, data, 0o644); err != nil {
		fmt.Println("Error writing data file:", err)
		return
	}

	return
}

func SummonShortURL() string {
	shortURL := ""
	for i := 0; i < 6; i++ {
		shortURL += string(SHORT_KEYS[rand.Intn(SHORT_LEN)])
	}

	if _, ok := urlCache[shortURL]; ok {
		return SummonShortURL()
	}

	return shortURL
}

func CreateShortULR(data *CreateData) string {
	longURL := data.URL
	if shortURL, ok := longURLCache[longURL]; ok {
		return shortURL
	}
	if _, ok := urlCache[data.CustomURL]; ok {
		return ""
	}

	shortURL := data.CustomURL
	if len(shortURL) == 0 {
		shortURL = SummonShortURL()
	}

	urlCache[shortURL] = URLData{
		TargetURL:   longURL,
		Count:       0,
		Title:       data.Title,
		Description: data.Description,
	}
	longURLCache[longURL] = shortURL
	saveCacheURLData()

	return shortURL
}

func GetURL(shortURL string) (urlData URLData, ok bool) {
	urlData, ok = urlCache[shortURL]
	return
}

func isValidURL(addr string) bool {
	return reURL.MatchString(addr)
}

func (urlData *URLData) increaseCount(shortURL string) (err error) {
	urlData.Count++
	urlCache[shortURL] = *urlData
	err = saveCacheURLData()

	return
}
