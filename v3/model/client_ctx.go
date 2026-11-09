package model

import (
	"context"
	"net/http"
	"time"
)

// ClientCtx extends Client without changing the interface implemented by legacy drivers.
type ClientCtx interface {
	Client
	SendMessageCtx(ctx context.Context, dest, sender, template string, vars Vars) error
}

// ClientOptions configures clients created by NewClientCtx. A nil Transport and zero
// Timeout use driver defaults. Transports must be safe for concurrent use.
type ClientOptions struct {
	Transport http.RoundTripper
	Timeout   time.Duration
}

// DriverCtx is optional. NewClientCtx rejects drivers that do not implement it.
type DriverCtx interface {
	NewClientCtx(params DriverClientParam, options ClientOptions) (ClientCtx, error)
}
