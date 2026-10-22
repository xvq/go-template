package resource

import (
	"github.com/xvq/go-template/internal/model"
	"time"
)

type LoginLog struct {
	ID        uint64    `json:"id"`
	AdminID   *uint64   `json:"admin_id"`
	Username  string    `json:"username"`
	Success   bool      `json:"success"`
	Reason    string    `json:"reason"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
}

func LoginLogItem(a model.AdminLoginLog) LoginLog {
	return LoginLog{ID: a.ID, AdminID: a.AdminID, Username: a.Username, Success: a.Success, Reason: a.Reason, IP: a.IP, UserAgent: a.UserAgent, CreatedAt: a.CreatedAt}
}
