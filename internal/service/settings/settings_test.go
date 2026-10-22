package settings

import (
	"context"
	"encoding/json"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"testing"
	"time"
)

func TestSettingsCacheUpdateAndValidation(t *testing.T) {
	db, m := mockDB(t)
	mr := miniredis.RunT(t)
	r := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer r.Close()
	s := NewSettings(db, r, "test", time.Minute, testLogger())
	ctx := context.Background()
	m.ExpectQuery("SELECT .*settings").WillReturnRows(sqlmock.NewRows([]string{"key", "value"}))
	name, err := s.String(ctx, "site.name")
	if err != nil || name != "Go Template" {
		t.Fatalf("%s %v", name, err)
	}
	if _, err = s.String(ctx, "site.name"); err != nil {
		t.Fatal("cache miss", err)
	}
	if err = s.Update(ctx, map[string]json.RawMessage{"unknown": json.RawMessage(`true`)}); err == nil {
		t.Fatal("unknown key accepted")
	}
	if err = s.Update(ctx, map[string]json.RawMessage{"site.maintenance": json.RawMessage(`"false"`)}); err == nil {
		t.Fatal("wrong type accepted")
	}
	if err = s.Update(ctx, map[string]json.RawMessage{"site.maintenance": json.RawMessage(` null `)}); err == nil {
		t.Fatal("null boolean accepted")
	}
	m.ExpectBegin()
	m.ExpectExec("INSERT INTO .*settings").WillReturnResult(sqlmock.NewResult(1, 1))
	m.ExpectCommit()
	if err = s.Update(ctx, map[string]json.RawMessage{"site.name": json.RawMessage(`"Changed"`)}); err != nil {
		t.Fatal(err)
	}
	m.ExpectQuery("SELECT .*settings").WillReturnRows(sqlmock.NewRows([]string{"key", "value"}).AddRow("site.name", `"Changed"`))
	name, err = s.String(ctx, "site.name")
	if err != nil || name != "Changed" {
		t.Fatalf("stale setting: %s %v", name, err)
	}
	if err = m.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
func TestStaleCacheFillRejectedAndSensitiveOmitted(t *testing.T) {
	db, m := mockDB(t)
	mr := miniredis.RunT(t)
	r := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer r.Close()
	s := NewSettings(db, r, "test", time.Minute, testLogger())
	ctx := context.Background()
	if err := invalidateSettings.Run(ctx, r, []string{s.key, s.key + ":version"}).Err(); err != nil {
		t.Fatal(err)
	}
	n, err := fillSettings.Run(ctx, r, []string{s.key, s.key + ":version"}, "0", `{"site.name":"stale"}`, 60000).Int()
	if err != nil || n != 0 {
		t.Fatalf("stale cache fill accepted %d %v", n, err)
	}
	s.defs = append(s.defs, SettingDefinition{Key: "private.secret", Type: "string", Default: json.RawMessage(`"secret"`), Sensitive: true, Validate: func(json.RawMessage) error { return nil }})
	m.ExpectQuery("SELECT .*settings").WillReturnRows(sqlmock.NewRows([]string{"key", "value"}))
	out, err := s.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range out {
		if v.Sensitive && v.Value != nil {
			t.Fatal("secret exposed")
		}
	}
	if err = m.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
