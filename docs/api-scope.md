# API Scope

This document defines the first release API surface for `go-curl-impersonate`.

## Supported in the Initial Go API Skeleton

| Area | Status | Evidence |
| --- | --- | --- |
| Client construction | Implemented | `client.NewClient` |
| Browser profile aliases | Implemented | `impersonate.Resolve`, `client.WithProfileName`, `.refs/curl-impersonate/browsers.json` parity test |
| GET/POST method passing | Request translation implemented, native pending | `internal/curl.NewRequestSpec` snapshots method |
| Request headers | Request translation implemented, native pending | `internal/curl.NewRequestSpec` snapshots headers; `RequestSpec.HeaderLines` emits deterministic curl slist lines |
| Request body | Request translation implemented, native pending | `internal/curl.NewRequestSpec` snapshots body and restores `req.Body` |
| Request option plan | Implemented, cgo pending | `RequestSpec.OptionSteps` fixes URL, method, header, and body callback operation order |
| Request body callback state | Implemented, cgo pending | `internal/curl.BodyReader`, `internal/curl.ReadBodyChunk` |
| Response status/headers/body | Response construction implemented, native callback pending | `internal/curl.ParseHeaderBlock`, `internal/curl.ResponseCollector`, `internal/curl.NewHTTPResponse` |
| Cookies | Request-side and response-side jar hooks implemented, native callback pending | `client.WithCookieJar`, `client.WithDefaultCookieJar` |
| Timeout | Option implemented, native pending | `client.WithTimeout` |
| Proxy | Option implemented, native pending | `client.WithProxy` |
| Redirects | Option implemented, native pending | `client.WithRedirects`, `client.WithMaxRedirects`, `CURLOPT_MAXREDIRS` native step |
| TLS verification | Option implemented, native pending | `client.WithTLSVerify` |
| HTTP/2 intent | Option implemented, native pending | `client.WithHTTP2` |
| Native option plan | Implemented, cgo pending | `internal/curl.NewNativePlan` validates options and `NativePlan.OptionSteps` fixes option order |
| Full operation plan | Implemented, cgo pending | `internal/curl.NewOperationPlan` combines native and request option order |
| Request validation | Implemented before native execution | `client.Do` returns validation errors before native-unavailable errors |
| Error conversion | Implemented, cgo pending | `internal/curl.NewError`, `internal/curl.ErrorKind` |
| Handle lease lifecycle | Implemented, cgo pending | `internal/curl.HandlePool`, race-tested lease/release |
| Backend metadata probe | Implemented, native artifacts pending | `internal/curl.ProbeBackendPkgConfig` checks Chrome and Firefox backend metadata separately |
| Concurrency contract | Documented | README and `internal/curl/doc.go` define immutable client config and per-request easy handle ownership |

## Native Integration Pending

The following items require `internal/curl` to translate Go request state to
libcurl options:

- Easy handle lifecycle.
- Header and body callbacks.
- Native wiring for error code conversion.
- Native callback wiring for response status/header/body bytes.
- Native callback wiring for response cookie jar updates.
- Proxy configuration.
- Timeout configuration.
- Redirect policy.
- TLS verification switch.
- HTTP version and HTTP/2 settings.
- Browser impersonation through `curl_easy_impersonate`.

The native backend must allocate or lease one easy handle per active request.
It must not use one easy handle concurrently.

## Deferred

- Python `curl_cffi` API compatibility.
- Custom JA3/Akamai/extra fingerprint overrides.
- WebSocket support.
- QUIC/HTTP/3.
- Pure Go fallback.
- Prebuilt binary distribution.

## Profile Scope

The native target source of truth for the current repo snapshot is
`.refs/curl-impersonate/browsers.json`. The first profile aliases are:

| Alias | Target | Backend family |
| --- | --- | --- |
| `chrome` | `chrome116` | `curl-impersonate-chrome` |
| `chrome_android` | `chrome99_android` | `curl-impersonate-chrome` |
| `firefox` | `ff117` | `curl-impersonate-ff` |
| `ff` | `ff117` | `curl-impersonate-ff` |
