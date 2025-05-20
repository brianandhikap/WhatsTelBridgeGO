package bot

import (
    "fmt"
    "log"
    "os"
    "strconv"
    "strings"
    "sync"

    "wa-bridge/internal/db"

    tele "github.com/tucnak/telebot"
)

var (
    bot        *tele.Bot
    topicGroup int64
    fullGroup  int64
    superadmins map[int64]string
    users      map[int64]string // telegram_id => initial

    topicsLock sync.Mutex
    // mapping topicID Telegram -> Topic (db.Topic)
    activeTopics map[int64]*db.Topic
)

func StartBot() {
    token := os.Getenv("TELEGRAM_BOT_TOKEN")
    var err error
    bot, err = tele.NewBot(tele.Settings{
        Token: token,
        Poller: &tele.LongPoller{Timeout: 10},
    })
    if err != nil {
        log.Fatal("Failed to start bot:", err)
    }

    topicGroup, _ = strconv.ParseInt(os.Getenv("TELEGRAM_TOPIC_GROUP"), 10, 64)
    fullGroup, _ = strconv.ParseInt(os.Getenv("TELEGRAM_FULL_GROUP"), 10, 64)

    superadmins = make(map[int64]string)
    for _, sa := range strings.Split(os.Getenv("SUPERADMINS"), ",") {
        if sa == "" {
            continue
        }
        id, _ := strconv.ParseInt(sa, 10, 64)
        superadmins[id] = ""
    }

    users = make(map[int64]string) // loaded lazily on demand or after addUser

    activeTopics = make(map[int64]*db.Topic)

    bot.Handle(tele.OnText, handleText)
    log.Println("Telegram bot started")
    bot.Start()
}

func handleText(c tele.Context) error {
    sender := c.Sender()
    senderId := sender.ID

    text := c.Text()
    lowerText := strings.ToLower(text)

    // Only process commands starting with '!'
    if !strings.HasPrefix(lowerText, "!") {
        // If message is reply in a topic group, treat as message reply
        if c.Chat().ID == topicGroup && c.Message().ReplyTo != nil {
            return handleReplyMessage(c, senderId)
        }
        return nil
    }

    // Parse command
    parts := strings.Fields(text)
    cmd := strings.ToLower(parts[0])

    switch cmd {
    case "!add":
        return cmdAdd(c, senderId, parts)
    case "!rm":
        return cmdRemove(c, senderId, parts)
    case "!chat":
        return cmdChat(c, senderId, parts)
    case "!close":
        return cmdClose(c, senderId)
    default:
        return c.Reply("Perintah tidak dikenal")
    }
}

// !add <telegram_id> <initial>
func cmdAdd(c tele.Context, senderId int64, parts []string) error {
    if !isSuperadmin(senderId) {
        return c.Reply("Anda tidak memiliki izin.")
    }
    if len(parts) < 3 {
        return c.Reply("Format: !add <telegram_id> <initial>")
    }
    id, err := strconv.ParseInt(parts[1], 10, 64)
    if err != nil {
        return c.Reply("ID Telegram tidak valid")
    }
    initial := parts[2]
    err = db.AddUser(id, initial)
    if err != nil {
        return c.Reply("Gagal menambahkan user: " + err.Error())
    }
    users[id] = initial
    return c.Reply(fmt.Sprintf("User %d dengan inisial %s berhasil ditambahkan", id, initial))
}

// !rm <telegram_id>
func cmdRemove(c tele.Context, senderId int64, parts []string) error {
    if !isSuperadmin(senderId) {
        return c.Reply("Anda tidak memiliki izin.")
    }
    if len(parts) < 2 {
        return c.Reply("Format: !rm <telegram_id>")
    }
    id, err := strconv.ParseInt(parts[1], 10, 64)
    if err != nil {
        return c.Reply("ID Telegram tidak valid")
    }
    err = db.RemoveUser(id)
    if err != nil {
        return c.Reply("Gagal menghapus user: " + err.Error())
    }
    delete(users, id)
    return c.Reply(fmt.Sprintf("User %d berhasil dihapus", id))
}

// !chat <nomor> <pesan...>
func cmdChat(c tele.Context, senderId int64, parts []string) error {
    if len(parts) < 3 {
        return c.Reply("Format: !chat <nomor> <pesan>")
    }
    nomor := parts[1]
    pesan := strings.Join(parts[2:], " ")

    // Cek apakah nomor sudah ada topic aktif
    topic, err := db.GetTopicByTelegramID(int64(c.Chat().ID))
    if err != nil {
        // buat topic baru
        contactName := nomor // default pakai nomor dulu
        topic = &db.Topic{
            WaNumber: nomor,
            ContactName: contactName,
            TelegramTopicID: c.Chat().ID,
        }
        // TODO: Simpan ke DB
        // Untuk sekarang kita simpan di memory saja (Nanti disambungkan ke DB)
        activeTopics[c.Chat().ID] = topic
    }

    // TODO: Kirim pesan ke WhatsApp (via WhatsMeow nanti)

    // Kirim pesan ke Telegram topicGroup dengan footer inisial
    initial := users[senderId]
    footer := ""
    if initial != "" {
        footer = fmt.Sprintf("\n\n-%s", initial)
    }

    msg := pesan + footer
    _, err = bot.Send(c.Chat(), msg)
    return err
}

// !close
func cmdClose(c tele.Context, senderId int64) error {
    // Hanya boleh di chat topicGroup
    if c.Chat().ID != topicGroup {
        return c.Reply("Perintah !close hanya dapat digunakan di group topic")
    }

    topic, err := db.GetTopicByTelegramID(topicGroup)
    if err != nil {
        return c.Reply("Topic tidak ditemukan")
    }

    // TODO: Hapus topic di DB dan hapus dari activeTopics
    delete(activeTopics, topicGroup)
    // Kirim info ke group
    _, err = bot.Send(c.Chat(), "Topic telah ditutup dan dihapus dari database.")
    return err
}

func handleReplyMessage(c tele.Context, senderId int64) error {
    reply := c.Message().ReplyTo
    if reply == nil {
        return nil
    }

    // Kirim pesan ke WhatsApp sesuai topic terkait
    // TODO: cari topic dari chat ID
    topic, ok := activeTopics[c.Chat().ID]
    if !ok {
        return c.Reply("Topic tidak ditemukan, silakan mulai chat dengan !chat nomor pesan")
    }

    initial := users[senderId]
    footer := ""
    if initial != "" {
        footer = fmt.Sprintf("\n\n-%s", initial)
    }

    msg := c.Text() + footer

    // TODO: Kirim ke WhatsApp via WhatsMeow

    _, err := bot.Send(c.Chat(), "Pesan diteruskan ke WhatsApp: "+msg)
    return err
}

func isSuperadmin(id int64) bool {
    _, ok := superadmins[id]
    return ok
}
