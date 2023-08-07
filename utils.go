package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"net/url"
	"os"
	"strings"
	"sync"
)

const (
	DATA_PATH  = "urls.json"
	SHORT_KEYS = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

var (
	SHORT_LEN = len(SHORT_KEYS)
	// short -> long
	// [k: ShortURL as string]: LongURL as string
	urlCache = map[string]string{}
	// long -> short
	// [k: LongURL as string]: ShortURL as string
	longURLCache = map[string]string{}
	fileLock     sync.Mutex
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
		longURLCache[long] = short
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

func CreateShortULR(longURL string) string {
	if shortURL, ok := longURLCache[longURL]; ok {
		return shortURL
	}

SUMMON:
	shortURL := ""
	for i := 0; i < 6; i++ {
		shortURL += string(SHORT_KEYS[rand.Intn(SHORT_LEN)])
	}

	if _, ok := urlCache[shortURL]; ok {
		goto SUMMON
	}

	urlCache[shortURL] = longURL
	longURLCache[longURL] = shortURL
	saveCacheURLData()

	return shortURL
}

func GetURL(shortURL string) (longURL string, ok bool) {
	longURL, ok = urlCache[shortURL]
	return
}

func HasIsURL(addr string) bool {
	url, err := url.ParseRequestURI(addr)
	fmt.Println(err)
	if err != nil || !strings.HasPrefix(url.Scheme, "http") {
		return false
	}

	if net.ParseIP(url.Host) == nil {
		return strings.Contains(url.Host, ".")
	}

	return true
}
