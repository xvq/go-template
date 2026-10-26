package session

import (
	"context"
	"errors"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"strings"
	"sync"
	"testing"
	"time"
)

func setup(t *testing.T) (*Store, *miniredis.Miniredis, *redis.Client) {
	t.Helper()
	m := miniredis.RunT(t)
	r := redis.NewClient(&redis.Options{Addr: m.Addr(), MaxRetries: -1})
	t.Cleanup(func() { r.Close() })
	return New(r), m, r
}
func TestConcurrentSingleSession(t *testing.T) {
	s, _, _ := setup(t)
	ctx := context.Background()
	tokens := make(chan string, 20)
	errs := make(chan error, 20)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			token, _, err := s.Issue(ctx, "admin", 1, time.Hour, false)
			if err != nil {
				errs <- err
			} else {
				tokens <- token
			}
		}()
	}
	wg.Wait()
	close(tokens)
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	count := 0
	for token := range tokens {
		if _, err := s.Validate(ctx, token); err == nil {
			count++
		} else if !errors.Is(err, ErrInvalid) {
			t.Fatal(err)
		}
	}
	if count != 1 {
		t.Fatalf("single session left %d valid tokens", count)
	}
}
func TestMultiSessionRevokeTTLAndRealm(t *testing.T) {
	s, m, r := setup(t)
	ctx := context.Background()
	a, _, err := s.Issue(ctx, "admin", 1, time.Hour, true)
	if err != nil {
		t.Fatal(err)
	}
	b, _, err := s.Issue(ctx, "admin", 1, 2*time.Hour, true)
	if err != nil {
		t.Fatal(err)
	}
	web, _, err := s.Issue(ctx, "web", 1, time.Hour, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range m.Keys() {
		if strings.Contains(key, a) || strings.Contains(key, b) {
			t.Fatal("raw token stored")
		}
	}
	if err = s.Revoke(ctx, a); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Validate(ctx, a); !errors.Is(err, ErrInvalid) {
		t.Fatal("revoked token still valid")
	}
	if _, err = s.Validate(ctx, b); err != nil {
		t.Fatal(err)
	}
	if err = s.RevokeAll(ctx, "admin", 1); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Validate(ctx, b); !errors.Is(err, ErrInvalid) {
		t.Fatal("revoke all failed")
	}
	if _, err = s.Validate(ctx, web); err != nil {
		t.Fatal("realm isolation failed", err)
	}
	m.FastForward(2 * time.Hour)
	if _, err = s.Validate(ctx, web); !errors.Is(err, ErrInvalid) {
		t.Fatal("TTL failed")
	}
	if n := r.Exists(ctx, s.indexKey("web", 1)).Val(); n != 0 {
		t.Fatal("expired index leaked")
	}
}
func TestSwitchToSingleAndInfrastructureError(t *testing.T) {
	s, m, _ := setup(t)
	ctx := context.Background()
	old, _, err := s.Issue(ctx, "admin", 1, time.Hour, true)
	if err != nil {
		t.Fatal(err)
	}
	latest, _, err := s.Issue(ctx, "admin", 1, time.Hour, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Validate(ctx, old); !errors.Is(err, ErrInvalid) {
		t.Fatal("old session kept")
	}
	m.SetError("ERR unavailable")
	if _, err = s.Validate(ctx, latest); err == nil || errors.Is(err, ErrInvalid) {
		t.Fatal("infrastructure error disguised as invalid token")
	}
	if _, _, err = s.Issue(ctx, "admin", 1, time.Hour, true); err == nil {
		t.Fatal("failed issue returned success")
	}
}
