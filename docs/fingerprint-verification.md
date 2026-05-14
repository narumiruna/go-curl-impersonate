# Fingerprint Verification

Fingerprint verification is implemented as a local smoke gate for the Go
client. It reuses upstream curl-impersonate signature fixtures and avoids
public fingerprint endpoints for the main check.

## Preferred Strategy

Reuse the upstream curl-impersonate test approach from:

- `.refs/curl-impersonate/tests/signatures/*.yaml`
- `.refs/curl-impersonate/tests/test_impersonate.py`
- `.refs/curl-impersonate/tests/README.md`

Verify the checked-in reference fixtures are present with:

```sh
sh ./scripts/check-fingerprint-fixtures.sh
```

The first release must cover at least:

- `.refs/curl-impersonate/tests/signatures/chrome.yaml`
- `.refs/curl-impersonate/tests/signatures/firefox.yaml`

The local verifier avoids flaky public endpoints:

1. Capture the Go client's TLS ClientHello with a local TCP listener.
2. Start `nghttpd -v` for HTTP/2 pseudo-header and header ordering capture.
3. Send requests through the Go client with Chrome and Firefox profiles.
4. Compare ClientHello and HTTP/2 settings/header ordering against known
   signatures.

The verifier requires the native backend build inputs plus `/usr/bin/python3`
with PyYAML and `nghttpd`:

```sh
sudo apt install python3-yaml nghttp2-server
```

Run it against a local curl-impersonate prefix:

```sh
PKG_CONFIG_PATH=/tmp/curl-impersonate-local/lib/pkgconfig \
GOCACHE=/tmp/go-build \
/usr/bin/python3 scripts/check-fingerprint.py --profile chrome
```

Current verified output for the locally built Chrome backend:

```text
TLS fingerprint matches chrome_116.0.5845.180_win10
HTTP/2 fingerprint matches chrome_116.0.5845.180_win10
```

Firefox HTTP/2 header ordering is also verified:

```sh
PKG_CONFIG_PATH=/tmp/curl-impersonate-local/lib/pkgconfig \
GOCACHE=/tmp/go-build \
/usr/bin/python3 scripts/check-fingerprint.py --profile firefox --skip-tls
```

Current verified output:

```text
HTTP/2 fingerprint matches firefox_117.0.1_win10
```

Firefox TLS verification is still blocked by one fixture mismatch in the local
capture:

```text
TLS fingerprint mismatch for firefox_117.0.1_win10: TLS extension lists differ: Symmatric difference [<TLSExtensionType.psk_key_exchange_modes: 45>]
```

That mismatch means the repo must not yet claim complete Firefox TLS
fingerprint parity.

## Public Endpoint Smoke Test

External fingerprint endpoints may be useful for manual inspection, but they
must not be the only CI gate. Public services can change output or rate-limit
requests.

The current manual HTTP smoke command is:

```sh
PKG_CONFIG_PATH=/tmp/curl-impersonate-local/lib/pkgconfig \
LD_LIBRARY_PATH=/tmp/curl-impersonate-local/lib \
sh ./scripts/smoke-atp.sh
```

It uses `examples/basic`, which requests:

```text
https://app.atptour.com/api/v2/gateway/livematches/website?scoringTournamentLevel=tour
```

This smoke command first runs `scripts/check-native.sh`, so it fails fast when
curl-impersonate headers/libraries are not available. It is not a replacement
for local fingerprint verification.

With the locally built Chrome and Firefox backends installed under
`/tmp/curl-impersonate-local`, `scripts/check-native.sh` verified both native
libraries against local Go integration tests. The public ATP smoke then returned:

```text
200 OK
```

## Release Requirement

Before `v0.1.0`, the remaining release blocker is to resolve or explain the
Firefox TLS `psk_key_exchange_modes` mismatch with upstream parity evidence.
