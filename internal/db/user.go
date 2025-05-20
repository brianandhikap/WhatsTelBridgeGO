package db

import "fmt"

func AddUser(id int64, initial string) error {
    _, err := DB.Exec("REPLACE INTO users (telegram_id, initial) VALUES (?, ?)", id, initial)
    return err
}

func RemoveUser(id int64) error {
    _, err := DB.Exec("DELETE FROM users WHERE telegram_id = ?", id)
    return err
}

func GetUserByID(id int64) (string, error) {
    var initial string
    err := DB.QueryRow("SELECT initial FROM users WHERE telegram_id = ?", id).Scan(&initial)
    return initial, err
}
