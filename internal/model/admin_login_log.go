package model

import "time"

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
