package resource

import (
	base "github.com/xvq/go-template/internal/core/resource"
	"github.com/xvq/go-template/internal/model"
	"time"
)

type admin struct {
	ID          uint64    `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AdminResource maps rows from admins. Treat this shared resource as read-only.
var AdminResource = base.New(adminResource{}.transform)

type adminResource struct{}

func (adminResource) transform(a model.Admin) admin {
	return admin{ID: a.ID, Username: a.Username, DisplayName: a.DisplayName, Enabled: a.Enabled, CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt}
}
