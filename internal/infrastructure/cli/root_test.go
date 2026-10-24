package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHelpAndHelloWithoutConfig(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"server", "--help"}, {"queue:work", "--help"}, {"hello"}, {"migrate", "--help"}} {
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
	root.SetArgs([]string{"-c", path, "queue:work"})
	if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "队列未启用") {
		t.Fatal(err)
	}
	root = NewRoot()
	root.SetArgs([]string{"-c", "missing.yaml", "migrate", "down"})
	if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatal(err)
	}
}

func TestCommandConfigurationValidatedBeforeResources(t *testing.T) {
	for _, tc := range []struct{ command, config, want string }{
		{"server", "server:\n  port: -1\n", "server 配置无效"},
		{"schedule", "schedule:\n  timezone: invalid/timezone\n", "schedule.timezone 无效"},
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
