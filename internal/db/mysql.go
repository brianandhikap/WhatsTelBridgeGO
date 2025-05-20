package db

import (
    "database/sql"
    "fmt"
    "log"
    "os"

    _ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

func Init() {
    dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s",
        os.Getenv("MYSQL_USER"),
        os.Getenv("MYSQL_PASS"),
        os.Getenv("MYSQL_HOST"),
        os.Getenv("MYSQL_DB"))

    var err error
    DB, err = sql.Open("mysql", dsn)
    if err != nil {
        log.Fatalf("Failed to connect DB: %v", err)
    }

    if err := DB.Ping(); err != nil {
        log.Fatalf("DB ping failed: %v", err)
    }

    fmt.Println("Connected to MySQL!")
}
