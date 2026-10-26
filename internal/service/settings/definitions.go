package settings

import (
	"encoding/json"
	"github.com/xvq/go-template/internal/apperror"
	"strings"
)

type SettingDefinition struct {
	Key      string
	Type     string
	Default  json.RawMessage
	Validate func(json.RawMessage) error
}

// Definitions are code-owned; arbitrary undeclared keys cannot be saved.
func SettingDefinitions() []SettingDefinition {
	return []SettingDefinition{
		{Key: "site.name", Type: "string", Default: json.RawMessage(`"Go Template"`), Validate: func(raw json.RawMessage) error {
			var value string
			if json.Unmarshal(raw, &value) != nil || strings.TrimSpace(value) == "" || len([]rune(value)) > 100 {
				return apperror.ErrInvalidSetting
			}
			return nil
		}},
		{Key: "site.maintenance", Type: "boolean", Default: json.RawMessage(`false`)},
	}
}

// validate enforces the declared JSON type before any setting-specific rules.
func (d SettingDefinition) validate(raw json.RawMessage) error {
	valid := false
	switch d.Type {
	case "string":
		var value *string
		valid = json.Unmarshal(raw, &value) == nil && value != nil
	case "boolean":
		var value *bool
		valid = json.Unmarshal(raw, &value) == nil && value != nil
	case "integer":
		var value *int
		valid = json.Unmarshal(raw, &value) == nil && value != nil
	case "string_array":
		var values *[]*string
		valid = json.Unmarshal(raw, &values) == nil && values != nil
		if valid {
			for _, value := range *values {
				if value == nil {
					valid = false
					break
				}
			}
		}
	}
	if !valid {
		return apperror.ErrInvalidSetting
	}
	if d.Validate != nil {
		return d.Validate(raw)
	}
	return nil
}
