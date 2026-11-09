package client

import (
	"context"
	"errors"
	"testing"

	smserrors "go.gh.ink/smsutils/v3/errors"
	"go.gh.ink/smsutils/v3/internal/state"
	"go.gh.ink/smsutils/v3/model"
)

type ctxMock struct{ MockClient }

func (m *ctxMock) SendMessageCtx(ctx context.Context, dest, sender, template string, vars model.Vars) error {
	return ctx.Err()
}

type ctxDriver struct{}

func (ctxDriver) NewClient(p model.DriverClientParam) (model.Client, error) { return &ctxMock{}, nil }
func (ctxDriver) NewClientCtx(p model.DriverClientParam, o model.ClientOptions) (model.ClientCtx, error) {
	if p.Marshal == nil || p.Unmarshal == nil {
		return nil, errors.New("missing JSON functions")
	}
	return &ctxMock{}, nil
}
func TestContextFactoryCompatibility(t *testing.T) {
	state.Drivers["old"] = &MockDriver{mockClient: &MockClient{}}
	state.Drivers["ctx"] = ctxDriver{}
	defer delete(state.Drivers, "old")
	defer delete(state.Drivers, "ctx")
	if _, err := NewClient(model.Config{Credentials: model.C{"old": {}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := NewClientCtx(model.Config{Credentials: model.C{"old": {}}}); !errors.Is(err, smserrors.ErrDriverContextUnsupported) {
		t.Fatal(err)
	}
	c, err := NewClientCtx(model.Config{Credentials: model.C{"ctx": {}}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err = c["ctx"].SendMessageCtx(ctx, "", "", "", nil); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err = NewClientCtx(model.Config{Credentials: model.C{"missing": {}}}); !errors.Is(err, smserrors.ErrDriverNotRegistered) {
		t.Fatal(err)
	}
}
