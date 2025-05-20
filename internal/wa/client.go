package wa

import (
    "context"
    "fmt"
    "log"

    "go.mau.fi/whatsmeow"
    "go.mau.fi/whatsmeow/store/sqlstore"
    "go.mau.fi/whatsmeow/types/events"
    "go.mau.fi/whatsmeow/types"
    _ "github.com/mattn/go-sqlite3"
    waLog "go.mau.fi/whatsmeow/util/log"

    waProto "go.mau.fi/whatsmeow/binary/proto"
    "google.golang.org/protobuf/proto"
)

var Client *whatsmeow.Client

func StartWA() {
    ctx := context.Background()

    logger := waLog.Stdout("db", "DEBUG")
    container, err := sqlstore.New(ctx, "sqlite3", "file:wa_session.db?_foreign_keys=on", logger)
    if err != nil {
        log.Fatalf("Failed to connect DB: %v", err)
    }

    deviceStore, err := container.GetFirstDevice(ctx)
    if err != nil {
        log.Fatalf("Failed to get device: %v", err)
    }

    Client = whatsmeow.NewClient(deviceStore, nil)

    Client.AddEventHandler(handleWAEvent)

    if Client.Store.ID == nil {
        qrChan, _ := Client.GetQRChannel(ctx)
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

func SendToWhatsApp(number string, msg string) error {
    jid := types.NewJID(number, types.DefaultUserServer)
    _, err := Client.SendMessage(context.Background(), jid, &waProto.Message{
        Conversation: proto.String(msg),
    })
    return err
}

func handleIncomingWA(evt *events.Message) {
    msg := evt.Message.GetConversation()
    sender := evt.Info.Sender.User

    fmt.Printf("📩 WA Message from %s: %s\n", sender, msg)

    // TODO: implement forwarding to Telegram here

    log.Printf("Pesan masuk dari %s: %s\n", sender, msg)
}
