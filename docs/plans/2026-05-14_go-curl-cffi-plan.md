## Goal

建立一個 Go 版本的 `curl_cffi`：以 Go 套件形式封裝 `lwthiker/curl-impersonate`，提供可設定瀏覽器 impersonation profile 的 HTTP client，成功條件是能在本機與 CI 透過 Go API 發送 Chrome/Firefox impersonated requests，並以測試證明 TLS/HTTP2 指紋與基本 HTTP 行為可用。

本文件是實作計畫，不是完成紀錄。未勾選的 task/checklist 代表後續實作工作；此 plan 完成的標準是足以讓下一位 agent 或 engineer 依序落地、驗證與切版發佈。

## Context

本計畫保留從規劃階段到第一版落地的歷史脈絡。參考資料目前分成 pinned upstream source 與 local-only design reference；後續實作先以可落地的 Go/cgo wrapper 為核心，逐步補齊建置、API、測試與發佈流程。

參考實作目前分為：

- `references/curl_cffi`: optional local-only Python `curl_cffi` reference，可參考 `curl_cffi/curl.py` 的 handle lifecycle/callback、`curl_cffi/requests/utils.py` 的 option mapping、`curl_cffi/requests/impersonate.py` 與 `curl_cffi/fingerprints.py` 的 profile/alias/fingerprint 設計。
- `third_party/curl-impersonate`: pinned upstream `curl-impersonate` submodule，可參考 `README.md#libcurl-impersonate`、`browsers.json`、`tests/signatures/*.yaml`、`tests/test_impersonate.py`。

`curl_cffi` 的核心價值不是重新實作 HTTP stack，而是把 `curl-impersonate` 的 libcurl 能力包成易用 API；Go 版本也應避免重新實作 TLS/HTTP2 指紋。`libcurl-impersonate` 提供額外 C API `curl_easy_impersonate(CURL *data, const char *target, int default_headers)`，會設定 HTTP version、TLS、HTTP/2 pseudo-header order、ALPS、signature algorithms、cert compression、extension permutation 等非標準 curl options。

## Architecture

- `internal/curl`: cgo 層，直接包裝 `libcurl` / `curl-impersonate` easy/multi/share APIs 與 `curl_easy_impersonate`，隔離 C 指標、slist、callbacks、error code、cleanup。
- `impersonate`: profile 定義與映射，例如 `chrome`, `chrome116`, `firefox`，負責 alias resolution、default headers flag，以及從 `third_party/curl-impersonate/browsers.json` / `curl_cffi` profile 清單產生或驗證支援矩陣。
- `client`: Go 使用者 API，提供類似 `http.Client` 的 request/response 操作、cookie、headers、proxy、timeout、redirect、HTTP/2 設定。
- `cmd/go-curl-impersonate`（可選）：最小 CLI，用於手動驗證與除錯。
- `testdata` / integration tests：驗證基本 HTTP 行為與指紋輸出。

## Tech Stack

- Go module + cgo。
- `curl-impersonate` 作為外部 native dependency；初期支援 system-installed / pkg-config，後續再評估 vendored prebuilt binaries。
- GitHub Actions 或等效 CI：至少跑 Go unit tests；integration tests 可在具備 native dependency 的 runner 上執行。

## Non-Goals

- 第一版不重寫 TLS、HTTP/2、QUIC 或瀏覽器指紋演算法。
- 第一版不保證完全覆蓋 Python `curl_cffi` 的所有 API。
- 第一版不支援非 cgo 純 Go fallback。
- 第一版不承諾支援所有 OS/CPU；先鎖定 Linux amd64，再擴展 macOS。

## Assumptions

- 可以接受 Go 套件依賴 cgo 與 native `curl-impersonate` runtime library。
- 第一版 API 以 Go idiom 為主，不必 1:1 複製 Python `curl_cffi`。
- 測試環境可安裝或建置 `curl-impersonate`。

## Unknowns

- `curl-impersonate` 在目標平台的安裝、linking、版本偵測方式要採 system package、submodule build，或預編譯 artifact。
- 需要支援哪些 browser profiles 與版本名稱，需對齊 upstream `curl-impersonate` 的實際 options；目前可從 `third_party/curl-impersonate/browsers.json` 與 `references/curl_cffi/curl_cffi/requests/impersonate.py` 比對。
- 指紋驗證要使用哪個外部服務或本地測試工具；若依賴外部服務，CI 可能不穩定。

## Plan

### Phase 0: Scope and Native Dependency Discovery

- [x] 建立 Go module 與最小專案骨架，包含 `go.mod`、`README.md`、`internal/curl`、`impersonate`、`client` 目錄；verified with `go test ./...` passing in the default no-native build.
- [x] 研究 `references/curl_cffi` 與 `third_party/curl-impersonate` 的必要 API surface，整理第一版支援矩陣到 `docs/api-scope.md`；verified with 文件列出 request method、headers、body、cookies、proxy、timeout、redirect、HTTP/2、browser profile、JA3/Akamai/extra fingerprint 的支援/延後狀態。
- [x] 從 `third_party/curl-impersonate/README.md#libcurl-impersonate` 與 header/cdef 來源確認 `curl_easy_impersonate` symbol、library names、non-standard options；verified with `docs/native-api.md` recording the C function signature, option ordering, backend families, and current native-backend gap.
- [x] 決定 native dependency 策略，優先選一條 Linux amd64 可重現路徑，例如 `pkg-config`、明確的 `CGO_CFLAGS` / `CGO_LDFLAGS`，或 repo-local build script；verified with `internal/curl.ProbePkgConfig`, backend-specific `ProbeBackendPkgConfig`, env-based `DetectLinkConfig`, `go run ./cmd/go-curl-impersonate`, artifact- and symbol-validating `scripts/check-native.sh`, and validated `scripts/write-pkg-config.sh` covering pkg-config/env/local-prefix metadata paths. Chrome and Firefox local-prefix build/install were verified under `/tmp/curl-impersonate-local`, including local redirect/proxy/timeout/HTTP2 native tests, and `scripts/check-native.sh` passed under a clean `env -i` environment with only required PATH/HOME/PKG_CONFIG_PATH/LD_LIBRARY_PATH/GOCACHE.

### Phase 1: Low-Level Curl Binding

- [x] 實作 `internal/curl` 的 easy handle lifecycle、option 設定、header/body callback、錯誤碼轉換與 cleanup，參考 `references/curl_cffi/curl_cffi/curl.py`；verified with handle lease lifecycle, full operation plan, request body snapshotting, response collection helpers, CURLcode error conversion, an `integration native` cgo backend, Chrome/Firefox local GET/POST/redirect/proxy/timeout/HTTP2 integration tests, Chrome ATP smoke returning `200 OK`, and Chrome/Firefox TLS/HTTP2 fingerprint checks matching upstream fixtures.
- [x] 定義 cgo build tags 與 no-native fallback 行為，例如 `integration` tag 才 link native library、一般 unit tests 不需要安裝 `curl-impersonate`；verified with default `go test ./...`, placeholder `go test -tags=integration ./...`, and documented separation in `docs/build.md`.
- [x] 加入 memory/lifetime guardrails，包含 C string、slist、callback buffer、easy handle reset/reuse 的 ownership 規則；verified with `internal/curl/doc.go` package contract and `go test -race ./...`.

### Phase 2: Profiles and High-Level Client API

- [x] 實作 impersonation profile API，例如 `impersonate.Chrome()`, `impersonate.Firefox()` 或 `client.WithProfile(...)`，以 `curl_easy_impersonate(handle, target, defaultHeaders)` 為第一路徑；verified with alias/default header tests plus Chrome and Firefox `integration native` local-server requests using real curl-impersonate backends.
- [x] 以 `third_party/curl-impersonate/browsers.json` 作為 native profile source of truth，並把 Python `curl_cffi` 的 latest aliases 當作對照參考；verified with table-driven tests that Chrome/Firefox aliases resolve to supported native targets and a parity test against `third_party/curl-impersonate/browsers.json`.
- [x] 實作 high-level client API，支援 `NewClient`, `Do`, method、URL、headers、body、response status、response headers、response body；verified with `internal/curl.NewRequestSpec` unit tests, response parser tests, Chrome/Firefox `integration native` local-server GET/POST/header/body tests, and Chrome ATP smoke returning `200 OK`.
- [x] 加入 cookie、proxy、redirect、timeout、TLS verify 開關與 HTTP/2 設定；verified with client option/unit tests plus Chrome/Firefox `integration native` tests for cookie jar, proxy routing, redirect following, timeout error mapping, TLS verify disabled for local TLS, and HTTP/2 negotiation.
- [x] 加入並發與 handle reuse 策略，明確定義 client 是否 goroutine-safe、request 是否可並行、handle pool 是否存在；verified with `go test -race ./...`, README concurrency section, and `internal/curl/doc.go`.

### Phase 3: Fingerprint Verification and Release Readiness

- [x] 建立指紋驗證流程，優先重用 `third_party/curl-impersonate/tests/signatures/*.yaml` 與 `tests/test_impersonate.py` 的方法，外部 endpoint 僅作 smoke test；verified with `scripts/check-fingerprint.py` capturing TLS ClientHello locally and parsing `nghttpd -v` HTTP/2 output against upstream YAML fixtures. Chrome TLS/HTTP2 match `chrome_116.0.5845.180_win10`; Firefox TLS/HTTP2 match `firefox_117.0.1_win10`. The verifier can also compare the Go client directly with local upstream wrappers for native binding parity.
- [x] 補齊 README 快速開始、安裝需求、範例程式、限制與 troubleshooting；verified with `README.md` and `examples/basic`, with the documented limitation that the current default build reports native backend unavailable.
- [x] 設定 CI，至少執行格式檢查、`go test ./...`，並把需要 `curl-impersonate` 的 integration tests 標記 build tag；verified with `.github/workflows/test.yml`, local `sh ./scripts/check-fingerprint-fixtures.sh`, `go test ./...`, `go test -tags=integration ./...`, and `go test -race ./...` passing.
- [x] 規劃發佈切片，先發 `v0.1.0` alpha，列出已知限制與不相容變更政策；verified with `CHANGELOG.md` release-scope draft.

## Risks

- cgo 與 native library linking 會提高安裝難度，可能讓使用者體驗比 Python wheel 差。
- upstream `curl-impersonate` profile 名稱、`curl_easy_impersonate` symbol、non-standard option 或 build artifact 變動會造成 API 或 CI 破裂。
- 外部 fingerprint 驗證服務不穩定，可能導致 flaky tests。
- 若 high-level API 過早承諾與 `net/http` 完全相容，後續維護成本會變高。
- Go `net/http` 的 request/response 型別很誘人，但若直接承諾完全相容，可能會掩蓋 libcurl option 與 Go transport model 的差異；第一版應明確定義相容範圍。
- Chrome 與 Firefox 版本可能需要不同 native library 或 build artifact；若第一版 API 把 profile 當成純字串而不檢查 installed backend，錯誤會延後到 request time 才暴露。

## Rollback / Recovery

- 若 high-level API 設計卡住，先保留低階 `curl` wrapper 與 experimental `client` package，避免在 `v0.1.0` 承諾穩定 API。
- 若 native dependency 安裝無法穩定，先把 integration tests 改為手動驗證流程，CI 僅跑不需 native library 的 unit tests。
- 若 profile 驗證不穩定，將外部 fingerprint check 從必跑測試降級為 documented smoke test。

## Completion Checklist

- [x] Go module 與主要 package 已建立，並由 `go test ./...` 驗證通過。
- [x] `curl-impersonate` native dependency 的安裝與 linking 方式已記錄於 `docs/build.md`，並由乾淨環境建置紀錄或 CI job 驗證。
- [x] High-level Go API 可完成 GET/POST、headers、body、cookies、proxy、timeout、redirect 與 HTTP/2 基本情境，並由 tests 驗證。
- [x] 至少 Chrome 與 Firefox impersonation profile 可用，profile 名稱與 alias 已對照 `third_party/curl-impersonate/browsers.json` / `curl_cffi`，並由 `integration native` local-server tests 驗證。
- [x] README 包含安裝、快速開始、範例、限制與 troubleshooting，並由 `examples/basic` 可執行證明；default execution stops with the documented native-unavailable message, and Chrome native execution returned `200 OK` from the ATP endpoint.
- [x] CI 或本機等效命令完成格式檢查、unit tests、race test，以及可用時的 integration tests；verified locally with `go test ./...` and `env GOCACHE=/tmp/go-build go test -race ./...`.
- [x] `v0.1.0` 發佈範圍與已知限制已記錄於 `CHANGELOG.md` 或 release notes draft。
