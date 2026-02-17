# DPP Go Sync

Sinkronisasi data device & konfigurasi dzikir dari **Firebase Firestore** ke **PostgreSQL**.

## Fitur

- **Full Sync** — baca semua dokumen dari Firestore (run pertama kali otomatis full)
- **Incremental Sync** — hanya baca dokumen yang berubah sejak sync terakhir (hemat biaya Firestore)
- **Upsert** — data yang sudah ada di Postgres akan di-update, yang baru akan di-insert
- **Chunked Insert** — insert dalam batch 1000 rows per transaksi

## Data yang Di-sync

| Firestore Collection | PostgreSQL Table | Keterangan |
|---|---|---|
| `fcm_devices` | `devices` | Data device FCM (token, timezone, jadwal dzikir, dll) |
| `dzikir_config/general` | `dzikir_config` | Index dzikir pagi & petang |

## Setup

### 1. Siapkan `.env`

```env
FIREBASE_CREDENTIALS=serviceAccountKey.json
FIREBASE_PROJECT_ID=your-firebase-project-id
DATABASE_URL=postgres://user:password@host:5432/dbname?sslmode=require
```

### 2. Siapkan `serviceAccountKey.json`

Download dari Firebase Console → Project Settings → Service Accounts → Generate New Private Key.

### 3. Jalankan

```bash
# Langsung run (otomatis full sync pertama kali, incremental selanjutnya)
go run .

# Paksa full sync
go run . --full
```

### Build untuk Linux

```powershell
# Dari PowerShell (Windows)
$env:GOOS="linux"; $env:GOARCH="amd64"; go build -o go-sync .
```

```bash
# Upload ke server lalu jalankan
chmod +x go-sync
./go-sync
```

## Estimasi Biaya Firestore

| Skenario | Reads/run | Biaya |
|---|---|---|
| Full sync (138K devices) | ~138K | ~Rp 850 |
| Incremental (1K berubah/hari) | ~1K | Gratis (free tier) |
