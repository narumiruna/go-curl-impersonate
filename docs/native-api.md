# Native API

`go-curl-impersonate` should call `libcurl-impersonate` rather than
reimplementing TLS or HTTP/2 fingerprints.

## Required C API

The upstream reference documents the additional impersonation function as:

```c
CURLcode curl_easy_impersonate(struct Curl_easy *data, const char *target, int default_headers);
```

The Go binding should declare this function at the cgo boundary and call it
after `curl_easy_init` and before request-specific overrides that are meant to
win over profile defaults.

## Backend Families

The checked-in `third_party/curl-impersonate/browsers.json` maps browser targets to
two backend families:

- `curl-impersonate-chrome`: Chrome, Edge, Safari, and Chrome Android targets.
- `curl-impersonate-ff`: Firefox targets.

The Go package must fail early when the requested profile requires a backend
family that is not installed or linked.

## Expected Linking Inputs

The exact final strategy is still open, but the implementation must document
one reproducible Linux amd64 path before native integration is considered done:

- Headers exposing libcurl APIs and `curl_easy_impersonate`.
- A linkable curl-impersonate library for the selected backend family.
- Runtime loader configuration for the selected shared library.
- The minimum curl-impersonate version or commit used for tests.

## Option Ordering

`curl_easy_impersonate` sets browser-specific curl options. Later
`curl_easy_setopt` calls can override those options, so the binding should:

1. Create or reset the easy handle.
2. Apply `curl_easy_impersonate`.
3. Apply request-specific URL, method, headers, body, proxy, timeout, redirect,
   TLS verify, and HTTP version options.
4. Perform the request.
5. Release request-owned C strings, slists, and buffers.

## Current Implementation State

The default build does not link native libraries yet. `internal/curl` exposes
the boundary and returns `curl.ErrNativeUnavailable` until an integration build
tag and cgo implementation are added.

`internal/curl.NewRequestSpec` now snapshots validated Go requests into the
method, URL, header, body, and option state that the cgo implementation should
translate to `curl_easy_setopt` calls. `RequestSpec.HeaderLines` returns
deterministically ordered header lines for curl slists. `RequestSpec.OptionSteps`
fixes the request-specific operation order:

1. `CURLOPT_URL`
2. `CURLOPT_CUSTOMREQUEST`
3. `CURLOPT_HTTPHEADER` when headers are present
4. `CURLOPT_POSTFIELDSIZE_LARGE` when a body is present
5. `CURLOPT_READFUNCTION` when a body is present

`internal/curl.ParseHeaderBlock`, `internal/curl.ResponseCollector`, and
`internal/curl.NewHTTPResponse` convert native callback state into standard
`*http.Response` values. The collector ignores informational 1xx responses and
keeps the latest final response when redirects produce multiple header blocks.
The remaining work is to connect libcurl header/body callbacks to those helpers.

`internal/curl.BodyReader` and `ReadBodyChunk` define the request-body read
callback state machine, including partial reads, EOF, reset, and error status.

`internal/curl.NewError` maps native `CURLcode` values into stable Go error
kinds for DNS, connect, timeout, TLS, proxy, HTTP/2, impersonation, and unknown
failures. The cgo implementation should wrap failed `curl_easy_perform` results
with this converter.

`internal/curl.HandlePool` defines the easy-handle lease lifecycle used by the
future native backend: each active request gets exclusive ownership of one
handle lease, released handles may be reused, and closed pools reject new
leases.

`internal/curl.NewNativePlan` validates and normalizes profile, default header,
timeout, proxy, redirect, TLS verification, and HTTP/2 settings before the cgo
implementation maps them to curl options.

`NativePlan.OptionSteps` fixes the expected operation order:

1. `curl_easy_impersonate.target`
2. `curl_easy_impersonate.default_headers`
3. `CURLOPT_TIMEOUT_MS` when a timeout is set
4. `CURLOPT_PROXY` when a proxy is set
5. `CURLOPT_FOLLOWLOCATION`
6. `CURLOPT_MAXREDIRS` when redirects are enabled and a limit is set
7. `CURLOPT_SSL_VERIFYPEER`
8. `CURLOPT_SSL_VERIFYHOST`
9. `CURLOPT_HTTP_VERSION` when HTTP/2 is requested

`internal/curl.NewOperationPlan` combines native profile/options and
request-specific options into one ordered operation list. The cgo backend should
apply the native plan first, then URL, method, headers, and body callback
settings.
