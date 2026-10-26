package resource

import (
	"encoding/json"
	base "github.com/xvq/go-template/internal/core/resource"
	"github.com/xvq/go-template/internal/service/settings"
)

type setting struct {
	Key       string          `json:"key"`
	Type      string          `json:"type"`
	Value     json.RawMessage `json:"value,omitempty"`
	Sensitive bool            `json:"sensitive"`
}

// SettingResource maps loaded settings entries without querying their values.
var SettingResource = base.New(settingResource{}.transform)

type settingResource struct{}

func (settingResource) transform(entry settings.Entry) setting {
	out := setting{Key: entry.Key, Type: entry.Type, Sensitive: entry.Sensitive}
	if !entry.Sensitive {
		out.Value = append(json.RawMessage(nil), entry.Value...)
	}
	return out
}
