# API Scope

This document defines the first release API surface for `go-curl-impersonate`.

## Supported in the Initial Go API Skeleton

| Area | Status | Evidence |
| --- | --- | --- |
| Client construction | Implemented | `client.NewClient` |
| Browser profile aliases | Implemented | `impersonate.Resolve`, `client.WithProfileName`, `third_party/curl-impersonate/browsers.json` parity test |
| GET/POST method passing | Implemented in the native backend | `internal/curl.NewRequestSpec`, `client/native_integration_test.go` |
| Request headers | Implemented in the native backend | `RequestSpec.HeaderLines`, `CURLOPT_HTTPHEADER`, local native GET integration test |
| Buffered request body | Implemented in the native backend | `internal/curl.NewRequestSpec` snapshots body and restores `req.Body`; `CURLOPT_COPYPOSTFIELDS` sends buffered POST bodies |
| Request option plan | Implemented | `RequestSpec.OptionSteps` fixes URL, method, header, and buffered body option order |
| Response status/headers/body | Implemented in native callbacks | `goCurlHeaderCallback`, `goCurlWriteCallback`, `internal/curl.ResponseCollector`, `internal/curl.NewHTTPResponse` |
| Cookies | Implemented through `http.CookieJar` hooks | `client.WithCookieJar`, `client.WithDefaultCookieJar`, local native GET/POST cookie integration test |
| Timeout | Implemented in the native backend | `client.WithTimeout`, `CURLOPT_TIMEOUT_MS`, native timeout test |
| Proxy | Implemented in the native backend | `client.WithProxy`, `CURLOPT_PROXY`, native proxy test |
| Redirects | Implemented in the native backend | `client.WithRedirects`, `client.WithMaxRedirects`, `CURLOPT_FOLLOWLOCATION`, `CURLOPT_MAXREDIRS`, native redirect test |
| TLS verification | Implemented in the native backend | `client.WithTLSVerify`, `CURLOPT_SSL_VERIFYPEER`, `CURLOPT_SSL_VERIFYHOST`, native HTTP/2 TLS test |
| HTTP/2 intent | Implemented in the native backend | `client.WithHTTP2`, `CURLOPT_HTTP_VERSION`, native HTTP/2 test, fingerprint verifier |
| Native option plan | Implemented | `internal/curl.NewNativePlan` validates options and `NativePlan.OptionSteps` fixes option order |
| Full operation plan | Implemented | `internal/curl.NewOperationPlan` combines native and request option order |
| Request validation | Implemented before native execution | `client.Do` returns validation errors before native-unavailable errors |
| Error conversion | Implemented in the native backend | `internal/curl.NewError`, `internal/curl.ErrorKind`, `curl_easy_perform` error wrapping |
| Easy handle lifecycle | Implemented per request | `internal/curl/perform_native.go` calls `curl_easy_init` and `curl_easy_cleanup` per request |
| Backend metadata probe | Implemented | `internal/curl.ProbeBackendPkgConfig` checks Chrome and Firefox backend metadata separately |
| Native bundle distribution | Implemented for Linux amd64 | `.github/workflows/release.yml`, `scripts/package-native-bundle.sh`, `docs/native-distribution.md` |
| Concurrency contract | Documented | README and `internal/curl/doc.go` define immutable client config and per-request easy handle ownership |

## First Release Limits

- Native requests require cgo, `-tags="integration native"`, and a compatible
  Linux amd64 curl-impersonate bundle or system installation.
- Default builds and `-tags=integration` builds intentionally keep the
  dependency-light no-native placeholder.
- Request bodies are buffered and sent with `CURLOPT_COPYPOSTFIELDS`. Streaming
  request-body read callbacks are not part of the first release backend.
- The reusable `internal/curl.HandlePool` and `internal/curl.BodyReader`
  helpers are tested groundwork, but the first cgo backend uses one fresh easy
  handle and one buffered request body per request.
- Publishing v0.1.0 is an operational release step: create a SemVer tag through
  the version-bump workflow or by pushing an annotated tag, then verify the
  release workflow publishes the Linux amd64 bundle and checksum.

## Deferred

- Python `curl_cffi` API compatibility.
- Custom JA3/Akamai/extra fingerprint overrides.
- WebSocket support.
- QUIC/HTTP/3.
- Pure Go fallback.
- Zero-setup native artifact modules or runtime-loader integration.
- macOS, Windows, and non-amd64 native bundles.

## Profile Scope

The native target source of truth for the current repo snapshot is
`third_party/curl-impersonate/browsers.json`. The first profile aliases are:

| Alias | Target | Backend family |
| --- | --- | --- |
| `chrome` | `chrome116` | `curl-impersonate-chrome` |
| `chrome_android` | `chrome99_android` | `curl-impersonate-chrome` |
| `firefox` | `ff117` | `curl-impersonate-ff` |
| `ff` | `ff117` | `curl-impersonate-ff` |
