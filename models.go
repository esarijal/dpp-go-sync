package main

import "time"

type FCMDevice struct {
	DeviceID          string
	FCMToken          string
	TZ                string
	DzikirMorningTime string
	DzikirEveningTime string
	LastSentMorning   *time.Time
	LastSentEvening   *time.Time
	Shard             int
	IsActive          bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type DzikirConfig struct {
	MorningIndex int
	EveningIndex int
}
