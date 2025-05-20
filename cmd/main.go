// file main.go
package main

import (
	"log"
	"os"
	"time"

	"wa-bridge/internal/bot"
	"wa-bridge/internal/db"
	"wa-bridge/internal/wa"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	// Initialize database
	db.InitDB()
	defer db.CloseDB()

	// Start WhatsApp client
	go wa.StartWA()

	// Allow some time for WhatsApp to connect
	time.Sleep(2 * time.Second)

	// Start Telegram bot
	bot.StartBot()
}
