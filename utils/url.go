package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"sync"
)

var HOSTNAME string

const (
	DATA_PATH  = "urls.json"
	SHORT_KEYS = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

type (
	LongURL  string
	ShortURL string
)

var (
	SHORT_LEN = len(SHORT_KEYS)
	// short -> long
	// [k: ShortURL as string]: LongURL as URLData struct
	urlCache = map[ShortURL]URLData{}
	// long -> short
	// [k: LongURL as string]: ShortURL as string
	longURLCache = map[LongURL]ShortURL{}
	fileLock     sync.Mutex
	// URL validation regex
	reURL = regexp.MustCompile(`(https?://)([\S]+\.)?([^\s/]+\.[^\s/]{2,})(/?[\S]+)?`)
)

func init() {
	HOSTNAME = os.Getenv("HOSTNAME")
	updateCacheURLData()
}

type URLData struct {
	ShortURL    ShortURL `json:"-"`
	TargetURL   LongURL  `json:"url"`
	Count       int      `json:"count"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	ImageURL    string   `json:"image"`
}

type CreateData struct {
	URL         LongURL  `json:"url" binding:"required"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	CustomURL   ShortURL `json:"customUrl"`
}

func (urlData *URLData) IncreaseCount() error {
	urlData.Count++
	urlCache[urlData.ShortURL] = *urlData

	return saveCacheURLData()
}

func summonShortURL() ShortURL {
	shortURL := ""
	for i := 0; i < 6; i++ {
		shortURL += string(SHORT_KEYS[rand.Intn(SHORT_LEN)])
	}

	if _, ok := urlCache[ShortURL(shortURL)]; ok {
		return summonShortURL()
	}

	return ShortURL(shortURL)
}

func (data CreateData) CreateShortURL() URLData {
	longURL := data.URL
	shortURL := data.CustomURL
	if len(shortURL) == 0 {
		shortURL = summonShortURL()
	}

	urlData := URLData{
		ShortURL:    shortURL,
		TargetURL:   longURL,
		Count:       0,
		Title:       data.Title,
		Description: data.Description,
		// ImageURL:    data.ImageURL,
	}
	urlCache[shortURL] = urlData
	longURLCache[longURL] = shortURL
	saveCacheURLData()

	return urlData
}

func (shortURL ShortURL) GetData() (urlData URLData, ok bool) {
	urlData, ok = urlCache[shortURL]
	return
}

func (longURL LongURL) GetShortURL() (shortURL ShortURL, ok bool) {
	shortURL, ok = longURLCache[longURL]
	return
}

func (longURL LongURL) GetData() (URLData, bool) {
	shortUrl, ok := longURL.GetShortURL()
	if !ok {
		return URLData{}, false
	}

	return shortUrl.GetData()
}

func IsValidURL(addr string) (bool, error) {
	match := reURL.FindStringSubmatch(addr)
	if len(match) == 0 {
		return false, errors.New("invalid URL format")
	}
	if match[3] == HOSTNAME {
		return false, errors.New("illegal URL")
	}
	return true, nil
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
