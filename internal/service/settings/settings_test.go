package settings

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/xvq/go-template/internal/apperror"
	"github.com/xvq/go-template/internal/core/config"
	"github.com/xvq/go-template/internal/core/testutil"
	"strings"
	"testing"
	"time"
)

func TestSettingsCacheUpdateAndValidation(t *testing.T) {
	db, m := mockDB(t)
	mr := miniredis.RunT(t)
	r := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer r.Close()
	c := config.Defaults()
	testutil.UseCore(t, c, db, r, testLogger())
	ctx := context.Background()
	m.ExpectQuery("SELECT .*settings").WillReturnRows(sqlmock.NewRows([]string{"key", "value"}))
	name, err := String(ctx, "site.name")
	if err != nil || name != "Go Template" {
		t.Fatalf("%s %v", name, err)
	}
	if r.Type(ctx, cacheKey).Val() != "hash" || mr.HGet(cacheKey, "site.name") != `"Go Template"` || mr.TTL(cacheKey) != 0 {
		t.Fatal("cache must be a permanent JSON-valued hash")
	}
	if _, err = String(ctx, "site.name"); err != nil {
		t.Fatal("cache miss", err)
	}
	if err = Update(ctx, map[string]json.RawMessage{"unknown": json.RawMessage(`true`)}); err == nil {
		t.Fatal("unknown key accepted")
	}
	if err = Update(ctx, map[string]json.RawMessage{"site.maintenance": json.RawMessage(`"false"`)}); err == nil {
		t.Fatal("wrong type accepted")
	}
	if err = Update(ctx, map[string]json.RawMessage{"site.maintenance": json.RawMessage(` null `)}); err == nil {
		t.Fatal("null boolean accepted")
	}
	m.ExpectBegin()
	m.ExpectExec("INSERT INTO .*settings").WillReturnResult(sqlmock.NewResult(1, 1))
	m.ExpectCommit()
	if err = Set(ctx, "site.name", "Changed"); err != nil {
		t.Fatal(err)
	}
	m.ExpectQuery("SELECT .*settings").WillReturnRows(sqlmock.NewRows([]string{"key", "value"}).AddRow("site.name", `"Changed"`))
	name, err = String(ctx, "site.name")
	if err != nil || name != "Changed" {
		t.Fatalf("stale setting: %s %v", name, err)
	}
	if err := m.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
func TestStaleCacheFillRejectedAndManualClear(t *testing.T) {
	db, m := mockDB(t)
	mr := miniredis.RunT(t)
	r := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer r.Close()
	c := config.Defaults()
	testutil.UseCore(t, c, db, r, testLogger())
	ctx := context.Background()
	if err := invalidateSettings.Run(ctx, r, []string{cacheKey, cacheKey + ":version"}).Err(); err != nil {
		t.Fatal(err)
	}
	n, err := fillSettings.Run(ctx, r, []string{cacheKey, cacheKey + ":version"}, "0", "site.name", `"stale"`).Int()
	if err != nil || n != 0 {
		t.Fatalf("stale cache fill accepted %d %v", n, err)
	}
	if mr.Exists(cacheKey) {
		t.Fatal("stale fill created hash")
	}
	if _, err := fillSettings.Run(ctx, r, []string{cacheKey, cacheKey + ":version"}, "1", "site.name", `"Fresh"`, "site.maintenance", `false`).Int(); err != nil {
		t.Fatal(err)
	}
	if err := ClearCache(ctx); err != nil {
		t.Fatal(err)
	}
	if mr.Exists(cacheKey) || r.Get(ctx, cacheKey+":version").Val() != "2" || mr.TTL(cacheKey+":version") != 0 {
		t.Fatal("clear must delete snapshot and increment permanent version")
	}
	if err := ClearCache(ctx); err != nil {
		t.Fatal(err)
	}
	if r.Get(ctx, cacheKey+":version").Val() != "3" {
		t.Fatal("empty-cache invalidation must still increment version")
	}

	if err := m.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDefinitionTypesAndDefaults(t *testing.T) {
	for _, tc := range []struct {
		kind, raw string
		valid     bool
	}{
		{"string", `"hello"`, true}, {"string", `null`, false},
		{"boolean", `false`, true}, {"boolean", `"false"`, false}, {"boolean", `null`, false},
		{"integer", `0`, true}, {"integer", `-7`, true}, {"integer", `10`, true},
		{"integer", `1.5`, false}, {"integer", `"10"`, false}, {"integer", `null`, false},
		{"integer", `9999999999999999999999999`, false},
		{"string_array", `["jpg","png","webp"]`, true}, {"string_array", `[]`, true},
		{"string_array", `null`, false}, {"string_array", `["jpg",null]`, false},
		{"string_array", `["jpg",1]`, false}, {"string_array", `[false]`, false},
	} {
		t.Run(tc.kind+tc.raw, func(t *testing.T) {
			d := SettingDefinition{Type: tc.kind}
			if err := d.validate(json.RawMessage(tc.raw)); (err == nil) != tc.valid {
				t.Fatalf("valid=%v err=%v", tc.valid, err)
			}
		})
	}
	defs := []SettingDefinition{
		{Key: "number", Type: "integer", Default: json.RawMessage(`10`)},
		{Key: "extensions", Type: "string_array", Default: json.RawMessage(`["jpg","png"]`)},
	}
	values := map[string]json.RawMessage{"number": json.RawMessage(`20`), "extensions": json.RawMessage(`[]`)}
	if !validCache(values, defs) {
		t.Fatal("valid typed values rejected")
	}
	rows := entries(values, defs)
	if string(rows[0].Default) != `10` || string(rows[0].Value) != `20` || string(rows[1].Default) != `["jpg","png"]` || string(rows[1].Value) != `[]` {
		t.Fatalf("invalid entries: %+v", rows)
	}
	rows[0].Default[0] = '9'
	rows[0].Value[0] = '9'
	if string(defs[0].Default) != `10` || string(values["number"]) != `20` {
		t.Fatal("entries must not mutate source values/defaults")
	}
}

func TestIncompleteCacheReloadAndRedisFallback(t *testing.T) {
	db, m := mockDB(t)
	mr := miniredis.RunT(t)
	r := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: -1})
	defer r.Close()
	testutil.UseCore(t, config.Defaults(), db, r, testLogger())
	ctx := context.Background()
	if err := r.HSet(ctx, cacheKey, "site.name", `"Old"`).Err(); err != nil {
		t.Fatal(err)
	}
	m.ExpectQuery("SELECT .*settings").WillReturnRows(sqlmock.NewRows([]string{"key", "value"}))
	name, err := String(ctx, "site.name")
	if err != nil || name != "Go Template" || r.HLen(ctx, cacheKey).Val() != 2 {
		t.Fatalf("incomplete cache not reloaded: %s %v", name, err)
	}
	if _, err := Int(ctx, "site.name"); !errors.Is(err, apperror.ErrInvalidSetting) {
		t.Fatal("integer getter accepted string", err)
	}
	if _, err := Strings(ctx, "site.maintenance"); !errors.Is(err, apperror.ErrInvalidSetting) {
		t.Fatal("array getter accepted bool", err)
	}
	list, err := List(ctx)
	if err != nil || len(list) != 2 || string(list[0].Default) != `"Go Template"` {
		t.Fatal("list missing defaults", err)
	}
	mr.SetError("ERR unavailable")
	m.ExpectQuery("SELECT .*settings").WillReturnRows(sqlmock.NewRows([]string{"key", "value"}).AddRow("site.maintenance", `true`))
	maintenance, err := Bool(ctx, "site.maintenance")
	if err != nil || !maintenance {
		t.Fatal("database fallback failed", err)
	}
	if err := m.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCommittedUpdateReportsInvalidationFailure(t *testing.T) {
	db, m := mockDB(t)
	mr := miniredis.RunT(t)
	r := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: -1})
	defer r.Close()
	testutil.UseCore(t, config.Defaults(), db, r, testLogger())
	mr.SetError("ERR unavailable")
	m.ExpectBegin()
	m.ExpectExec("INSERT INTO .*settings").WillReturnResult(sqlmock.NewResult(1, 1))
	m.ExpectCommit()
	started := time.Now()
	err := Set(context.Background(), "site.name", "Changed")
	if err == nil || !strings.Contains(err.Error(), "settings committed") {
		t.Fatal("invalidation failure returned success", err)
	}
	if time.Since(started) > 4*time.Second {
		t.Fatal("invalidation was not bounded")
	}
	if err := m.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
