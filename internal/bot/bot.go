package bot

import (
    "fmt"
    "log"
    "os"
    "strconv"
    "strings"
    "sync"

    "wa-bridge/internal/db"

    tele "gopkg.in/tucnak/telebot.v2"
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

    // Cek apakah sudah ada topic berdasarkan WA number
    topic, err := db.GetTopic(nomor)
    if err != nil {
        return c.Reply("❌ Gagal cek DB.")
    }

    var topicID int64
    var contactName string

    if topic == nil {
        // Belum ada: Buat topic baru
        contactName = nomor
        topicID, err = CreateTopic(contactName)
        if err != nil {
            return c.Reply("❌ Gagal buat topic.")
        }

        // Simpan ke DB
        err = db.SaveTopic(nomor, contactName, topicID)
        if err != nil {
            return c.Reply("❌ Gagal simpan ke DB.")
        }
    } else {
        topicID = topic.TelegramTopicID
        contactName = topic.ContactName
    }

    // Kirim ke WhatsApp
    err = SendToWhatsApp(nomor, pesan)
    if err != nil {
        return c.Reply("❌ Gagal kirim ke WhatsApp.")
    }

    // Footer inisial pengirim
    initial := users[senderId]
    footer := ""
    if initial != "" {
        footer = fmt.Sprintf("\n\n-%s", initial)
    }

    // Kirim ke Telegram topic
    finalMsg := fmt.Sprintf("📤 *Ke:* %s\n📱 *No:* %s\n\n%s%s", contactName, nomor, pesan, footer)
    SendToTopic(finalMsg, topicID)

    // Juga kirim ke full forwarder
    fullText := fmt.Sprintf("📤 %s (%s): %s%s", contactName, nomor, pesan, footer)
    SendToFullGroup(fullText)

    return nil
}

// !close
func cmdClose(c tele.Context) error {
    chatID := c.Chat().ID

    // Cek topik berdasarkan TelegramTopicID
    topic, err := db.GetTopicByTelegramID(chatID)
    if err != nil || topic == nil {
        return c.Reply("❌ Topik tidak ditemukan atau sudah ditutup.")
    }

    // Hapus dari database
    err = db.DeleteTopicByTelegramID(chatID)
    if err != nil {
        return c.Reply("❌ Gagal menghapus topik dari database.")
    }

    // Hapus dari memory cache jika ada
    delete(activeTopics, chatID)

    // (Opsional) Hapus topic Telegram (jika pakai forum)
    // bot.DeleteForumTopic(c.Chat(), chatID) // Jika pakai metode forum topic

    return c.Reply(fmt.Sprintf("✅ Topik untuk *%s* (%s) telah ditutup.", topic.ContactName, topic.WaNumber), &tele.SendOptions{
        ParseMode: tele.ModeMarkdown,
    })
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
