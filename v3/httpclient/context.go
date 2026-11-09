// Package httpclient adapts SDK HTTP clients without changing process globals.
package httpclient

import (
	"context"
	"io"
	"net/http"
	"sync"
	"time"
)

// Doer is implemented by both net/http.Client and SDK HTTP client interfaces.
type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

type contextDoer struct {
	ctx  context.Context
	next Doer
}

// Bind forwards ctx to the actual HTTP request, retaining SDK cancellation too.
// The returned adapter is scoped to one SendMessageCtx call, not shared state.
func Bind(ctx context.Context, next Doer) Doer { return contextDoer{ctx, next} }
func (d contextDoer) Do(req *http.Request) (*http.Response, error) {
	if err := d.ctx.Err(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(d.ctx)
	stop := context.AfterFunc(req.Context(), cancel)
	cleanup := func() { stop(); cancel() }
	resp, err := d.next.Do(req.Clone(ctx))
	if err != nil {
		cleanup()
		return nil, err
	}
	if resp == nil || resp.Body == nil {
		cleanup()
		return resp, nil
	}
	resp.Body = &body{ReadCloser: resp.Body, cleanup: cleanup}
	return resp, nil
}

type body struct {
	io.ReadCloser
	once    sync.Once
	cleanup func()
}

func (b *body) Close() error { defer b.once.Do(b.cleanup); return b.ReadCloser.Close() }

type contextTransport struct {
	ctx  context.Context
	next http.RoundTripper
}

func (t contextTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return contextDoer{t.ctx, roundTripperDoer{t.next}}.Do(req)
}

type roundTripperDoer struct{ http.RoundTripper }

func (d roundTripperDoer) Do(req *http.Request) (*http.Response, error) { return d.RoundTrip(req) }

// BindClient makes a per-call copy; caller-owned clients and transports are unchanged.
func BindClient(ctx context.Context, base *http.Client) *http.Client {
	if base == nil {
		base = &http.Client{Timeout: 8 * time.Second}
	}
	copyBase := *base
	next := copyBase.Transport
	if next == nil {
		next = http.DefaultTransport
	}
	copyBase.Transport = contextTransport{ctx, next}
	return &copyBase
}

// BindTransport preserves SDK HTTP timeout settings while binding cancellation.
func BindTransport(ctx context.Context, next http.RoundTripper) http.RoundTripper {
	if next == nil {
		next = http.DefaultTransport
	}
	return contextTransport{ctx, next}
}
