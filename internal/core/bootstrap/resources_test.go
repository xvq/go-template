package bootstrap

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/xvq/go-template/internal/core"
	"github.com/xvq/go-template/internal/core/config"
	"github.com/xvq/go-template/internal/core/infrastructure/session"
	"github.com/xvq/go-template/internal/core/infrastructure/taskqueue"
)

func TestGlobalResourcesLifecycle(t *testing.T) {
	c := config.Defaults()
	c.Queue.Enabled = true // An unrequested queue must still remain disconnected.
	r, err := Open(context.Background(), c, Options{Name: "test", HTTP: true})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if core.HTTP == nil || core.Log == nil || core.Config.App.Name != c.App.Name {
		t.Fatal("globals were not published")
	}
	if core.DB != nil || core.Redis != nil || core.Session != nil {
		t.Fatal("initialized unrequested components")
	}
	if _, err := core.Queue.Enqueue(context.Background(), "test", nil); !errors.Is(err, taskqueue.ErrDisabled) {
		t.Fatalf("unrequested queue is not disabled: %v", err)
	}
	http := core.HTTP
	if _, err := Open(context.Background(), c, Options{Name: "other"}); err == nil {
		t.Fatal("replaced active resources")
	}
	if core.HTTP != http {
		t.Fatal("failed Open changed globals")
	}
	r.Close()
	if core.DB != nil || core.Redis != nil || core.Session != nil || core.HTTP != nil || core.Queue != nil || core.Log != nil || core.Config.App.Name != "" {
		t.Fatal("closed globals were not cleared")
	}
	next, err := Open(context.Background(), c, Options{Name: "next"})
	if err != nil {
		t.Fatal(err)
	}
	defer next.Close()
	r.Close() // An old owner must not close or clear the new resource group.
	if core.Log == nil {
		t.Fatal("old Close cleared new resources")
	}
}

func TestFailedOpenDoesNotPublishAndCanRetry(t *testing.T) {
	mr := miniredis.RunT(t)
	c := config.Defaults()
	c.Redis.Address = mr.Addr()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Open(ctx, c, Options{Name: "test", HTTP: true, Redis: true}); err == nil {
		t.Fatal("canceled Redis initialization succeeded")
	}
	if core.HTTP != nil || core.Log != nil || core.Redis != nil || core.Session != nil {
		t.Fatal("partial resources were published")
	}
	r, err := Open(context.Background(), c, Options{Name: "test", Redis: true})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	client := core.Redis
	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Fatal(err)
	}
	if core.Session == nil {
		t.Fatal("Redis initialization did not prepare the shared session store")
	}
	token, _, err := core.Session.Issue(context.Background(), "admin", 7, time.Hour, true)
	if err != nil {
		t.Fatal(err)
	}
	if !mr.Exists("auth:session:" + session.Hash(token)) {
		t.Fatal("session store did not use the shared Redis")
	}
	identity, err := core.Session.Validate(context.Background(), token)
	if err != nil || identity.SubjectID != 7 || identity.Realm != "admin" {
		t.Fatalf("shared session validation failed: identity=%+v err=%v", identity, err)
	}
	r.Close()
	if core.Session != nil {
		t.Fatal("Close did not clear the shared session store")
	}
	if err := client.Ping(context.Background()).Err(); err == nil {
		t.Fatal("Redis connection was not closed")
	}
}

func TestOpenValidatesOnlySelectedResources(t *testing.T) {
	c := config.Defaults()
	c.Server.Port = -1
	c.Schedule.Timezone = "invalid/timezone"
	c.Redis.Address = ""
	// Command settings and unrequested Redis must not affect resource startup.
	r, err := Open(context.Background(), c, Options{Name: "test", HTTP: true})
	if err != nil {
		t.Fatal(err)
	}
	r.Close()
	if _, err := Open(context.Background(), c, Options{Name: "test", Redis: true}); err == nil {
		t.Fatal("selected Redis configuration was not validated")
	}
	if core.Log != nil || core.Redis != nil || core.Session != nil {
		t.Fatal("invalid options published globals")
	}
}
