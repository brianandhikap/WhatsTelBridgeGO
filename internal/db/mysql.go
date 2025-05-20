// file db.go
package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

// Topic struktur yang menyimpan informasi topic
type Topic struct {
	ID             int64
	WANumber       string // WhatsApp number
	ContactName    string
	TelegramTopicID int64
}

// InitDB inisialisasi koneksi database
func InitDB() {
	var err error
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "wa_bridge.db"
	}

	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Buat tabel jika belum ada
	createTables()
}

// createTables membuat tabel yang diperlukan
func createTables() {
	// Tabel untuk menyimpan topic
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS topics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		wa_number TEXT UNIQUE,
		contact_name TEXT,
		telegram_topic_id INTEGER UNIQUE
	)
	`)
	if err != nil {
		log.Fatal("Failed to create topics table:", err)
	}

	// Tabel untuk menyimpan user
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS users (
		telegram_id INTEGER PRIMARY KEY,
		initial TEXT
	)
	`)
	if err != nil {
		log.Fatal("Failed to create users table:", err)
	}
}

// GetTopic mendapatkan informasi topic berdasarkan nomor WhatsApp
func GetTopic(waNumber string) (*Topic, error) {
	row := db.QueryRow("SELECT id, wa_number, contact_name, telegram_topic_id FROM topics WHERE wa_number = ?", waNumber)
	
	var topic Topic
	err := row.Scan(&topic.ID, &topic.WANumber, &topic.ContactName, &topic.TelegramTopicID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &topic, nil
}

// GetTopicByTelegramTopicID mendapatkan informasi topic berdasarkan ID topic Telegram
func GetTopicByTelegramTopicID(telegramTopicID int64) (*Topic, error) {
	row := db.QueryRow("SELECT id, wa_number, contact_name, telegram_topic_id FROM topics WHERE telegram_topic_id = ?", telegramTopicID)
	
	var topic Topic
	err := row.Scan(&topic.ID, &topic.WANumber, &topic.ContactName, &topic.TelegramTopicID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &topic, nil
}

// SaveTopic menyimpan informasi topic
func SaveTopic(waNumber string, contactName string, telegramTopicID int64) error {
	_, err := db.Exec(
		"INSERT INTO topics (wa_number, contact_name, telegram_topic_id) VALUES (?, ?, ?)",
		waNumber, contactName, telegramTopicID,
	)
	return err
}

// DeleteTopic menghapus topic berdasarkan nomor WhatsApp
func DeleteTopic(waNumber string) error {
	_, err := db.Exec("DELETE FROM topics WHERE wa_number = ?", waNumber)
	return err
}

// AddUser menambahkan user baru
func AddUser(telegramID int64, initial string) error {
	_, err := db.Exec(
		"INSERT INTO users (telegram_id, initial) VALUES (?, ?) "+
		"ON CONFLICT(telegram_id) DO UPDATE SET initial = ?",
		telegramID, initial, initial,
	)
	return err
}

// RemoveUser menghapus user
func RemoveUser(telegramID int64) error {
	_, err := db.Exec("DELETE FROM users WHERE telegram_id = ?", telegramID)
	return err
}

// GetAllUsers mendapatkan semua user
func GetAllUsers() (map[int64]string, error) {
	rows, err := db.Query("SELECT telegram_id, initial FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make(map[int64]string)
	for rows.Next() {
		var id int64
		var initial string
		if err := rows.Scan(&id, &initial); err != nil {
			return nil, err
		}
		users[id] = initial
	}

	return users, nil
}

// CloseDB menutup koneksi database
func CloseDB() {
	if db != nil {
		db.Close()
	}
}
