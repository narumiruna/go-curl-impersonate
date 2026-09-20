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

The first release path is a Linux amd64 native bundle or compatible system
installation that provides:

- Headers exposing libcurl APIs and `curl_easy_impersonate`.
- A linkable curl-impersonate library for the selected backend family.
- Pkg-config metadata for `libcurl-impersonate-chrome`,
  `libcurl-impersonate-ff`, or the generic `libcurl-impersonate` fallback.
- Runtime loader configuration through `LD_LIBRARY_PATH` or an equivalent
  system loader path.
- Version metadata in the release bundle's `VERSION` file, including the
  `go-curl-impersonate` commit and upstream curl-impersonate commit.

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

Builds that do not enable both `integration` and `native`, or that disable
cgo, return `curl.ErrNativeUnavailable` after request validation. The
`-tags="integration native"` build selects the cgo backend in
`internal/curl/perform_native.go`.

`internal/curl.NewRequestSpec` validates options and snapshots Go requests into
the method, URL, header, body, and option state translated by the cgo backend.
`RequestSpec.HeaderLines` returns deterministically ordered header lines for curl
slists. `perform_native.go` is the authoritative option order:

1. `curl_easy_impersonate`
2. `CURLOPT_NOSIGNAL`
3. `CURLOPT_TIMEOUT_MS` when a timeout is set
4. `CURLOPT_PROXY` when a proxy is set
5. `CURLOPT_FOLLOWLOCATION`
6. `CURLOPT_MAXREDIRS` when redirects are enabled and a limit is set
7. `CURLOPT_SSL_VERIFYPEER`
8. `CURLOPT_SSL_VERIFYHOST`
9. `CURLOPT_HTTP_VERSION` when HTTP/2 is requested
10. `CURLOPT_URL`
11. `CURLOPT_CUSTOMREQUEST`
12. `CURLOPT_HTTPHEADER` when headers are present
13. `CURLOPT_POSTFIELDSIZE_LARGE` and `CURLOPT_COPYPOSTFIELDS` when a buffered
    body is present

`internal/curl.ParseHeaderBlock`, `internal/curl.ResponseCollector`, and
`internal/curl.NewHTTPResponse` convert native callback state into standard
`*http.Response` values. The collector ignores informational 1xx responses and
keeps the latest final response when redirects produce multiple header blocks.
The cgo backend connects libcurl header and write callbacks to those helpers
through `goCurlHeaderCallback` and `goCurlWriteCallback`.

The backend snapshots request bodies and sends them with
`CURLOPT_COPYPOSTFIELDS`. Streaming request-body callbacks are not implemented.
It initializes and cleans up one easy handle per request, so concurrent requests
do not share an easy handle.

`internal/curl.NewError` maps native `CURLcode` values into stable Go error
kinds for DNS, connect, timeout, TLS, proxy, HTTP/2, impersonation, and unknown
failures. The cgo implementation wraps failed `curl_easy_perform` results with
this converter.
