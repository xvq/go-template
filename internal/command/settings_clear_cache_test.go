package command

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/xvq/go-template/internal/core/config"
	"github.com/xvq/go-template/internal/core/testutil"
)

func TestSettingsClearCacheNeedsOnlyRedis(t *testing.T) {
	m := miniredis.RunT(t)
	m.HSet("settings", "site.name", `"Old"`)
	if err := m.Set("settings:version", "7"); err != nil {
		t.Fatal(err)
	}
	testutil.UseCore(t, config.Defaults(), nil, nil, slog.Default())
	path := filepath.Join(t.TempDir(), "config.yaml")
	// Invalid database options prove the command does not initialize MySQL.
	if err := os.WriteFile(path, []byte("database:\n  port: -1\nredis:\n  address: "+m.Addr()+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := newSettingsClearCache(&path)
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	version, err := m.Get("settings:version")
	if err != nil || version != "8" || m.Exists("settings") {
		t.Fatalf("cache not invalidated: version=%s err=%v", version, err)
	}
}
