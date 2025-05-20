package main

import (
    "log"
    "wa-bridge/internal/bot"
    "wa-bridge/internal/db"

    "github.com/joho/godotenv"
)

func main() {
    err := godotenv.Load()
    if err != nil {
        log.Fatal("Error loading .env file")
    }

    db.InitDB()
    bot.StartBot()
}
