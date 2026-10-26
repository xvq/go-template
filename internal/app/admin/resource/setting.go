package resource

import (
	"encoding/json"
	base "github.com/xvq/go-template/internal/core/resource"
	"github.com/xvq/go-template/internal/service/settings"
)

type setting struct {
	Key     string          `json:"key"`
	Type    string          `json:"type"`
	Value   json.RawMessage `json:"value"`
	Default json.RawMessage `json:"default"`
}

var SettingResource = base.New(settingResource{}.transform)

type settingResource struct{}

func (settingResource) transform(entry settings.Entry) setting {
	return setting{
		Key: entry.Key, Type: entry.Type,
		Value:   append(json.RawMessage(nil), entry.Value...),
		Default: append(json.RawMessage(nil), entry.Default...),
	}
}
