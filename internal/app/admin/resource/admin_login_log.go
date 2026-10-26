package resource

import (
	base "github.com/xvq/go-template/internal/core/resource"
	"github.com/xvq/go-template/internal/model"
	"time"
)

type adminLoginLog struct {
	ID        uint64    `json:"id"`
	AdminID   *uint64   `json:"admin_id"`
	Username  string    `json:"username"`
	Success   bool      `json:"success"`
	Reason    string    `json:"reason"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
}

// AdminLoginLogResource maps rows from admin_login_logs.
var AdminLoginLogResource = base.New(adminLoginLogResource{}.transform)

type adminLoginLogResource struct{}

func (adminLoginLogResource) transform(a model.AdminLoginLog) adminLoginLog {
	return adminLoginLog{ID: a.ID, AdminID: a.AdminID, Username: a.Username, Success: a.Success, Reason: a.Reason, IP: a.IP, UserAgent: a.UserAgent, CreatedAt: a.CreatedAt}
}
