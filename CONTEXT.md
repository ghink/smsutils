# Context-aware SMS sending

## Compatibility

The existing `model.Client`, `model.Driver`, `model.Config`, `model.DriverClientParam`,
`client.NewClient` and `SendMessage` signatures are unchanged. Existing callers and
third-party drivers can continue using them. The bundled drivers additionally expose
`SendMessageCtx`; no type assertion is needed when using the new factory:

```go
package example

import (
    "context"
    "time"

    _ "go.gh.ink/smsutils/aliyun/v3"
    "go.gh.ink/smsutils/v3/client"
    "go.gh.ink/smsutils/v3/model"
)

func Send(ctx context.Context, config model.Config, dest string) error {
    clients, err := client.NewClientCtx(config, model.ClientOptions{
        Timeout: 5 * time.Second,
        // Transport: a caller-owned, concurrency-safe http.RoundTripper,
    })
    if err != nil {
        return err
    }
    return clients["aliyun"].SendMessageCtx(ctx, dest, "YourSignName", "SMS_TEMPLATE", model.Vars{
        {Key: "code", Value: "123456"},
    })
}
```

`config.Credentials` must include `aliyun` in this example, as in the main README.
`NewClientCtx(config)` also works without explicit options. It returns
`map[string]model.ClientCtx`; those clients still support `SendMessage`.
A registered driver without `model.DriverCtx` support returns
`errors.ErrDriverContextUnsupported`, rather than silently ignoring cancellation.
The original factory still accepts that driver.

The new factory validates Tencent Cloud `smsSdkAppID` and Volcengine `smsAccount`,
as well as each provider's credentials. Negative timeouts are rejected.

## Request lifetime and network settings

- Pass a non-nil context. Cancellation and deadlines reach the actual HTTP request;
  `errors.Is(err, context.Canceled)` and `errors.Is(err, context.DeadlineExceeded)` work.
- Use `context.WithTimeout` for a precise end-to-end deadline. `ClientOptions.Timeout`
  configures the SDK/client HTTP timeout; Tencent Cloud rounds it up to whole seconds.
  Zero uses driver defaults, which differ by provider. Set an explicit deadline in services.
- `Transport` is optional and scoped to constructed clients. It is useful for proxies,
  network restrictions and tests. It must be safe for concurrent use, honor request
  contexts and obey the `http.RoundTripper` contract. Do not mutate it after construction.
- Shared SDK clients are not mutated during sends. Aliyun, BCE and UCloud use per-call
  HTTP wrappers; Tencent Cloud and Volcengine use their native context APIs. SDK retries
  are disabled on these sending paths; no automatic provider fallback is performed.
- Cancellation or a network error does not prove that the provider did not accept the
  message. Retrying blindly can send duplicates. A nil error means provider acceptance,
  not confirmed delivery to the handset.
- Keep client configuration, SDK credential fields, JSON callbacks and `vars` immutable
  while sending concurrently. Registry changes belong at initialization time.

For UCloud, use `NewClientCtx` options to configure the context-aware transport. Its SDK
has no getter for an externally installed HTTP client, so calling `SendMessageCtx` on a
legacy concrete `Client` uses a fresh context-bound HTTP wrapper and driver defaults.
Legacy `SendMessage` preserves the legacy SDK HTTP client's configuration.

## Variables and response validation

Tencent Cloud and UCloud parameters are positional: the order in `model.Vars` is preserved;
`Key` is not used for binding. Aliyun, BCE and Volcengine parameters are named: keys must
be non-empty and unique. All drivers reject nil variable entries with
`errors.ErrMessageInvalid`, before HTTP I/O. Parameterless templates can still use nil vars.
These input checks apply to both old and new send methods in the bundled drivers.

Malformed/empty responses are errors. In particular, Tencent Cloud needs exactly one
successful status for the requested phone number, UCloud needs a successful return code
and a session number, and Volcengine needs message IDs. BCE client initialization errors
are returned instead of being discarded. Tencent Cloud response envelopes are validated
before SDK parsing to prevent malformed JSON `null` from causing a panic.

## Implementing a driver

The original interfaces remain small. Context support is an additional contract:

```go
type ClientCtx interface {
    Client
    SendMessageCtx(context.Context, string, string, string, Vars) error
}

type DriverCtx interface {
    NewClientCtx(DriverClientParam, ClientOptions) (ClientCtx, error)
}
```

Implement both factory interfaces and register the driver as before. Honor cancellation
inside the provider transport; a goroutine that merely stops waiting for `SendMessage`
does not cancel a network send. Avoid changing package-global HTTP clients or credentials.

## Local integration before release

No release version has been selected. All existing `go.mod` and `go.sum` files are retained.
The driver changes depend on new core symbols and must be tested together with the local
core source until it is published. Do not run `go mod tidy` on a driver against the older
published core at this stage.

From the parent directory containing the six repositories, create a local workspace
if one does not already exist:

```sh
go work init ./smsutils/v3 ./smsutils-aliyun/v3 ./smsutils-qcloud/v3 ./smsutils-bce/v3 ./smsutils-ucloud/v3 ./smsutils-volc/v3
go test -race -timeout 60s ./smsutils/v3/... ./smsutils-aliyun/v3/... ./smsutils-qcloud/v3/... ./smsutils-bce/v3/... ./smsutils-ucloud/v3/... ./smsutils-volc/v3/...
go vet ./smsutils/v3/... ./smsutils-aliyun/v3/... ./smsutils-qcloud/v3/... ./smsutils-bce/v3/... ./smsutils-ucloud/v3/... ./smsutils-volc/v3/...
```

If a workspace exists, add these module paths using `go work use` instead of reinitializing.
Use a Go toolchain that satisfies all modules (currently at least Go 1.26.2). The workspace
uses local modules without changing driver dependency versions or adding `replace`
directives to their `go.mod` files. Keep the local workspace outside release commits.
For an application, add its module directory to the same workspace using `go work use`.

The new tests use simulated HTTP transports and placeholder credentials; they send no
real SMS. They cover cancellation, concurrent request isolation, legacy calls, variable
order, malformed responses and absence of retries on rejected HTTP requests. Real-provider
credentials and delivery behavior still need provider-side integration testing.

When a core release version is chosen, publish the core first, update each driver's
core requirement to that actual version, then run `go mod tidy` and tests per module
with `GOWORK=off` before releasing the drivers. No tags or commits are created by these steps.
