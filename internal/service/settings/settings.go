package settings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/xvq/go-template/internal/apperror"
	"github.com/xvq/go-template/internal/core"
	"github.com/xvq/go-template/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SettingDefinition struct {
	Key       string
	Type      string
	Default   json.RawMessage
	Sensitive bool
	Validate  func(json.RawMessage) error
}

// Definitions are code-owned. Add business settings here, not through arbitrary
// API keys. Do not put database/Redis credentials into settings.
func SettingDefinitions() []SettingDefinition {
	return []SettingDefinition{
		{Key: "site.name", Type: "string", Default: json.RawMessage(`"Go Template"`), Validate: func(raw json.RawMessage) error {
			var v string
			if json.Unmarshal(raw, &v) != nil || strings.TrimSpace(v) == "" || len([]rune(v)) > 100 {
				return apperror.ErrInvalidSetting
			}
			return nil
		}},
		{Key: "site.maintenance", Type: "boolean", Default: json.RawMessage(`false`), Validate: func(raw json.RawMessage) error {
			var v *bool
			if json.Unmarshal(raw, &v) != nil || v == nil {
				return apperror.ErrInvalidSetting
			}
			return nil
		}},
	}
}

func cacheKey() string { return core.Config.Redis.Prefix + ":settings" }

// Cache fill is guarded by a generation value. A reader that began before a
// successful invalidation cannot repopulate the cache with stale database data.
var fillSettings = redis.NewScript(`
local version = redis.call('GET', KEYS[2]) or '0'
if version == ARGV[1] then
 redis.call('SET', KEYS[1], ARGV[2], 'PX', ARGV[3])
 return 1
end
return 0
`)
var invalidateSettings = redis.NewScript(`redis.call('INCR', KEYS[2]); redis.call('DEL', KEYS[1]); return 1`)

func all(ctx context.Context) (map[string]json.RawMessage, error) {
	defs := SettingDefinitions()
	version := ""
	cacheOK := core.Redis != nil
	if cacheOK {
		raw, err := core.Redis.Get(ctx, cacheKey()).Bytes()
		if err == nil {
			var cached map[string]json.RawMessage
			if json.Unmarshal(raw, &cached) == nil && validCache(cached, defs) {
				return cached, nil
			}
			core.Log.Warn("invalid settings cache; reloading")
		}
		if err != nil && !errors.Is(err, redis.Nil) {
			cacheOK = false
			core.Log.Warn("settings cache unavailable; using database")
		}
		if cacheOK {
			v, e := core.Redis.Get(ctx, cacheKey()+":version").Result()
			if errors.Is(e, redis.Nil) {
				version = "0"
			} else if e == nil {
				version = v
			} else {
				cacheOK = false
			}
		}
	}
	var rows []model.Setting
	if err := core.DB.WithContext(ctx).Find(&rows).Error; err != nil {
		return nil, err
	}
	values := map[string]json.RawMessage{}
	for _, d := range defs {
		values[d.Key] = append(json.RawMessage(nil), d.Default...)
	}
	for _, row := range rows {
		for _, d := range defs {
			if d.Key == row.Key {
				raw := json.RawMessage(row.Value)
				if err := d.Validate(raw); err != nil {
					return nil, fmt.Errorf("invalid stored setting %s", row.Key)
				}
				values[row.Key] = raw
				break
			}
		}
	}
	if cacheOK {
		raw, err := json.Marshal(values)
		if err != nil {
			return nil, err
		}
		if err = fillSettings.Run(ctx, core.Redis, []string{cacheKey(), cacheKey() + ":version"}, version, string(raw), core.Config.Settings.CacheTTL.Milliseconds()).Err(); err != nil {
			core.Log.Warn("settings cache fill failed")
		}
	}
	return values, nil
}
func validCache(values map[string]json.RawMessage, defs []SettingDefinition) bool {
	if len(values) != len(defs) {
		return false
	}
	for _, d := range defs {
		raw, ok := values[d.Key]
		if !ok || d.Validate(raw) != nil {
			return false
		}
	}
	return true
}
func String(ctx context.Context, key string) (string, error) {
	m, e := all(ctx)
	if e != nil {
		return "", e
	}
	v, ok := m[key]
	if !ok {
		return "", apperror.ErrInvalidSetting
	}
	var out string
	if e = json.Unmarshal(v, &out); e != nil {
		return "", fmt.Errorf("setting %s is not a string", key)
	}
	return out, nil
}
func Bool(ctx context.Context, key string) (bool, error) {
	m, e := all(ctx)
	if e != nil {
		return false, e
	}
	v, ok := m[key]
	if !ok {
		return false, apperror.ErrInvalidSetting
	}
	var out bool
	if e = json.Unmarshal(v, &out); e != nil {
		return false, fmt.Errorf("setting %s is not a boolean", key)
	}
	return out, nil
}

// Entry is a safe business-level setting description, without HTTP formatting.
// Sensitive values are excluded here as well as in the resource layer.
type Entry struct {
	Key       string
	Type      string
	Value     json.RawMessage
	Sensitive bool
}

func List(ctx context.Context) ([]Entry, error) {
	values, err := all(ctx)
	if err != nil {
		return nil, err
	}
	return entries(values, SettingDefinitions()), nil
}

func entries(values map[string]json.RawMessage, defs []SettingDefinition) []Entry {
	out := make([]Entry, 0, len(defs))
	for _, d := range defs {
		entry := Entry{Key: d.Key, Type: d.Type, Sensitive: d.Sensitive}
		if !d.Sensitive {
			entry.Value = values[d.Key]
		}
		out = append(out, entry)
	}
	return out
}

// Set validates and updates one setting, using the same invalidation as Update.
func Set(ctx context.Context, key string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return Update(ctx, map[string]json.RawMessage{key: raw})
}

func Update(ctx context.Context, values map[string]json.RawMessage) error {
	if len(values) == 0 {
		return apperror.ErrInvalidSetting
	}
	defs := SettingDefinitions()
	rows := make([]model.Setting, 0, len(values))
	for key, raw := range values {
		found := false
		for _, d := range defs {
			if d.Key == key {
				found = true
				if d.Validate(raw) != nil {
					return fmt.Errorf("%w: %s", apperror.ErrInvalidSetting, key)
				}
				break
			}
		}
		if !found {
			return fmt.Errorf("%w: %s", apperror.ErrInvalidSetting, key)
		}
		rows = append(rows, model.Setting{Key: key, Value: string(raw)})
	}
	if err := core.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "key"}}, DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"})}).Create(&rows).Error
	}); err != nil {
		return err
	}
	if core.Redis != nil {
		// Invalidation still gets a bounded attempt when the HTTP request is canceled.
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer cancel()
		var err error
		for i := 0; i < 3; i++ {
			err = invalidateSettings.Run(cleanup, core.Redis, []string{cacheKey(), cacheKey() + ":version"}).Err()
			if err == nil {
				break
			}
		}
		if err != nil {
			core.Log.Error("settings committed but cache invalidation failed; bounded by cache TTL", "error", err)
		}
	}
	return nil
}
