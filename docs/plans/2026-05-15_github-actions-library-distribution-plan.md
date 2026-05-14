## Goal

把這個 repo 推進成 Go 版本的 `curl_cffi`：Go API 直接封裝
`curl-impersonate`，使用者可以像使用一般 Go library 一樣導入並發送
browser-impersonated requests，而 contributor / CI 能在 GitHub Actions 上可重現地
建置與驗證 native backend。成功條件是：

- Default CI 在沒有 native dependency 的 plain checkout 上穩定通過。
- Native CI 能從 pinned upstream source 建出 Chrome/Firefox
  `curl-impersonate` artifacts，跑過 Go native integration tests 與
  TLS/HTTP2 fingerprint checks。
- 使用者能用清楚的 documented path 在外部 Go module 中 `go get` /
  import `github.com/narumiruna/go-curl-impersonate/client`，並透過第一版
  native artifact distribution strategy 送出 impersonated request。
- Plan 明確區分 contributor/CI source pinning 與 consumer native artifact
  distribution，避免把 git submodule 誤當成 `go get` 使用者的安裝方案。

## Context

目前 default CI 只跑 no-native Go tests。Native backend 已能在本機透過
`/tmp/curl-impersonate-local`、`PKG_CONFIG_PATH`、`LD_LIBRARY_PATH` 驗證，
但 GitHub Actions 還沒有負責建 upstream `curl-impersonate` artifacts。

目前 reference source layout 已收斂為：

- `third_party/curl-impersonate`：pinned upstream source / submodule，供
  contributor 與 GitHub Actions build native artifacts、讀取 fingerprint
  fixtures、做 profile parity。
- `references/curl_cffi`：optional local-only design reference，供 API 與行為
  設計比對；不是 CI build 必需品。

`curl-impersonate` 是 native build、fingerprint fixtures、profile parity 的
必要 source。`curl_cffi` 目前主要是設計參考，不是 CI build 必需品。

Git submodule 可以讓 repo checkout 和 GitHub Actions 取得 pinned upstream
source，但 Go module 消費者透過 `go get` 取得的是 module source archive；這條路徑
不會替使用者初始化 repo submodules，也不會在 `go install` 後執行 post-install
hook 來 build 或安裝 native libraries。因此 submodule 是 contributor/CI 的解法，
不是 consumer distribution 的完整解法。

## Architecture

- GitHub Actions/reference source：使用 git submodule pin
  `third_party/curl-impersonate`，讓 CI 可以用固定 commit build native artifacts
  與讀取 fixture。不要把整個 source parent 從 `.gitignore` 拿掉；只允許 submodule
  gitlink 與必要 metadata，避免誤 commit upstream build output。
- Go library consumers：使用者端不能只靠 submodule。需要選擇一條 native
  artifact distribution strategy，讓外部 module 可以穩定取得 headers/libs 或在
  runtime 載入 bundled native artifacts。
- Native distribution candidates:
  - Phase 1：release native bundle，內含 Chrome/Firefox shared libraries、
    headers、pkg-config metadata、checksums，以及可選 CLI binary。Library 使用者
    下載 bundle 後設定 `PKG_CONFIG_PATH` / `LD_LIBRARY_PATH` 或 source helper env
    file。
  - Phase 2 candidate：platform-specific Go native artifact module，例如
    `go-curl-impersonate-native-linux-amd64`，讓 `go get` 可下載 pinned headers/libs
    並用 `#cgo CFLAGS/LDFLAGS` 指向 module 內 artifacts。需要先驗證 module size、
    Go proxy behavior、license、static/dynamic linking 與 runtime loader path。
  - Phase 2 candidate：runtime loader / embedded native bundle，接近 Python wheel
    體驗。這可能避免 compile-time pkg-config，但 libcurl callbacks、dynamic loading、
    extraction path、安全更新與 cross-platform behavior 都需要專門設計。

## Tech Stack

- GitHub Actions `ubuntu-latest` / Linux amd64 first。
- `actions/checkout` with `submodules: true` for native workflow。
- `actions/setup-go` with `go-version-file: go.mod`。
- apt packages from README/build docs, plus `python3-yaml` and
  `nghttp2-server` for fingerprint verification。
- Upstream `curl-impersonate` build installed into a CI-local prefix such as
  `${RUNNER_TEMP}/curl-impersonate-local`。

## Non-Goals

- 不在第一階段承諾 macOS、Windows、arm64 native bundles。
- 不把 full upstream source 或 build output vendored into Go module。
- 不在第一階段承諾所有平台都能 `go get` 後零設定使用；第一階段先讓 Linux
  amd64 有可下載、可驗證、可文件化的 native artifact。後續再評估是否把
  artifact 納入 platform-specific Go modules 或 runtime loader。

## Assumptions

- 第一個 supported consumer target 是 Linux amd64。
- 使用者可以接受 cgo build tag 與 native runtime library。
- Release bundle 比把 native `.so` 直接放進主 Go module 更適合第一版，因為主
  module 應保持輕量、可審查；大型或平台特定 artifacts 需要獨立的 distribution
  boundary。

## Unknowns

- GitHub-hosted runner build upstream Chrome/Firefox artifacts 的耗時是否能接受；
  early task 要記錄 workflow duration，必要時改成 manual/tag-only workflow。
- Release bundle 是否要包含 CLI binary；這不是 library 使用的必要條件，但可以
  當 smoke/debug tool。
- 是否需要 long-term 提供 installer script 下載 latest release bundle 並印出
  env exports；先以文檔與 release artifact 完成第一階段。
- 若要做到接近 `curl_cffi` wheel 的使用體驗，應採用 platform artifact modules、
  runtime loader，或兩者混合；需要 prototype 後才能選定。

## Plan

- [x] 將 upstream `curl-impersonate` 放到 `third_party/curl-impersonate` pinned git
  submodule，並調整 `.gitignore` 只允許 submodule gitlink，不允許
  `third_party/*/build` 或其他 build output；verified with `git submodule status`,
  `git status --ignored third_party`, and `git ls-files third_party .gitmodules`.
- [x] 保留 `references/curl_cffi` 為 ignored local reference，除非後續 CI 真的需要它；
  verified by documenting this decision in `docs/build.md` and confirming
  `git ls-files references/curl_cffi` is empty.
- [ ] 把目前文件中的 native build commands 收斂成
  `scripts/build-curl-impersonate.sh <prefix>`，輸出 headers、Chrome/Firefox
  libraries、pkg-config files 到指定 prefix；verify with
  `PREFIX=/tmp/curl-impersonate-local sh ./scripts/build-curl-impersonate.sh "$PREFIX"`.
- [ ] 更新 `scripts/check-native.sh` 讓它可以直接接受 `PREFIX` 或
  `PKG_CONFIG_PATH`，並維持目前 artifact/header/symbol/native Go tests 的檢查；
  verify with clean `env -i ... sh ./scripts/check-native.sh`.
- [ ] 新增 `.github/workflows/native.yml`，在 `workflow_dispatch`、tag push，或
  `main` path changes 時 checkout submodules、install apt deps、build native
  artifacts、run `scripts/check-native.sh`、run Chrome/Firefox
  `scripts/check-fingerprint.py`; verify with a successful GitHub Actions run URL
  or `gh run view` evidence.
- [ ] 保持 `.github/workflows/test.yml` 為 no-native fast path，plain checkout 不需要
  submodule；verify with `go test ./...`, `go test -tags=integration ./...`,
  `go test -race ./...`, and successful default workflow on GitHub.
- [ ] 寫一份 consumer distribution decision record 到 `docs/native-distribution.md`，
  比較 release bundle、platform-specific Go artifact module、runtime loader /
  embedded bundle、system pkg-config 四種方案；verify with a recommendation for
  Phase 1 and explicit follow-up criteria for `curl_cffi`-like zero-setup usage.
- [ ] 新增 release workflow 或 release job，從 native prefix 打包
  `go-curl-impersonate-native-linux-amd64.tar.gz`，內容至少包含 `lib/`,
  `include/`, `lib/pkgconfig/`, `VERSION`, `SHA256SUMS`，可選 `bin/`
  diagnostic CLI；verify by unpacking the artifact in a clean temp dir and running
  `PKG_CONFIG_PATH=<unpack>/lib/pkgconfig LD_LIBRARY_PATH=<unpack>/lib sh ./scripts/check-native.sh`.
- [ ] 補 `docs/quickstart.md` 或 README quickstart，明確分成 library path 與 CLI
  path：library path 使用 `go get`, import `client`, download native bundle,
  set env, run with `-tags="integration native"`；CLI path 使用 release binary or
  `go install` with native deps；verify docs examples by executing them from a
  temporary external module.
- [ ] 新增外部 consumer smoke script，例如 `scripts/smoke-external-module.sh`，
  建一個 temp Go module、`go get github.com/narumiruna/go-curl-impersonate`,
  寫入最小 `client.NewClient` example，使用 release/native prefix 送出 local
  httptest 或 ATP smoke request；verify with the script passing locally and in
  native GitHub Actions.
- [ ] Prototype one `curl_cffi`-like consumer path beyond manual env setup:
  either a platform-specific artifact module or runtime loader experiment; verify
  by running an external temp module with only `go get` plus documented minimal
  setup for Linux amd64.
- [ ] 更新 README status wording，避免暗示 `go install` 會自動帶 native libraries；
  verify README clearly says `go get` is for library source, native bundle/pkg-config
  is required for `integration native` builds, and release binary is optional tooling.
- [ ] 若 native workflow duration 太長，將 native CI 設為 manual/tag-only，並在
  default CI 保留 reference fixture skip behavior；verify by recording the chosen
  trigger policy in `docs/build.md`.

## Risks

- Upstream `curl-impersonate` build time may be too long for every push.
- Go module consumers will be confused if docs imply submodules are part of
  `go get`; documentation must separate contributor CI source from consumer
  native artifacts.
- Putting native artifacts into Go modules may hit size, license, update,
  vulnerability scanning, and runtime loader constraints; prototype before
  committing to that path.
- Dynamic library runtime path can be fragile; release bundle needs clear
  `LD_LIBRARY_PATH` guidance or helper env file.
- Shipping native libraries may raise license and security update questions;
  release notes should include upstream commit and rebuild instructions.

## Rollback / Recovery

- If submodule workflow proves too noisy, remove `.gitmodules`, restore
  `third_party/curl-impersonate` to ignored local-only reference, and have native workflow clone
  `curl-impersonate` into `${RUNNER_TEMP}` at a pinned SHA.
- If native build is too slow for GitHub-hosted runners, keep native workflow
  `workflow_dispatch` / tag-only and publish bundles from manual release runs.
- If release bundles are not acceptable, fall back to documented system
  installation plus pkg-config and keep the library API unchanged.

## Completion Checklist

- [ ] A fresh GitHub Actions native run builds Chrome/Firefox native artifacts
  from pinned upstream source and passes `scripts/check-native.sh`, verified by
  workflow URL or `gh run view` output.
- [ ] Chrome and Firefox TLS/HTTP2 fingerprints pass in GitHub Actions with
  `scripts/check-fingerprint.py --profile chrome` and `--profile firefox`,
  verified by workflow logs.
- [ ] Default CI passes without native artifacts or initialized submodules,
  verified by successful `.github/workflows/test.yml` run.
- [ ] A release artifact for Linux amd64 can be unpacked into a clean directory
  and used by `PKG_CONFIG_PATH` / `LD_LIBRARY_PATH` to build and run a sample
  external Go module, verified by `scripts/smoke-external-module.sh`.
- [ ] README or quickstart docs show the library user path with `go get`, import
  example, native bundle setup, build tags, and a successful command output.
- [x] `third_party/curl-impersonate` source handling is intentional and reproducible
  at the repo-file level: committed as a submodule with `.gitmodules`. Native
  workflow logs are covered by the GitHub Actions checklist items above.
- [ ] `docs/native-distribution.md` states why submodule alone does not solve
  consumer installation, and identifies the selected next step toward
  `curl_cffi`-like Go library ergonomics.
