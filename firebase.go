package main

import (
	"context"
	"log"
	"time"

	"cloud.google.com/go/firestore"
)

func GetDzikirConfig(ctx context.Context, client *firestore.Client) (DzikirConfig, error) {
	configDoc, err := client.Doc("dzikir_config/general").Get(ctx)
	if err != nil {
		return DzikirConfig{}, err
	}
	return DzikirConfig{
		MorningIndex: getInt(configDoc.Data()["morningIndex"]),
		EveningIndex: getInt(configDoc.Data()["eveningIndex"]),
	}, nil
}

func GetDeviceIterator(ctx context.Context, client *firestore.Client, since time.Time) *firestore.DocumentIterator {
	if since.IsZero() {
		log.Println("📥 Mode: FULL SYNC (first run)")
		// Return iterator for all docs
		return client.Collection("fcm_devices").Documents(ctx)
	} else {
		log.Printf("📥 Mode: INCREMENTAL SYNC (since %s)", since.Format(time.RFC3339))
		// Return iterator for updated docs
		return client.Collection("fcm_devices").Where("updatedAt", ">", since).Documents(ctx)
	}
}

func ParseDevice(doc *firestore.DocumentSnapshot) FCMDevice {
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
