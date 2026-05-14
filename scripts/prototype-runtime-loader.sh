#!/usr/bin/env sh
set -eu

repo_path=$(pwd)
module_path=github.com/narumiruna/go-curl-impersonate
tmp_dir=${GO_CURL_IMPERSONATE_RUNTIME_LOADER_DIR:-$(mktemp -d "${TMPDIR:-/tmp}/go-curl-impersonate-runtime-loader.XXXXXX")}
cleanup=0
if [ -z "${GO_CURL_IMPERSONATE_RUNTIME_LOADER_DIR:-}" ]; then
  cleanup=1
fi
trap '[ "$cleanup" -eq 0 ] || rm -rf "$tmp_dir"' EXIT HUP INT TERM

if [ "$#" -gt 1 ]; then
  echo "usage: sh ./scripts/prototype-runtime-loader.sh [PREFIX]" >&2
  exit 2
fi

if [ "$#" -eq 1 ]; then
  GO_CURL_IMPERSONATE_NATIVE=$1
  GO_CURL_IMPERSONATE_NATIVE=$(cd "$GO_CURL_IMPERSONATE_NATIVE" && pwd)
  export GO_CURL_IMPERSONATE_NATIVE
elif [ -n "${GO_CURL_IMPERSONATE_NATIVE:-}" ]; then
  GO_CURL_IMPERSONATE_NATIVE=$(cd "$GO_CURL_IMPERSONATE_NATIVE" && pwd)
  export GO_CURL_IMPERSONATE_NATIVE
else
  echo "missing native prefix; pass PREFIX or set GO_CURL_IMPERSONATE_NATIVE" >&2
  exit 2
fi

mkdir -p "$tmp_dir"
if [ -f "$tmp_dir/go.mod" ]; then
  echo "runtime-loader directory already contains go.mod: $tmp_dir" >&2
  exit 1
fi
cd "$tmp_dir"
go mod init example.com/go-curl-impersonate-runtime-loader
if [ -n "${GO_CURL_IMPERSONATE_MODULE_VERSION:-}" ]; then
  go get "$module_path@${GO_CURL_IMPERSONATE_MODULE_VERSION}"
else
  go mod edit -replace "$module_path=$repo_path"
  go mod edit -require "$module_path@v0.0.0"
fi

cat >main.go <<'EOF'
package main

/*
#cgo linux LDFLAGS: -ldl
#include <dlfcn.h>
#include <stdlib.h>

#define GCI_CURL_GLOBAL_DEFAULT 3L

typedef void CURL;
typedef int CURLcode;
typedef CURLcode (*curl_global_init_fn)(long flags);
typedef void (*curl_global_cleanup_fn)(void);
typedef CURL* (*curl_easy_init_fn)(void);
typedef void (*curl_easy_cleanup_fn)(CURL *);
typedef CURLcode (*curl_easy_impersonate_fn)(CURL *, const char *, int);

static CURLcode call_global_init(void *f, long flags) {
	return ((curl_global_init_fn)f)(flags);
}

static void call_global_cleanup(void *f) {
	((curl_global_cleanup_fn)f)();
}

static CURL *call_easy_init(void *f) {
	return ((curl_easy_init_fn)f)();
}

static void call_easy_cleanup(void *f, CURL *h) {
	((curl_easy_cleanup_fn)f)(h);
}

static CURLcode call_easy_impersonate(void *f, CURL *h, const char *target, int default_headers) {
	return ((curl_easy_impersonate_fn)f)(h, target, default_headers);
}
*/
import "C"

import (
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	"github.com/narumiruna/go-curl-impersonate/impersonate"
)

func main() {
	prefix := os.Getenv("GO_CURL_IMPERSONATE_NATIVE")
	if prefix == "" {
		panic("GO_CURL_IMPERSONATE_NATIVE is empty")
	}
	libPath := filepath.Join(prefix, "lib", "libcurl-impersonate-chrome.so")
	if _, err := os.Stat(libPath); err != nil {
		panic(err)
	}

	handle, err := dlopen(libPath)
	if err != nil {
		panic(err)
	}
	defer C.dlclose(handle)

	globalInit, err := dlsym(handle, "curl_global_init")
	if err != nil {
		panic(err)
	}
	globalCleanup, err := dlsym(handle, "curl_global_cleanup")
	if err != nil {
		panic(err)
	}
	easyInit, err := dlsym(handle, "curl_easy_init")
	if err != nil {
		panic(err)
	}
	easyCleanup, err := dlsym(handle, "curl_easy_cleanup")
	if err != nil {
		panic(err)
	}
	easyImpersonate, err := dlsym(handle, "curl_easy_impersonate")
	if err != nil {
		panic(err)
	}

	if code := C.call_global_init(globalInit, C.GCI_CURL_GLOBAL_DEFAULT); code != 0 {
		panic(fmt.Sprintf("curl_global_init failed: %d", int(code)))
	}
	defer C.call_global_cleanup(globalCleanup)

	easy := C.call_easy_init(easyInit)
	if easy == nil {
		panic("curl_easy_init returned nil")
	}
	defer C.call_easy_cleanup(easyCleanup, easy)

	profile, err := impersonate.Resolve("chrome")
	if err != nil {
		panic(err)
	}
	target := C.CString(profile.Target)
	defer C.free(unsafe.Pointer(target))
	defaultHeaders := C.int(0)
	if profile.DefaultHeaders {
		defaultHeaders = 1
	}
	if code := C.call_easy_impersonate(easyImpersonate, easy, target, defaultHeaders); code != 0 {
		panic(fmt.Sprintf("curl_easy_impersonate failed: %d", int(code)))
	}

	fmt.Printf("runtime loader prototype ok: %s\n", profile.Target)
}

func dlopen(path string) (unsafe.Pointer, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))
	handle := C.dlopen(cPath, C.RTLD_NOW|C.RTLD_LOCAL)
	if handle == nil {
		return nil, fmt.Errorf("dlopen %s: %s", path, dlerror())
	}
	return handle, nil
}

func dlsym(handle unsafe.Pointer, name string) (unsafe.Pointer, error) {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))
	sym := C.dlsym(handle, cName)
	if sym == nil {
		return nil, fmt.Errorf("dlsym %s: %s", name, dlerror())
	}
	return sym, nil
}

func dlerror() string {
	msg := C.dlerror()
	if msg == nil {
		return "unknown dynamic loader error"
	}
	return C.GoString(msg)
}
EOF

go mod tidy
unset CGO_CFLAGS
unset CGO_LDFLAGS
unset LD_LIBRARY_PATH
go run .
