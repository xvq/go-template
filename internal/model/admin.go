package model

import "time"

type Admin struct {
	ID           uint64    `gorm:"primaryKey" json:"-"`
	Username     string    `gorm:"size:64;uniqueIndex" json:"-"`
	DisplayName  string    `gorm:"size:100" json:"-"`
	PasswordHash string    `gorm:"size:255" json:"-"`
	Enabled      bool      `json:"-"`
	AuthVersion  uint64    `json:"-"`
	CreatedAt    time.Time `json:"-"`
	UpdatedAt    time.Time `json:"-"`
}

type AdminLoginLog struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	AdminID   *uint64   `json:"admin_id"`
	Username  string    `gorm:"size:64" json:"username"`
	Success   bool      `json:"success"`
	Reason    string    `gorm:"size:64" json:"reason"`
	IP        string    `gorm:"size:64" json:"ip"`
	UserAgent string    `gorm:"size:512" json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
}

type Setting struct {
	Key       string `gorm:"primaryKey;size:128"`
	Value     string `gorm:"type:json"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
