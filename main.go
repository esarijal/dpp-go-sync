package main

import (
	"context"
	"log"
	"os"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("⚠️ .env file not found, using system env")
	}

	ctx := context.Background()

	// Init Firestore Client
	sa := option.WithCredentialsFile(os.Getenv("FIREBASE_CREDENTIALS"))
	client, err := firestore.NewClient(ctx, os.Getenv("FIREBASE_PROJECT_ID"), sa)
	if err != nil {
		log.Fatal("❌ Failed to create Firestore client:", err)
	}
	defer client.Close()

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

	// 1. Get Config
	config, err := GetDzikirConfig(ctx, client)
	if err != nil {
		log.Fatal("🔥 Failed to fetch config:", err)
	}

	// 2. Get Iterator
	iter := GetDeviceIterator(ctx, client, since)
	defer iter.Stop()

	// 3. Process Stream
	log.Println("📥 Streaming data from Firestore...")

	const batchSize = 500
	var batch []FCMDevice
	totalProcessed := 0

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatal("🔥 Error iterating Firestore:", err)
		}

		batch = append(batch, ParseDevice(doc))

		if len(batch) >= batchSize {
			if err := SaveToPostgres(batch, config); err != nil {
				log.Fatal("🔥 Save error:", err)
			}
			totalProcessed += len(batch)
			batch = nil // clear buffer
			log.Printf("🔹 Processed %d devices...", totalProcessed)
		}
	}

	// Flush remaining
	if len(batch) > 0 {
		if err := SaveToPostgres(batch, config); err != nil {
			log.Fatal("🔥 Save error:", err)
		}
		totalProcessed += len(batch)
	}

	log.Printf("✅ Synced %d devices in %s\n", totalProcessed, time.Since(totalStart))

	if totalProcessed == 0 {
		log.Println("💤 No updates found.")
	}

	// Update last sync time
	// Note: We only update IF successful. If it crashed mid-way, we retry everything next time.
	// This ensures consistency but might be expensive if crashes are frequent.
	// User asked for "cheap/simple", so we stick to this for now.
	err = UpdateLastSyncTime(syncStart)
	if err != nil {
		log.Fatal("🔥 Failed to update last sync time:", err)
	}

	log.Printf("🎉 Finished! Total time: %s\n", time.Since(totalStart))
}
