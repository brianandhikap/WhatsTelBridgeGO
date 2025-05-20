package wa

import (
    "context"
    "fmt"
    "log"
    "os"

    "go.mau.fi/whatsmeow"
    "go.mau.fi/whatsmeow/store/sqlstore"
    "go.mau.fi/whatsmeow/types/events"
    "go.mau.fi/whatsmeow/types"
    _ "github.com/mattn/go-sqlite3"
    "go.mau.fi/whatsmeow/store"
    "time"
)

var Client *whatsmeow.Client

func StartWA() {
    container, err := sqlstore.New("sqlite3", "file:wa_session.db?_foreign_keys=on", nil)
    if err != nil {
        log.Fatalf("Failed to connect DB: %v", err)
    }

    deviceStore, err := container.GetFirstDevice()
    if err != nil {
        log.Fatalf("Failed to get device: %v", err)
    }

    Client = whatsmeow.NewClient(deviceStore, nil)

    Client.AddEventHandler(handleWAEvent)

    if Client.Store.ID == nil {
        qrChan, _ := Client.GetQRChannel(context.Background())
        err = Client.Connect()
        if err != nil {
            log.Fatalf("Failed to connect: %v", err)
        }

        for evt := range qrChan {
            if evt.Event == "code" {
                fmt.Println("QR Code:", evt.Code)
            } else {
                fmt.Println("Login Event:", evt.Event)
            }
        }
    } else {
        err = Client.Connect()
        if err != nil {
            log.Fatalf("Failed to reconnect: %v", err)
        }
    }

    fmt.Println("WhatsApp connected!")
}

func handleWAEvent(evt interface{}) {
    switch v := evt.(type) {
    case *events.Message:
        handleIncomingWA(v)
    }
}

func handleIncomingWA(evt *events.Message) {
    msg := evt.Message.GetConversation()
    sender := evt.Info.Sender.User

    fmt.Printf("📩 WA Message from %s: %s\n", sender, msg)

    // TODO:
    // 1. Cari topik Telegram untuk sender ini
    // 2. Jika belum ada, buat topik baru dan simpan ke DB
    // 3. Kirim pesan ke topicGroup dan fullGroup
    // 4. Kirim sebagai reply di topic

    // Contoh kirim ke console
    log.Printf("Pesan masuk dari %s: %s\n", sender, msg)
}
