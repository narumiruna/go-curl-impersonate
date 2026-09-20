# Repository Simplification Plan

## Goal

Reduce maintenance burden and independent implementations while preserving the current public Go API, request behavior, native fingerprints, supported build-tag matrix, release artifacts, and diagnostic CLI output.

Success means the runtime has one authoritative native option path, unused speculative code is removed, no-native builds use one implementation, native CI/release logic has one owner, and all available behavior checks pass.

## Context

The repository currently has a direct `client.Do` → `internal/curl.Perform` request path, but several surrounding implementations duplicate or model behavior that is implemented elsewhere:

- `NativePlan`, `OptionStep`, and `OperationPlan` symbolically duplicate option ordering from `internal/curl/perform_native.go` without driving it.
- `BodyReader` and `HandlePool` are tested future groundwork with no production callers.
- Default and integration-only builds select identical no-native implementations from two files.
- Native and release workflows independently define nearly the same native build, fingerprint, smoke, package, and artifact pipeline.
- A release-critical runtime-loader prototype implements a deferred native-loading design that is not used by the shipped client.
- `.gitignore` contains an exact 221-line duplicate block.
- Native callback state includes branches that cannot be reached.

The reviewed baseline passed `go test ./...`, `go test -tags=integration ./...`, `go test -race ./...`, `go vet ./...`, Go formatting, shell syntax, and `git diff --check`. Native libraries, initialized upstream fixtures, and `nghttpd` were unavailable during review, so native and fingerprint checks must be run during implementation in a prepared environment or GitHub Actions.

## Non-Goals

- Change the public `client` or `impersonate` API.
- Change request validation order, error strings, timeout rounding, cookie behavior, response behavior, or libcurl option order.
- Replace buffered request bodies with streaming bodies.
- Add handle pooling or runtime loading.
- Change supported browser targets, native library names, bundle contents, or platform support.
- Replace the existing `integration native` build-tag contract.
- Broadly redesign documentation or `.gitignore` beyond references and exact duplication affected by this work.

## Assumptions

- Symbols under `internal/curl` that have no production callers are not supported consumer APIs.
- `scripts/prototype-runtime-loader.sh` is experimental, as documented, and is not a supported consumer interface.
- SemVer release tags should execute the native verification pipeline once before publication; preserving its gates and outputs matters, not preserving duplicate workflow runs.
- A native prefix with Chrome and Firefox pkg-config metadata can be provided locally or through GitHub Actions before completion.

## Architecture

Keep these boundaries:

- `client`: public configuration and `net/http` integration.
- `impersonate`: public profile and backend-family resolution.
- `internal/curl/request.go`: request validation and immutable request snapshot.
- `internal/curl/perform_native.go`: the single authoritative libcurl option order and native resource lifecycle.
- One no-native implementation selected for every build that does not satisfy `integration && native && cgo`.
- One reusable GitHub Actions native pipeline called by direct native checks and release publication.

## Plan

- [x] Record the implementation baseline and native test prerequisites: `go test ./...`, `go test -tags=integration ./...`, `go test -race ./...`, `go vet ./...`, Go formatting, shell syntax, and `git diff --check` passed on `main` at `31bf49b`; no local native prefix, curl-impersonate pkg-config metadata, initialized submodule, or `nghttpd` was available, so Chrome/Firefox validation will use a manual GitHub Actions native run on this branch.

- [x] Replace the shadow option-planning model in `internal/curl/options.go`, `internal/curl/operation.go`, and `internal/curl/request.go` with one small options validator plus the existing timeout conversion; make `applyNativeOptions` consume validated `Options` directly while retaining its current operation order; acceptance is removal of `NativePlan`, `OptionStep`, `OperationPlan`, and both `OptionSteps` methods with unchanged validation errors and timeout tests.

- [ ] Update option-related tests so they verify request validation and timeout normalization rather than a symbolic operation list; remove `internal/curl/operation_test.go` and plan-only assertions from `options_test.go` and `request_test.go`; acceptance is coverage of empty profile, negative timeout, negative redirect count, sub-millisecond timeout rounding, request snapshots, and native behavior through the actual integration path.

- [ ] Remove unused future groundwork from `internal/curl/body.go`, `internal/curl/body_test.go`, `internal/curl/handle.go`, and `internal/curl/handle_test.go`; acceptance is no remaining references to `BodyReader`, `ReadBodyChunk`, `HandlePool`, `HandleLease`, or `ErrHandlePoolClosed`, while buffered-body and concurrent-client tests still pass.

- [ ] Simplify native callback state in `internal/curl/perform_native.go` by deleting the never-assigned `writeErr` field/check and the impossible `bytes.Buffer.Write` error branch while retaining real header parsing errors; acceptance is no unreachable callback error state and passing response/redirect integration tests.

- [x] Consolidate `internal/curl/perform_stub.go` and `internal/curl/perform_integration.go` into one no-native implementation guarded by `!integration || !native || !cgo`; acceptance is identical source selection and behavior for default, integration-only, native-without-cgo, and `integration native` builds, demonstrated with `go list` and the available test matrix.

- [x] Remove the redundant integration-only CI pass from `.github/workflows/test.yml` and the release Go-check block after the consolidated stub proves that `-tags=integration` selects no distinct implementation; acceptance is retention of default, race, and real native checks without a duplicate placeholder test run.

- [ ] Remove the deferred runtime-loader experiment from `scripts/prototype-runtime-loader.sh`, `.github/workflows/native.yml`, `.github/workflows/release.yml`, and its operational documentation; retain only a concise deferred-design note if useful; acceptance is no release gate or maintained implementation for runtime loading, while compile-time native consumer smoke tests still pass.

- [x] Extract the common native build pipeline from `.github/workflows/native.yml` and `.github/workflows/release.yml` into one reusable workflow with inputs for local-checkout versus tagged-module smoke testing; keep tag validation and GitHub Release publication in the release workflow; acceptance is one definition of dependency installation, native build, Chrome/Firefox checks, fingerprint checks, consumer smoke, package creation, and artifact upload.

- [ ] Adjust native workflow triggers so manual runs, relevant `main` changes, non-release native tag checks if still required, and SemVer releases retain equivalent coverage without running the shared native pipeline twice for one SemVer tag; acceptance is a documented trigger matrix and one native artifact-producing execution before each release publication.

- [x] Remove the exact duplicate `.gitignore` block at lines 316–536 while retaining the first copy and project-specific `.refs/` and `references/` entries; acceptance is equivalent `git check-ignore` results for every distinct pattern and representative nested paths before and after the edit.

- [x] Align `docs/api-scope.md`, `docs/native-api.md`, `docs/build.md`, `docs/native-distribution.md`, `README.md`, and `CHANGELOG.md` with the simplified implementation: remove symbolic plans, future groundwork, integration-only placeholder checks, and runtime-loader claims without changing documented public behavior; acceptance is no stale symbol, script, or workflow reference in active documentation or source found by `rg`. Historical completed plans are left unchanged.

- [x] Run the default validation suite after all code and documentation changes: `gofmt`, `sh -n scripts/*.sh`, `go test ./...`, `go test -race ./...`, `go vet ./...`, `git diff --check`, and the default diagnostic CLI; acceptance is every command passing and diagnostic output retaining its existing sections and actionable native setup guidance.

- [ ] Run the native behavior-preservation suite with both backend families: `scripts/check-native.sh <prefix>`, Chrome and Firefox fingerprint verification, external-module smoke testing, and native bundle packaging; acceptance is passing Chrome/Firefox local integration tests, matching TLS/HTTP2 fixtures, successful external consumption, and an artifact with the same name and required contents.

- [ ] Validate the workflow consolidation with `actionlint`, one manual native run, and one non-publishing SemVer-tag rehearsal or equivalent controlled test; acceptance is a single native pipeline execution, successful artifact transfer to the release job, and publication remaining gated on all native checks.

## Implementation Evidence

- Runtime validation and timeout tests pass, including exact error text, precedence of URL/options/body errors, and fractional-millisecond truncation.
- `go list` verified all eight combinations of `integration`, `native`, and cgo select exactly the intended backend. Default, integration-only, native-only, and cgo-disabled combined-tag tests pass.
- The native `checkCode` operation sequence is unchanged from `origin/main`; public `client`, `impersonate`, and diagnostic CLI implementation files are unchanged.
- `.gitignore` before/after checks matched for 618 representative root and nested paths, including negated `.pixi/config.toml`; the removed block was confirmed byte-identical.
- `actionlint` v1.7.7, per-script shell syntax checks, formatting, `go vet`, default/race tests, `git diff --check`, and CLI diagnostics pass. Local fixture validation skips because the submodule is absent.
- The shared native pipeline now includes a separate artifact download, checksum/content verification, and unpacked-bundle smoke job. This supplies a non-publishing cross-job artifact-handoff check; release publication remains dependent on the full shared pipeline. No tag or release workflow will be created or dispatched during this work.
- Full native validation and the new concurrency test remain pending a branch-native workflow run.

## Risks

- Removing the symbolic plan tests can reduce perceived unit coverage unless actual validation and native integration checks remain explicit.
- Build constraints are easy to get wrong across cgo and tag combinations; verify source selection, not only successful default compilation.
- Reusable workflow artifact and secret behavior differs from ordinary jobs; validate artifact download and token permissions before relying on the release path.
- Native validation is unavailable in the current plain checkout, so completion must not be declared from default Go tests alone.
- Removing the runtime-loader script is safe only while it remains an explicitly experimental, unsupported path.

## Rollback / Recovery

- Keep workflow consolidation and runtime code changes in separable commits so the release workflow can be restored without reverting internal cleanup.
- If reusable workflow artifact transfer or permissions fail, restore the previous release job from Git while retaining already-validated Go simplifications.
- If native behavior differs after removing `NativePlan`, restore the prior option application commit, compare the exact `curl_easy_setopt` sequence, and reapply only after Chrome and Firefox integration checks match.
- Do not publish a release from the changed workflow until a controlled native run proves bundle production and release-job artifact access.

## Completion Checklist

- [ ] Public `client` and `impersonate` APIs are unchanged.
- [ ] Request validation errors, timeout rounding, option order, response behavior, and native-unavailable behavior are unchanged.
- [ ] Symbolic option plans, unused body-reader/handle-pool groundwork, duplicate stubs, runtime-loader implementation, unreachable callback state, and duplicate `.gitignore` content are removed.
- [ ] Native build, verification, packaging, and artifact upload have one workflow implementation.
- [ ] Default tests, race tests, vet, formatting, shell syntax, and diff checks pass.
- [ ] Chrome and Firefox native integration and fingerprint checks pass.
- [ ] External-module smoke testing and native bundle packaging pass.
- [ ] A controlled workflow run verifies one native pipeline execution and release artifact handoff.
- [ ] Documentation contains no stale references to removed code or checks.
- [ ] The working tree contains only intended simplification changes and the implementation plan is updated with completion evidence.
