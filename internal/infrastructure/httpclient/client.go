package httpclient

import (
	"github.com/go-resty/resty/v2"
	"github.com/xvq/go-template/internal/config"
	"net"
	"net/http"
	"time"
)

// New returns an independently configurable client. Retries and debug logging
// are OFF; enable retries per upstream only when repeating requests is safe.
func New(c config.HTTPClient) *resty.Client {
	t := &http.Transport{DialContext: (&net.Dialer{Timeout: c.ConnectTimeout, KeepAlive: 30 * time.Second}).DialContext, ForceAttemptHTTP2: true, MaxIdleConns: c.MaxIdleConns, MaxIdleConnsPerHost: c.MaxIdleConnsPerHost, IdleConnTimeout: c.IdleConnTimeout, TLSHandshakeTimeout: c.ConnectTimeout, ExpectContinueTimeout: time.Second}
	client := resty.NewWithClient(&http.Client{Transport: t, Timeout: c.Timeout}).SetRetryCount(0).SetDebug(false)
	if c.Proxy != "" {
		client.SetProxy(c.Proxy)
	}
	return client
}
