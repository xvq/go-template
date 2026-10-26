package httpclient

import (
	"context"
	"errors"
	"github.com/xvq/go-template/internal/core/config"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestClientStreamingAndNoImplicitRetries(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path == "/fail" {
			w.WriteHeader(503)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: first\n\n")
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer srv.Close()
	c := New(config.Defaults().HTTPClient)
	defer c.GetClient().CloseIdleConnections()
	resp, err := c.R().Get(srv.URL + "/fail")
	if err != nil || resp.StatusCode() != 503 || calls.Load() != 1 {
		t.Fatalf("resp=%v err=%v calls=%d", resp, err, calls.Load())
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	resp, err = c.R().SetContext(ctx).SetDoNotParseResponse(true).Get(srv.URL + "/stream")
	if err != nil {
		t.Fatal(err)
	}
	body := resp.RawBody()
	defer body.Close()
	b := make([]byte, len("data: first\n\n"))
	if _, err = io.ReadFull(body, b); err != nil {
		t.Fatal(err)
	}
	cancel()
	_, err = body.Read(make([]byte, 1))
	if err == nil {
		t.Fatal("stream did not cancel")
	}
	canceled, stop := context.WithCancel(context.Background())
	stop()
	_, err = c.R().SetContext(canceled).Get(srv.URL)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation lost: %v", err)
	}
}
