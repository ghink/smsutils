package httpclient

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundtrip func(*http.Request) (*http.Response, error)

func (f roundtrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestCallerAndSDKCancellation(t *testing.T) {
	for _, caller := range []bool{true, false} {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		sdkCtx, sdkCancel := context.WithCancel(context.Background())
		defer sdkCancel()
		started := make(chan struct{})
		base := &http.Client{Transport: roundtrip(func(r *http.Request) (*http.Response, error) {
			close(started)
			<-r.Context().Done()
			return nil, r.Context().Err()
		})}
		r, _ := http.NewRequestWithContext(sdkCtx, "POST", "https://sms.example.test", nil)
		done := make(chan error, 1)
		go func() { _, err := BindClient(ctx, base).Do(r); done <- err }()
		<-started
		if caller {
			cancel()
		} else {
			sdkCancel()
		}
		select {
		case err := <-done:
			if !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
		case <-time.After(time.Second):
			t.Fatal("HTTP request was not cancelled")
		}
	}
}

type contextBody struct {
	ctx context.Context
	io.Reader
}

func (b contextBody) Read(p []byte) (int, error) {
	if err := b.ctx.Err(); err != nil {
		return 0, err
	}
	return b.Reader.Read(p)
}
func (b contextBody) Close() error { return nil }
func TestResponseBodyRemainsReadableAndClientUnchanged(t *testing.T) {
	tr := roundtrip(func(r *http.Request) (*http.Response, error) {
		if err := r.Context().Err(); err != nil {
			t.Fatal(err)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: contextBody{ctx: r.Context(), Reader: strings.NewReader("ok")}}, nil
	})
	base := &http.Client{Transport: &tr, Timeout: time.Second}
	original := base.Transport
	bound := BindClient(context.Background(), base)
	r, _ := http.NewRequest("POST", "https://sms.example.test", nil)
	res, err := bound.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(res.Body)
	if err != nil || string(b) != "ok" {
		t.Fatal(string(b), err)
	}
	res.Body.Close()
	// Pointer identity also detects replacement of the original transport.
	if base.Timeout != time.Second || bound == base || base.Transport != original {
		t.Fatal("caller client mutated")
	}
}
