package main

import (
    "wa-bridge/internal/bot"
    "wa-bridge/internal/wa"
    "wa-bridge/internal/db"

    "github.com/joho/godotenv"
    "log"
)

func main() {
    err := godotenv.Load()
    if err != nil {
        log.Fatal("Error loading .env file")
    }

    db.Init()
    go wa.StartWA()  // WhatsApp
    bot.StartBot()   // Telegram bot
}
