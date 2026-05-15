# Changelog

## Unreleased

- Add initial Go module and public packages.
- Add browser profile alias resolution for the checked-in curl-impersonate
  reference.
- Add high-level client configuration API.
- Add Linux amd64 cgo backend for Chrome and Firefox curl-impersonate requests.
- Add request/response translation for methods, headers, buffered bodies,
  cookies, timeout, proxy, redirects, TLS verification, HTTP/2, and native error
  conversion.
- Add local native integration tests and Chrome/Firefox TLS plus HTTP/2
  fingerprint verification against upstream fixtures.
- Add native bundle packaging, external consumer smoke tests, runtime-loader
  prototype, and GitHub Actions workflows for default checks, native checks,
  version bumping, and release publishing.
- Add docs for API scope, native API, build strategy, consumer quickstart,
  native distribution, and fingerprint verification.

## v0.1.0 alpha scope

Release candidate contents, not tagged as v0.1.0 yet:

- Native `curl-impersonate` backend for Linux amd64.
- Chrome and Firefox impersonated requests through Go API.
- Linux amd64 native bundle with headers, shared libraries, pkg-config metadata,
  version metadata, and checksums.
- Unit, race, and integration tests for basic HTTP behavior and fingerprint
  verification.
- Known limits: consumers must provide the native bundle or compatible
  pkg-config installation and build with `-tags="integration native"`; zero-setup
  native artifact modules and runtime-loader integration are deferred.
