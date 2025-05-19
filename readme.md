```
wa-bridge/
├── bot/
│   ├── telegram.js           # Handler bot Telegram (Telethon/GramJS)
│   ├── commands.js           # Logika command (!add, !chat, dsb)
│   └── userAccess.js         # Manajemen user dan inisial
├── wa/
│   └── whatsmeow.js          # WhatsApp handler (pesan masuk/keluar)
├── db/
│   └── connection.js         # Koneksi ke database MySQL
│   └── models.js             # Query: topic, user, dll
├── media/
│   ├── images/
│   └── videos/
├── .env
├── app.js                    # Entry point
└── readme.md
