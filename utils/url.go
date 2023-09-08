package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"regexp"
	"sync"
)

var HOSTNAME string

const (
	DATA_PATH  = "urls.json"
	SHORT_KEYS = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-"
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
	reURL       = regexp.MustCompile(`^(https?://)([\S]+\.)?([^\s/]+\.[^\s/]{2,})(/?[\S]+)?$`)
	reCustomURL = regexp.MustCompile(`^([\w\-]{1,32})$`)

	// customURL blacklist
	customURLBlacklist = []string{"api", "dashboard"}
)

func init() {
	HOSTNAME = os.Getenv("HOSTNAME")
	updateCacheURLData()
}

// Custom Meta
type CustomMeta struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	ImageURL    string `json:"image"`
	ThemeColor  string `json:"color"`
}

func (meta *CustomMeta) ImageURLIsValid() bool {
	return reURL.MatchString(meta.ImageURL)
}

// Shorten URL Data
type URLData struct {
	ShortURL  ShortURL    `json:"short"`
	TargetURL LongURL     `json:"url"`
	Meta      *CustomMeta `json:"meta"`
	Count     int         `json:"count"`
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

// API Requests Data
type CreateData struct {
	URL       LongURL     `json:"url"`
	CustomURL ShortURL    `json:"customUrl"`
	Meta      *CustomMeta `json:"meta"`
}

func (data *CreateData) CreateShortURL() URLData {
	longURL := data.URL
	shortURL := data.CustomURL
	if shortURL == "" {
		shortURL = summonShortURL()
	}

	urlData := URLData{
		ShortURL:  shortURL,
		TargetURL: longURL,
		Meta:      data.Meta,
	}
	urlCache[shortURL] = urlData
	longURLCache[longURL] = shortURL
	saveCacheURLData()

	return urlData
}

func (d *CreateData) InsertMeta() error {
	data, err := ExtractHtmlMetaFromURL(string(d.URL))
	if err != nil {
		log.Println("get meta error:", err)
		return err
	}

	meta := d.Meta
	if meta.Title == "" {
		meta.Title = data.Title
	}
	if meta.Description == "" {
		meta.Description = data.Description
	}
	if meta.ImageURL == "" {
		meta.ImageURL = data.Image
	}
	if meta.ThemeColor == "" {
		meta.ThemeColor = data.ThemeColor
	}

	return nil
}

func (shortURL ShortURL) GetData() (urlData URLData, ok bool) {
	urlData, ok = urlCache[shortURL]
	return
}

func (shortURL ShortURL) IsValid() error {
	if len(string(shortURL)) > 32 {
		return errors.New("custom url is too long")
	}
	if !reCustomURL.MatchString(string(shortURL)) {
		return errors.New("illegal custom url, only support [a-zA-Z0-9_-]")
	}
	for _, blacklist := range customURLBlacklist {
		if string(shortURL) == blacklist {
			return errors.New("illegal custom url, you cannot use " + blacklist + " as custom url")
		}
	}
	return nil
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

func (longURL LongURL) IsValid() error {
	match := reURL.FindStringSubmatch(string(longURL))
	if len(match) == 0 {
		return errors.New("invalid url format")
	}
	if (match[2] + match[3]) == HOSTNAME {
		return errors.New("illegal url, you cannot redirect to " + HOSTNAME)
	}
	return nil
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
