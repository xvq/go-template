package settings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/xvq/go-template/internal/apperror"
	"github.com/xvq/go-template/internal/core"
	"github.com/xvq/go-template/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const cacheKey = "settings"

// Cache fill is guarded by a generation value. A reader that began before a
// successful invalidation cannot repopulate the cache with stale database data.
var fillSettings = redis.NewScript(`
local version = redis.call('GET', KEYS[2]) or '0'
if version ~= ARGV[1] then return 0 end
redis.call('DEL', KEYS[1])
for i = 2, #ARGV, 2 do
 redis.call('HSET', KEYS[1], ARGV[i], ARGV[i + 1])
end
return 1
`)
var invalidateSettings = redis.NewScript(`redis.call('INCR', KEYS[2]); redis.call('DEL', KEYS[1]); return 1`)

func all(ctx context.Context) (map[string]json.RawMessage, error) {
	defs := SettingDefinitions()
	cacheOK := core.Redis != nil
	if cacheOK {
		fields, err := core.Redis.HGetAll(ctx, cacheKey).Result()
		if err == nil && len(fields) > 0 {
			cached := make(map[string]json.RawMessage, len(fields))
			for key, value := range fields {
				cached[key] = json.RawMessage(value)
			}
			if validCache(cached, defs) {
				return cached, nil
			}
			core.Log.Warn("invalid settings cache; reloading")
		}
		if err != nil {
			cacheOK = false
			core.Log.Warn("settings cache unavailable; using database")
		}
	}
	return loadSettings(ctx, defs, cacheOK)
}

// loadSettings validates a complete database snapshot and optionally fills Redis.
// Read the generation before querying so concurrent invalidation rejects stale fills.
func loadSettings(ctx context.Context, defs []SettingDefinition, cacheOK bool) (map[string]json.RawMessage, error) {
	version := ""
	if cacheOK {
		v, err := core.Redis.Get(ctx, cacheKey+":version").Result()
		if errors.Is(err, redis.Nil) {
			version = "0"
		} else if err == nil {
			version = v
		} else {
			cacheOK = false
		}
	}
	var rows []model.Setting
	if err := core.DB.WithContext(ctx).Find(&rows).Error; err != nil {
		return nil, err
	}
	values := map[string]json.RawMessage{}
	for _, d := range defs {
		if d.validate(d.Default) != nil {
			return nil, fmt.Errorf("invalid default setting %s", d.Key)
		}
		values[d.Key] = append(json.RawMessage(nil), d.Default...)
	}
	for _, row := range rows {
		for _, d := range defs {
			if d.Key == row.Key {
				raw := json.RawMessage(row.Value)
				if err := d.validate(raw); err != nil {
					return nil, fmt.Errorf("invalid stored setting %s", row.Key)
				}
				values[row.Key] = raw
				break
			}
		}
	}
	if cacheOK {
		args := []any{version}
		for key, value := range values {
			args = append(args, key, string(value))
		}
		if err := fillSettings.Run(ctx, core.Redis, []string{cacheKey, cacheKey + ":version"}, args...).Err(); err != nil {
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
		if !ok || d.validateType(raw) != nil {
			return false
		}
	}
	return true
}
func String(ctx context.Context, key string) (string, error) { return read[string](ctx, key, "string") }
func Bool(ctx context.Context, key string) (bool, error)     { return read[bool](ctx, key, "boolean") }
func Int(ctx context.Context, key string) (int, error)       { return read[int](ctx, key, "integer") }
func Strings(ctx context.Context, key string) ([]string, error) {
	return read[[]string](ctx, key, "string_array")
}

func read[T any](ctx context.Context, key, expected string) (T, error) {
	var out T
	defs := SettingDefinitions()
	var definition SettingDefinition
	for _, d := range defs {
		if d.Key == key {
			definition = d
			break
		}
	}
	if definition.Key == "" || definition.Type != expected {
		return out, fmt.Errorf("%w: setting %s is not %s", apperror.ErrInvalidSetting, key, expected)
	}
	cacheOK := core.Redis != nil
	if cacheOK {
		value, err := core.Redis.HGet(ctx, cacheKey, key).Result()
		switch {
		case err == nil:
			raw := json.RawMessage(value)
			if definition.validateType(raw) == nil {
				if err := json.Unmarshal(raw, &out); err != nil {
					return out, err
				}
				return out, nil
			}
			core.Log.Warn("invalid settings cache; reloading")
		case errors.Is(err, redis.Nil):
			// A missing field rebuilds the complete snapshot.
		default:
			cacheOK = false
			core.Log.Warn("settings cache unavailable; using database")
		}
	}
	values, err := loadSettings(ctx, defs, cacheOK)
	if err != nil {
		return out, err
	}
	value, ok := values[key]
	if !ok {
		return out, apperror.ErrInvalidSetting
	}
	if err := json.Unmarshal(value, &out); err != nil {
		return out, fmt.Errorf("%w: setting %s is not %s", apperror.ErrInvalidSetting, key, expected)
	}
	return out, nil
}

// Entry describes a setting's effective value and code-defined default.
type Entry struct {
	Key     string
	Type    string
	Value   json.RawMessage
	Default json.RawMessage
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
		entry := Entry{
			Key: d.Key, Type: d.Type,
			Value:   append(json.RawMessage(nil), values[d.Key]...),
			Default: append(json.RawMessage(nil), d.Default...),
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
				if d.validate(raw) != nil {
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
		// Commit succeeded: cache invalidation must still run if the request was canceled.
		if err := ClearCache(context.WithoutCancel(ctx)); err != nil {
			return fmt.Errorf("settings committed but cache invalidation failed: %w", err)
		}
	}
	return nil
}

// ClearCache invalidates the shared snapshot and rejects stale in-flight fills.
// Requires Redis. The caller initializes resources; no database access is needed.
func ClearCache(ctx context.Context) error {
	cleanup, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		err = invalidateSettings.Run(cleanup, core.Redis, []string{cacheKey, cacheKey + ":version"}).Err()
		if err == nil {
			return nil
		}
	}
	return fmt.Errorf("clear settings cache: %w", err)
}
