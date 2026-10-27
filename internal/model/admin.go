package model

import "time"

type Admin struct {
	ID           uint64    `gorm:"primaryKey" json:"-"`
	Username     string    `gorm:"size:64;uniqueIndex" json:"-"`
	DisplayName  string    `gorm:"size:100" json:"-"`
	PasswordHash string    `gorm:"size:255" json:"-"`
	Enabled      bool      `json:"-"`
	CreatedAt    time.Time `json:"-"`
	UpdatedAt    time.Time `json:"-"`
}
