package resource

import (
	"github.com/xvq/go-template/internal/model"
	"time"
)

type Admin struct {
	ID          uint64    `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AdminItem is the only field-selection function; collections/pages reuse it.
func AdminItem(a model.Admin) Admin {
	return Admin{ID: a.ID, Username: a.Username, DisplayName: a.DisplayName, Enabled: a.Enabled, CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt}
}
