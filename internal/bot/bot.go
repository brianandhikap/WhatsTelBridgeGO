// file bot.go
package bot

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"wa-bridge/internal/db"
	"wa-bridge/internal/wa"

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

	// Register message handler
	bot.Handle(tele.OnText, handleText)
	log.Println("Telegram bot started")
	bot.Start()
}

func handleText(m *tele.Message) error {
	sender := m.Sender
	senderId := sender.ID

	text := m.Text
	lowerText := strings.ToLower(text)

	// Only process commands starting with '!'
	if !strings.HasPrefix(lowerText, "!") {
		// If message is reply in a topic group, treat as message reply
		if m.Chat.ID == topicGroup && m.ReplyTo != nil {
			return handleReplyMessage(m, senderId)
		}
		return nil
	}

	// Parse command
	parts := strings.Fields(text)
	cmd := strings.ToLower(parts[0])

	switch cmd {
	case "!add":
		return cmdAdd(m, senderId, parts)
	case "!rm":
		return cmdRemove(m, senderId, parts)
	case "!chat":
		return cmdChat(m, senderId, parts)
	case "!close":
		return cmdClose(m)
	default:
		return bot.Reply(m, "Perintah tidak dikenal")
	}
}

// !add <telegram_id> <initial>
func cmdAdd(m *tele.Message, senderId int64, parts []string) error {
	if !isSuperadmin(senderId) {
		return bot.Reply(m, "Anda tidak memiliki izin.")
	}
	if len(parts) < 3 {
		return bot.Reply(m, "Format: !add <telegram_id> <initial>")
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return bot.Reply(m, "ID Telegram tidak valid")
	}
	initial := parts[2]
	err = db.AddUser(id, initial)
	if err != nil {
		return bot.Reply(m, "Gagal menambahkan user: "+err.Error())
	}
	users[id] = initial
	return bot.Reply(m, fmt.Sprintf("User %d dengan inisial %s berhasil ditambahkan", id, initial))
}

// !rm <telegram_id>
func cmdRemove(m *tele.Message, senderId int64, parts []string) error {
	if !isSuperadmin(senderId) {
		return bot.Reply(m, "Anda tidak memiliki izin.")
	}
	if len(parts) < 2 {
		return bot.Reply(m, "Format: !rm <telegram_id>")
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return bot.Reply(m, "ID Telegram tidak valid")
	}
	err = db.RemoveUser(id)
	if err != nil {
		return bot.Reply(m, "Gagal menghapus user: "+err.Error())
	}
	delete(users, id)
	return bot.Reply(m, fmt.Sprintf("User %d berhasil dihapus", id))
}

// !chat <nomor> <pesan...>
func cmdChat(m *tele.Message, senderId int64, parts []string) error {
	if len(parts) < 3 {
		return bot.Reply(m, "Format: !chat <nomor> <pesan>")
	}

	nomor := parts[1]
	pesan := strings.Join(parts[2:], " ")

	// Cek apakah sudah ada topic berdasarkan WA number
	topic, err := db.GetTopic(nomor)
	if err != nil {
		return bot.Reply(m, "❌ Gagal cek DB.")
	}

	var topicID int64
	var contactName string

	if topic == nil {
		// Belum ada: Buat topic baru
		contactName = nomor
		topicID, err = CreateTopic(contactName)
		if err != nil {
			return bot.Reply(m, "❌ Gagal buat topic.")
		}

		// Simpan ke DB
		err = db.SaveTopic(nomor, contactName, topicID)
		if err != nil {
			return bot.Reply(m, "❌ Gagal simpan ke DB.")
		}
	} else {
		topicID = topic.TelegramTopicID
		contactName = topic.ContactName
	}

	// Kirim ke WhatsApp
	err = wa.SendToWhatsApp(nomor, pesan)
	if err != nil {
		return bot.Reply(m, "❌ Gagal kirim ke WhatsApp.")
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
func cmdClose(m *tele.Message) error {
	chatID := m.Chat.ID

	// Cek topik berdasarkan TelegramTopicID
	topic, err := db.GetTopicByTelegramID(chatID)
	if err != nil || topic == nil {
		return bot.Reply(m, "❌ Topik tidak ditemukan atau sudah ditutup.")
	}

	// Hapus dari database
	err = db.DeleteTopicByTelegramID(chatID)
	if err != nil {
		return bot.Reply(m, "❌ Gagal menghapus topik dari database.")
	}

	// Hapus dari memory cache jika ada
	delete(activeTopics, chatID)

	// (Opsional) Hapus topic Telegram (jika pakai forum)
	// bot.DeleteForumTopic(c.Chat(), chatID) // Jika pakai metode forum topic

	return bot.Reply(m, fmt.Sprintf("✅ Topik untuk *%s* (%s) telah ditutup.", topic.ContactName, topic.WaNumber), &tele.SendOptions{
		ParseMode: tele.ModeMarkdown,
	})
}

func handleReplyMessage(m *tele.Message, senderId int64) error {
	reply := m.ReplyTo
	if reply == nil {
		return nil
	}

	// Kirim pesan ke WhatsApp sesuai topic terkait
	// TODO: cari topic dari chat ID
	topic, ok := activeTopics[m.Chat.ID]
	if !ok {
		return bot.Reply(m, "Topic tidak ditemukan, silakan mulai chat dengan !chat nomor pesan")
	}

	initial := users[senderId]
	footer := ""
	if initial != "" {
		footer = fmt.Sprintf("\n\n-%s", initial)
	}

	msg := m.Text + footer

	// Kirim ke WhatsApp via WhatsMeow
	err := wa.SendToWhatsApp(topic.WaNumber, msg)
	if err != nil {
		return bot.Reply(m, "❌ Gagal mengirim pesan ke WhatsApp: "+err.Error())
	}

	_, err = bot.Send(m.Chat, "Pesan diteruskan ke WhatsApp: "+msg)
	return err
}

func isSuperadmin(id int64) bool {
	_, ok := superadmins[id]
	return ok
}

// CreateTopic membuat topic baru di grup telegram
func CreateTopic(contactName string) (int64, error) {
	// TODO: Implementasi pembuatan topic
	// Contoh implementasi sederhana (jika menggunakan channel biasa):
	message := fmt.Sprintf("💬 Topic baru untuk kontak: %s", contactName)
	
	// Kirim pesan ke grup topic
	chat, err := bot.ChatByID(strconv.FormatInt(topicGroup, 10))
	if err != nil {
		return 0, err
	}
	
	m, err := bot.Send(chat, message)
	if err != nil {
		return 0, err
	}
	
	return int64(m.ID), nil
}

// SendToTopic mengirim pesan ke topic
func SendToTopic(message string, topicID int64) error {
	chat, err := bot.ChatByID(strconv.FormatInt(topicGroup, 10))
	if err != nil {
		return err
	}
	
	_, err = bot.Send(chat, message, &tele.SendOptions{
		ParseMode: tele.ModeMarkdown,
		// Jika menggunakan forum/topic Telegram, tambahkan ini:
		// ReplyTo: &tele.Message{ID: int(topicID)}
	})
	return err
}

// SendToFullGroup mengirim pesan ke grup full
func SendToFullGroup(message string) error {
	chat, err := bot.ChatByID(strconv.FormatInt(fullGroup, 10))
	if err != nil {
		return err
	}
	
	_, err = bot.Send(chat, message, &tele.SendOptions{
		ParseMode: tele.ModeMarkdown,
	})
	return err
}
