package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

const chunkSize = 1000

func SaveToPostgres(devices []FCMDevice, config DzikirConfig) error {
	total := len(devices)
	log.Printf("🔄 Saving to PostgreSQL for %d devices (in chunks of %d)\n", total, chunkSize)

	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		return fmt.Errorf("failed to open DB connection: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping DB: %w", err)
	}

	const stmtStr = `
		INSERT INTO devices (
			device_id, fcm_token, tz,
			dzikir_morning_time, dzikir_evening_time,
			last_sent_morning, last_sent_evening,
			shard, created_at, updated_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10
		)
		ON CONFLICT (device_id) DO UPDATE SET
			fcm_token = EXCLUDED.fcm_token,
			tz = EXCLUDED.tz,
			dzikir_morning_time = EXCLUDED.dzikir_morning_time,
			dzikir_evening_time = EXCLUDED.dzikir_evening_time,
			last_sent_morning = EXCLUDED.last_sent_morning,
			last_sent_evening = EXCLUDED.last_sent_evening,
			shard = EXCLUDED.shard,
			updated_at = EXCLUDED.updated_at;
	`

	chunkCount := (total + chunkSize - 1) / chunkSize
	for i := 0; i < total; i += chunkSize {
		end := i + chunkSize
		if end > total {
			end = total
		}
		chunk := devices[i:end]

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction (chunk %d): %w", i/chunkSize+1, err)
		}

		stmt, err := tx.Prepare(stmtStr)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to prepare statement: %w", err)
		}

		for _, d := range chunk {
			_, err := stmt.Exec(
				d.DeviceID, d.FCMToken, d.TZ,
				d.DzikirMorningTime, d.DzikirEveningTime,
				d.LastSentMorning, d.LastSentEvening,
				d.Shard, d.CreatedAt, d.UpdatedAt,
			)
			if err != nil {
				stmt.Close()
				tx.Rollback()
				return fmt.Errorf("failed to insert/update devices (chunk %d): %w", i/chunkSize+1, err)
			}
		}

		stmt.Close()

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction (chunk %d): %w", i/chunkSize+1, err)
		}

		log.Printf("✅ Committed chunk %d of %d — saved %d of %d devices",
			i/chunkSize+1, chunkCount, end, total)
	}

	// Upsert dzikir_config
	_, err = db.Exec(`
		INSERT INTO dzikir_config (id, morning_index, evening_index, updated_at)
		VALUES ('general', $1, $2, NOW())
		ON CONFLICT (id) DO UPDATE SET
			morning_index = EXCLUDED.morning_index,
			evening_index = EXCLUDED.evening_index,
			updated_at = NOW()
	`, config.MorningIndex, config.EveningIndex)
	if err != nil {
		return fmt.Errorf("failed to insert/update dzikir_config: %w", err)
	}

	log.Println("🎉 All chunks committed. Save to PostgreSQL completed successfully.")
	return nil
}

func openDB() (*sql.DB, error) {
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		return nil, fmt.Errorf("failed to open DB connection: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping DB: %w", err)
	}
	return db, nil
}

func GetLastSyncTime() (time.Time, error) {
	db, err := openDB()
	if err != nil {
		return time.Time{}, err
	}
	defer db.Close()

	// Create table if not exists
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sync_meta (
			key TEXT PRIMARY KEY,
			value TIMESTAMPTZ NOT NULL
		)
	`)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to create sync_meta table: %w", err)
	}

	var lastSync time.Time
	err = db.QueryRow(`SELECT value FROM sync_meta WHERE key = 'last_sync'`).Scan(&lastSync)
	if err == sql.ErrNoRows {
		return time.Time{}, nil // First run, return zero time
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get last sync time: %w", err)
	}
	return lastSync, nil
}

func UpdateLastSyncTime(t time.Time) error {
	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(`
		INSERT INTO sync_meta (key, value) VALUES ('last_sync', $1)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
	`, t)
	if err != nil {
		return fmt.Errorf("failed to update last sync time: %w", err)
	}
	return nil
}
