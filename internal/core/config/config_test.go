package config

import (
	template "github.com/xvq/go-template"
	"testing"
	"time"
)

func TestExampleAndScopedValidation(t *testing.T) {
	c, err := Parse(template.ExampleConfig)
	if err != nil {
		t.Fatal(err)
	}
	if err = c.Validate("server", "database", "redis", "http_client", "schedule"); err != nil {
		t.Fatal(err)
	}
	if c.Auth.Admin.TokenTTL != 24*time.Hour || !c.Auth.Admin.AllowMultipleSessions || c.Queue.Enabled || c.App.Timezone != "Asia/Hong_Kong" {
		t.Fatal("wrong example defaults")
	}
	c.Database.Database = ""
	c.Redis.Address = ""
	if c.Validate("schedule") != nil {
		t.Fatal("schedule must not require DB/Redis")
	}
	if c.Validate("database") == nil {
		t.Fatal("missing database accepted")
	}
	if c.Validate("queue") == nil {
		t.Fatal("disabled queue accepted")
	}
}

func TestApplicationTimezone(t *testing.T) {
	for _, zone := range []string{"Asia/Hong_Kong", "America/New_York", "UTC"} {
		c := Defaults()
		c.App.Timezone = zone
		loc, err := c.App.Location()
		if err != nil || loc.String() != zone {
			t.Fatalf("timezone %s: %v %v", zone, loc, err)
		}
		if err := c.Validate("schedule"); err != nil {
			t.Fatal(err)
		}
	}
	for _, zone := range []string{"", "Local", "invalid/timezone"} {
		c := Defaults()
		c.App.Timezone = zone
		if c.Validate() == nil || c.Validate("server") == nil || c.Validate("schedule") == nil {
			t.Fatalf("invalid shared timezone accepted: %q", zone)
		}
	}
	if _, err := Parse([]byte("schedule:\n  timezone: UTC\n")); err == nil {
		t.Fatal("removed schedule.timezone accepted")
	}
}
func TestStrictConfigAndFalseValues(t *testing.T) {
	if _, err := Parse([]byte("redis:\n  adress: localhost\n")); err == nil {
		t.Fatal("unknown field accepted")
	}
	if _, err := Parse([]byte("app: {}\n---\napp: {}\n")); err == nil {
		t.Fatal("multiple documents accepted")
	}
	c, err := Parse([]byte("auth:\n  admin:\n    allow_multiple_sessions: false\n"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Auth.Admin.AllowMultipleSessions {
		t.Fatal("explicit false lost")
	}
	c.Server.Port = -1
	if c.Validate("server") == nil {
		t.Fatal("bad port accepted")
	}
}
