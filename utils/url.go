package utils

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"math/rand"
	"os"
	"regexp"

	"github.com/compose-spec/compose-go/dotenv"
	"github.com/mattn/go-sqlite3"
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

	// URL validation regex
	reURL       = regexp.MustCompile(`^(https?://)([\S]+\.)?([^\s/]+\.[^\s/]{2,})(/?[\S]+)?$`)
	reCustomURL = regexp.MustCompile(`^([\w\-]{1,32})$`)

	// customURL blacklist
	customURLBlacklist = []string{"api", "dashboard"}
)

func init() {
	dotenv.Load()
	HOSTNAME = os.Getenv("HOSTNAME")
}

// Custom Meta Data
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
	_, err := db.Exec("UPDATE urls SET count = count + 1 WHERE id = ?", string(urlData.ShortURL))
	return err
}

// API Requests Data
type CreateData struct {
	URL       LongURL     `json:"url"`
	CustomURL ShortURL    `json:"customUrl"`
	Meta      *CustomMeta `json:"meta"`
}

// Create a short URL
func (data *CreateData) CreateShortURL() (*URLData, error) {
	longURL := data.URL
	var metaString any = nil
	if data.Meta != nil {
		metaBytes, err := json.Marshal(data.Meta)
		if err != nil {
			return nil, err
		}
		metaString = string(metaBytes)
	}
	shortURL, err := data.createShortURL(metaString)
	if err != nil {
		return nil, err
	}

	return &URLData{
		ShortURL:  ShortURL(shortURL),
		TargetURL: longURL,
		Meta:      data.Meta,
	}, nil
}

// Create a short URL (inner function)
func (data *CreateData) createShortURL(meta any) (string, error) {
	shortURL := ""
	if data.CustomURL == "" {
		for i := 0; i < 6; i++ {
			shortURL += string(SHORT_KEYS[rand.Intn(SHORT_LEN)])
		}
	} else {
		shortURL = string(data.CustomURL)
	}

	_, err := db.Exec("INSERT INTO urls (id, target_url, meta) VALUES (?, ?, ?)",
		shortURL, data.URL, meta)
	if err != nil {
		if errors.Is(err, sqlite3.ErrConstraint) {
			return data.createShortURL(meta)
		}
		if sqliteErr, ok := err.(sqlite3.Error); ok {
			errors.Is(sqliteErr, sqlite3.ErrConstraintUnique)
			if sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique ||
				sqliteErr.ExtendedCode == sqlite3.ErrConstraintPrimaryKey {
				return data.createShortURL(meta)
			}
		}
		return "", err
	}

	return shortURL, nil
}

// Insert meta into short url
func (data *CreateData) InsertMeta() error {
	htmlMeta, err := ExtractHtmlMetaFromURL(string(data.URL))
	if err != nil {
		log.Println("Error getting meta:", err)
		return err
	}

	meta := data.Meta
	if meta.Title == "" {
		meta.Title = htmlMeta.Title
	}
	if meta.Description == "" {
		meta.Description = htmlMeta.Description
	}
	if meta.ImageURL == "" {
		meta.ImageURL = htmlMeta.Image
	}
	if meta.ThemeColor == "" {
		meta.ThemeColor = htmlMeta.ThemeColor
	}

	return nil
}

// ShortURL functions

// Get url data from database
func (shortURL ShortURL) GetData() (urlData *URLData, err error) {
	var (
		id         string
		target_url string
		meta       sql.NullString
		count      int
		created_at sql.NullString
		created_by sql.NullString
		ip         sql.NullString
		expired_at sql.NullString
	)
	row := db.QueryRow("SELECT * FROM urls WHERE id = ?", string(shortURL))
	err = row.Scan(&id, &target_url, &meta, &count, &created_at, &created_by, &ip, &expired_at)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// not found
			return nil, nil
		}
		return nil, err
	}

	var customMeta *CustomMeta = &CustomMeta{}
	if meta.Valid {
		json.Unmarshal([]byte(meta.String), customMeta)
	} else {
		customMeta = nil
	}

	return &URLData{
		ShortURL:  ShortURL(id),
		TargetURL: LongURL(target_url),
		Meta:      customMeta,
		Count:     count,
		// CreatedAt: created_at,
		// CreatedBy: created_by,
		// ExpiredAt: expired_at,
	}, nil
}

// check if short url format is valid
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

// LongURL functions

// Check if long url meta which is in database is same as create data meta
func (longURL LongURL) CheckMetaSame(data CreateData) (urlData *URLData, err error) {
	rows, err := db.Query("SELECT id, meta FROM urls WHERE target_url = ?", string(longURL))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var CreateMeta string
	if data.Meta == nil {
		CreateMeta = ""
	} else {
		metaBytes, err := json.Marshal(data.Meta)
		if err != nil {
			return nil, err
		}
		CreateMeta = string(metaBytes)
	}

	for rows.Next() {
		var (
			id   string
			meta sql.NullString
		)
		err := rows.Scan(&id, &meta)
		if err != nil {
			log.Println("Error found meta:", err)
			continue
		}
		if meta.String == CreateMeta {
			return &URLData{ShortURL: ShortURL(id)}, nil
		}
	}

	return nil, nil
}

// Check if long url format is valid
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
