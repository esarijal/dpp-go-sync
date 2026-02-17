package main

import (
	"context"
	"log"
	"os"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/option"
)

func FetchFromFirestore(since time.Time) ([]FCMDevice, DzikirConfig, error) {
	ctx := context.Background()

	sa := option.WithCredentialsFile(os.Getenv("FIREBASE_CREDENTIALS"))
	client, err := firestore.NewClient(ctx, os.Getenv("FIREBASE_PROJECT_ID"), sa)
	if err != nil {
		return nil, DzikirConfig{}, err
	}
	defer client.Close()

	// Fetch fcm_devices (incremental if since is set)
	devices := []FCMDevice{}

	if since.IsZero() {
		log.Println("📥 Mode: FULL SYNC (first run)")
		snap, err := client.Collection("fcm_devices").Documents(ctx).GetAll()
		if err != nil {
			return nil, DzikirConfig{}, err
		}
		for _, doc := range snap {
			devices = append(devices, parseDevice(doc))
		}
	} else {
		log.Printf("📥 Mode: INCREMENTAL SYNC (since %s)", since.Format(time.RFC3339))
		snap, err := client.Collection("fcm_devices").Where("updatedAt", ">", since).Documents(ctx).GetAll()
		if err != nil {
			return nil, DzikirConfig{}, err
		}
		for _, doc := range snap {
			devices = append(devices, parseDevice(doc))
		}
	}

	// Fetch dzikir_config/general (always 1 read)
	configDoc, err := client.Doc("dzikir_config/general").Get(ctx)
	if err != nil {
		return nil, DzikirConfig{}, err
	}
	config := DzikirConfig{
		MorningIndex: getInt(configDoc.Data()["morningIndex"]),
		EveningIndex: getInt(configDoc.Data()["eveningIndex"]),
	}

	return devices, config, nil
}

func parseDevice(doc *firestore.DocumentSnapshot) FCMDevice {
	data := doc.Data()
	createdAt, _ := data["createdAt"].(time.Time)
	updatedAt, _ := data["updatedAt"].(time.Time)
	lastMorning, _ := data["lastSentMorning"].(time.Time)
	lastEvening, _ := data["lastSentEvening"].(time.Time)

	return FCMDevice{
		DeviceID:          doc.Ref.ID,
		FCMToken:          getString(data["fcmToken"]),
		TZ:                getString(data["tz"]),
		DzikirMorningTime: getString(data["dzikirMorningTime"]),
		DzikirEveningTime: getString(data["dzikirEveningTime"]),
		LastSentMorning:   &lastMorning,
		LastSentEvening:   &lastEvening,
		Shard:             getInt(data["shard"]),
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
	}
}

func getString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func getInt(v interface{}) int {
	if i, ok := v.(int64); ok {
		return int(i)
	}
	if f, ok := v.(float64); ok {
		return int(f)
	}
	return 0
}
