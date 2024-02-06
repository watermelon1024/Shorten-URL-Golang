package utils

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	"github.com/compose-spec/compose-go/dotenv"
)

// sqlite database
var db *sql.DB

func init() {
	dotenv.Load()
	dbFilePath := os.Getenv("DB_PATH")

	var err error
	// check/create database dir
	dir := filepath.Dir(dbFilePath)
	if _, err = os.Stat(dir); os.IsNotExist(err) {
		// dir not exist, create it
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			log.Fatalln("Error creating directory:", dir)
		}
	}
	// check/create database file
	if _, err = os.Stat(dbFilePath); os.IsNotExist(err) {
		// file not exist, create it
		file, err := os.Create(dbFilePath)
		if err != nil {
			log.Fatalln("Error creating database file:", err)
		}
		file.Close()
	}
	// connect to database
	db, err = sql.Open("sqlite3", dbFilePath)
	if err != nil {
		log.Fatalln("Error opening database:", err)
	}
	// create tables
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS urls (
		id TEXT PRIMARY KEY,
		target_url TEXT NOT NULL,
		meta TEXT,
		count INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		created_by TEXT,
		ip TEXT,
		expired_at DATETIME
	)`)
	if err != nil {
		log.Fatalln("Error creating table:", err)
	}
}

func CloseDB() error {
	return db.Close()
}
