package db

import (
    "database/sql"
    "log"
)

type User struct {
    TelegramID int64
    Initial string
}

func AddUser(id int64, initial string) error {
    _, err := DB.Exec("INSERT INTO users (telegram_id, initial) VALUES (?, ?) ON DUPLICATE KEY UPDATE initial = VALUES(initial)", id, initial)
    return err
}

func RemoveUser(id int64) error {
    _, err := DB.Exec("DELETE FROM users WHERE telegram_id = ?", id)
    return err
}

func GetAllUsers() ([]User, error) {
    rows, err := DB.Query("SELECT telegram_id, initial FROM users")
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var users []User
    for rows.Next() {
        var u User
        if err := rows.Scan(&u.TelegramID, &u.Initial); err != nil {
            log.Println("Error scanning user:", err)
            continue
        }
        users = append(users, u)
    }

    return users, nil
}

func GetUserInitial(id int64) (string, error) {
    var initial string
    err := DB.QueryRow("SELECT initial FROM users WHERE telegram_id = ?", id).Scan(&initial)
    if err == sql.ErrNoRows {
        return "", nil
    }
    return initial, err
}
