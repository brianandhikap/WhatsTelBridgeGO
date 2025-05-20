package db

import (
    "database/sql"
    "log"
)

type Topic struct {
    WANumber string
    ContactName string
    TelegramTopicID int64
}

func GetTopic(waNumber string) (*Topic, error) {
    row := DB.QueryRow("SELECT wa_number, contact_name, telegram_topic_id FROM topics WHERE wa_number = ?", waNumber)
    t := &Topic{}
    err := row.Scan(&t.WANumber, &t.ContactName, &t.TelegramTopicID)
    if err == sql.ErrNoRows {
        return nil, nil
    } else if err != nil {
        return nil, err
    }
    return t, nil
}

func SaveTopic(waNumber, name string, topicID int64) error {
    _, err := DB.Exec("INSERT INTO topics (wa_number, contact_name, telegram_topic_id) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE telegram_topic_id = VALUES(telegram_topic_id), contact_name = VALUES(contact_name)", waNumber, name, topicID)
    if err != nil {
        log.Printf("Error saving topic: %v", err)
    }
    return err
}

func DeleteTopic(waNumber string) error {
    _, err := DB.Exec("DELETE FROM topics WHERE wa_number = ?", waNumber)
    return err
}
