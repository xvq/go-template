package resource

import (
	"encoding/json"
	"github.com/xvq/go-template/internal/service/settings"
)

type Setting struct {
	Key       string          `json:"key"`
	Type      string          `json:"type"`
	Value     json.RawMessage `json:"value,omitempty"`
	Sensitive bool            `json:"sensitive"`
}

func SettingItem(entry settings.Entry) Setting {
	out := Setting{Key: entry.Key, Type: entry.Type, Sensitive: entry.Sensitive}
	if !entry.Sensitive {
		out.Value = append(json.RawMessage(nil), entry.Value...)
	}
	return out
}
