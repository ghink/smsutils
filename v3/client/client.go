package client

import (
	"encoding/json"
	"fmt"
	"reflect"

	"go.gh.ink/smsutils/v3/errors"
	"go.gh.ink/smsutils/v3/internal/state"
	"go.gh.ink/smsutils/v3/model"
)

func NewClient(config model.Config) (clients map[string]model.Client, err error) {
	// Prepare JSON
	if config.Marshal == nil {
		config.Marshal = json.Marshal
	}
	if config.Unmarshal == nil {
		config.Unmarshal = json.Unmarshal
	}

	clients = make(map[string]model.Client)

	for driverName, driverCredential := range config.Credentials {
		// Check driver registered
		newDriverClient, ok := state.Drivers[driverName]
		if !ok {
			return nil, errors.ErrDriverNotRegistered.WithDriverName(driverName)
		}

		// Create driver client
		clients[driverName], err = newDriverClient.NewClient(model.DriverClientParam{
			Credential: driverCredential,
			// JSON
			Marshal:   config.Marshal,
			Unmarshal: config.Unmarshal,
		})
		if err != nil {
			return nil, err
		}
	}

	return clients, nil
}

// NewClientCtx constructs context-aware clients. Legacy Client and Driver
// implementations continue to work through NewClient; there is no silent fallback.
func NewClientCtx(config model.Config, options ...model.ClientOptions) (map[string]model.ClientCtx, error) {
	if len(options) > 1 {
		return nil, fmt.Errorf("at most one ClientOptions value is allowed")
	}
	var option model.ClientOptions
	if len(options) == 1 {
		option = options[0]
	}
	if config.Marshal == nil {
		config.Marshal = json.Marshal
	}
	if config.Unmarshal == nil {
		config.Unmarshal = json.Unmarshal
	}
	clients := make(map[string]model.ClientCtx, len(config.Credentials))
	for name, credential := range config.Credentials {
		registered, ok := state.Drivers[name]
		if !ok {
			return nil, errors.ErrDriverNotRegistered.WithDriverName(name)
		}
		factory, ok := registered.(model.DriverCtx)
		if !ok {
			return nil, errors.ErrDriverContextUnsupported.WithDriverName(name)
		}
		c, err := factory.NewClientCtx(model.DriverClientParam{Credential: credential, Marshal: config.Marshal, Unmarshal: config.Unmarshal}, option)
		if err != nil {
			return nil, err
		}
		if c == nil || reflect.ValueOf(c).Kind() == reflect.Pointer && reflect.ValueOf(c).IsNil() {
			return nil, errors.ErrDriverContextUnsupported.WithDriverName(name)
		}
		clients[name] = c
	}
	return clients, nil
}
