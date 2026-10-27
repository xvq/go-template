package model

import "time"

type Setting struct {
	Key       string `gorm:"primaryKey;size:128"`
	Value     string `gorm:"type:json"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
