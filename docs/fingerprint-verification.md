# Fingerprint Verification

Fingerprint verification is not implemented yet. This document defines the
required verification strategy before a release can claim browser
impersonation works.

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

The preferred local verification should avoid flaky public endpoints:

1. Start the same local TLS/HTTP2 capture tools used by upstream tests.
2. Send requests through the Go client with Chrome and Firefox profiles.
3. Compare ClientHello and HTTP/2 settings/header ordering against known
   signatures.

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

Before `v0.1.0`, the repo must contain either:

- Automated integration tests that validate Chrome and Firefox signatures, or
- A documented manual smoke command with captured expected output and a reason
  automated CI is deferred.
