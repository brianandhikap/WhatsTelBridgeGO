package db

import "errors"

type Topic struct {
    WaNumber string
    ContactName string
    TelegramTopicID int64
}

func GetTopicByTelegramID(topicID int64) (*Topic, error) {
    var t Topic
    row := DB.QueryRow("SELECT wa_number, contact_name, telegram_topic_id FROM topics WHERE telegram_topic_id = ? AND status = 'open'", topicID)
    err := row.Scan(&t.WaNumber, &t.ContactName, &t.TelegramTopicID)
    if err != nil {
        return nil, errors.New("topic not found")
    }
    return &t, nil
}
