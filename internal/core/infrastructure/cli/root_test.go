package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHelpAndHelloWithoutConfig(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"server", "--help"}, {"queue:work", "--help"}, {"hello"}, {"migrate", "--help"}, {"settings:clear-cache", "--help"}} {
		root := NewRoot()
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetErr(&out)
		root.SetArgs(append([]string{"-c", "not-existing.yaml"}, args...))
		if err := root.Execute(); err != nil {
			t.Fatal(args, err)
		}
	}
}
func TestInitDoesNotOverwriteAndQueueDisabled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	root := NewRoot()
	root.SetArgs([]string{"-c", path, "config:init"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	root = NewRoot()
	root.SetArgs([]string{"-c", path, "config:init"})
	if root.Execute() == nil {
		t.Fatal("overwrote config")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("config changed")
	}
	root = NewRoot()
	var startup bytes.Buffer
	root.SetErr(&startup)
	root.SetArgs([]string{"-c", path, "queue:work"})
	if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "队列未启用") {
		t.Fatal(err)
	}
	if strings.Contains(startup.String(), "Queue Worker started") {
		t.Fatal("disabled queue printed a startup notice")
	}
	root = NewRoot()
	root.SetArgs([]string{"-c", "missing.yaml", "migrate", "down"})
	if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatal(err)
	}
}

func TestScheduleStartupNoticeIndependentOfLogLevel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("log:\n  level: error\n"), 0600); err != nil {
		t.Fatal(err)
	}
	root := NewRoot()
	var out bytes.Buffer
	root.SetErr(&out)
	root.SetArgs([]string{"-c", path, "schedule"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := root.ExecuteContext(ctx); err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), "Scheduler started, registered tasks: 1\n"; got != want {
		t.Fatalf("startup notice=%q, want %q", got, want)
	}
}

func TestCommandConfigurationValidatedBeforeResources(t *testing.T) {
	for _, tc := range []struct{ command, config, want string }{
		{"server", "server:\n  port: -1\n", "server 配置无效"},
		{"schedule", "app:\n  timezone: invalid/timezone\n", "app.timezone 无效"},
	} {
		t.Run(tc.command, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			if err := os.WriteFile(path, []byte(tc.config), 0600); err != nil {
				t.Fatal(err)
			}
			root := NewRoot()
			root.SetArgs([]string{"-c", path, tc.command})
			if err := root.Execute(); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want command configuration error %q, got %v", tc.want, err)
			}
		})
	}
}
