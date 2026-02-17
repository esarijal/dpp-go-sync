package main

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("❌ Gagal load .env file:", err)
	}

	totalStart := time.Now()
	log.Println("🔄 Sync Firestore → PostgreSQL...")

	// Check last sync time for incremental sync
	lastSync, err := GetLastSyncTime()
	if err != nil {
		log.Fatal("🔥 Failed to get last sync time:", err)
	}

	// Support --full flag to force full sync
	forceFullSync := false
	for _, arg := range os.Args[1:] {
		if arg == "--full" {
			forceFullSync = true
			break
		}
	}

	since := lastSync
	if forceFullSync {
		since = time.Time{} // zero time = full sync
		log.Println("⚠️  Force full sync (--full flag)")
	}

	syncStart := time.Now()

	log.Println("📥 Fetching data from Firestore...")
	fetchStart := time.Now()
	devices, config, err := FetchFromFirestore(since)
	if err != nil {
		log.Fatal("🔥 Fetch error:", err)
	}
	log.Printf("📥 Fetched %d devices + config in %s\n", len(devices), time.Since(fetchStart))

	if len(devices) == 0 {
		log.Println("✅ No updated devices found. Nothing to sync.")
		return
	}

	log.Println("📤 Saving to PostgreSQL...")
	saveStart := time.Now()
	err = SaveToPostgres(devices, config)
	if err != nil {
		log.Fatal("🔥 Save error:", err)
	}
	log.Printf("📤 Saved to PostgreSQL in %s\n", time.Since(saveStart))

	// Update last sync time
	err = UpdateLastSyncTime(syncStart)
	if err != nil {
		log.Fatal("🔥 Failed to update last sync time:", err)
	}

	log.Printf("✅ Sinkronisasi selesai! Total waktu: %s\n", time.Since(totalStart))
}
