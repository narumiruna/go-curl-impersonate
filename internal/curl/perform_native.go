//go:build integration && native && cgo

package curl

/*
#include <stdlib.h>
#include <curl/curl.h>

extern size_t goCurlWriteCallback(char *ptr, size_t size, size_t nmemb, void *userdata);
extern size_t goCurlHeaderCallback(char *ptr, size_t size, size_t nmemb, void *userdata);

extern CURLcode curl_easy_impersonate(CURL *data, const char *target, int default_headers);

static size_t gci_write_callback(char *ptr, size_t size, size_t nmemb, void *userdata) {
	return goCurlWriteCallback(ptr, size, nmemb, userdata);
}

static size_t gci_header_callback(char *ptr, size_t size, size_t nmemb, void *userdata) {
	return goCurlHeaderCallback(ptr, size, nmemb, userdata);
}

static CURLcode gci_impersonate(CURL *h, char *target, int default_headers) {
	return curl_easy_impersonate(h, target, default_headers);
}

static CURLcode gci_set_url(CURL *h, char *value) {
	return curl_easy_setopt(h, CURLOPT_URL, value);
}

static CURLcode gci_set_customrequest(CURL *h, char *value) {
	return curl_easy_setopt(h, CURLOPT_CUSTOMREQUEST, value);
}

static CURLcode gci_set_httpheader(CURL *h, struct curl_slist *headers) {
	return curl_easy_setopt(h, CURLOPT_HTTPHEADER, headers);
}

static CURLcode gci_set_postfieldsize(CURL *h, curl_off_t value) {
	return curl_easy_setopt(h, CURLOPT_POSTFIELDSIZE_LARGE, value);
}

static CURLcode gci_set_copy_postfields(CURL *h, void *value) {
	return curl_easy_setopt(h, CURLOPT_COPYPOSTFIELDS, value);
}

static CURLcode gci_set_timeout_ms(CURL *h, long value) {
	return curl_easy_setopt(h, CURLOPT_TIMEOUT_MS, value);
}

static CURLcode gci_set_proxy(CURL *h, char *value) {
	return curl_easy_setopt(h, CURLOPT_PROXY, value);
}

static CURLcode gci_set_followlocation(CURL *h, long value) {
	return curl_easy_setopt(h, CURLOPT_FOLLOWLOCATION, value);
}

static CURLcode gci_set_maxredirs(CURL *h, long value) {
	return curl_easy_setopt(h, CURLOPT_MAXREDIRS, value);
}

static CURLcode gci_set_ssl_verifypeer(CURL *h, long value) {
	return curl_easy_setopt(h, CURLOPT_SSL_VERIFYPEER, value);
}

static CURLcode gci_set_ssl_verifyhost(CURL *h, long value) {
	return curl_easy_setopt(h, CURLOPT_SSL_VERIFYHOST, value);
}

static CURLcode gci_set_http_version(CURL *h, long value) {
	return curl_easy_setopt(h, CURLOPT_HTTP_VERSION, value);
}

static CURLcode gci_set_nosignal(CURL *h, long value) {
	return curl_easy_setopt(h, CURLOPT_NOSIGNAL, value);
}

static CURLcode gci_set_writedata(CURL *h, void *value) {
	return curl_easy_setopt(h, CURLOPT_WRITEDATA, value);
}

static CURLcode gci_set_writefunction(CURL *h) {
	return curl_easy_setopt(h, CURLOPT_WRITEFUNCTION, gci_write_callback);
}

static CURLcode gci_set_headerdata(CURL *h, void *value) {
	return curl_easy_setopt(h, CURLOPT_HEADERDATA, value);
}

static CURLcode gci_set_headerfunction(CURL *h) {
	return curl_easy_setopt(h, CURLOPT_HEADERFUNCTION, gci_header_callback);
}
*/
import "C"

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"runtime/cgo"
	"sync"
	"unsafe"
)

var (
	curlGlobalOnce sync.Once
	curlGlobalErr  error
)

type nativeTransfer struct {
	collector    ResponseCollector
	headerBuffer bytes.Buffer
	writeErr     error
	headerErr    error
}

func nativeAvailable() bool {
	return true
}

func perform(ctx context.Context, req *http.Request, options Options) (*http.Response, error) {
	if err := initCurlGlobal(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	spec, err := NewRequestSpec(req, options)
	if err != nil {
		return nil, err
	}

	easy := C.curl_easy_init()
	if easy == nil {
		return nil, fmt.Errorf("curl: curl_easy_init returned nil")
	}
	defer C.curl_easy_cleanup(easy)

	nativeCleanup, err := applyNativeOptions(easy, spec.Options)
	if err != nil {
		return nil, err
	}
	defer nativeCleanup()
	requestCleanup, err := applyRequestOptions(easy, spec)
	if err != nil {
		return nil, err
	}
	defer requestCleanup()

	transfer := &nativeTransfer{}
	transferHandle := cgo.NewHandle(transfer)
	defer transferHandle.Delete()
	userdata := unsafe.Pointer(uintptr(transferHandle))

	if err := checkCode("CURLOPT_WRITEDATA", C.gci_set_writedata(easy, userdata)); err != nil {
		return nil, err
	}
	if err := checkCode("CURLOPT_WRITEFUNCTION", C.gci_set_writefunction(easy)); err != nil {
		return nil, err
	}
	if err := checkCode("CURLOPT_HEADERDATA", C.gci_set_headerdata(easy, userdata)); err != nil {
		return nil, err
	}
	if err := checkCode("CURLOPT_HEADERFUNCTION", C.gci_set_headerfunction(easy)); err != nil {
		return nil, err
	}

	code := C.curl_easy_perform(easy)
	if transfer.writeErr != nil {
		return nil, transfer.writeErr
	}
	if transfer.headerErr != nil {
		return nil, transfer.headerErr
	}
	if err := newNativeError(code); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return transfer.collector.Response(req)
}

func initCurlGlobal() error {
	curlGlobalOnce.Do(func() {
		curlGlobalErr = newNativeError(C.curl_global_init(C.CURL_GLOBAL_DEFAULT))
	})
	return curlGlobalErr
}

func applyNativeOptions(easy unsafe.Pointer, options Options) (func(), error) {
	var cleanups []func()
	cleanup := func() {
		for i := len(cleanups) - 1; i >= 0; i-- {
			cleanups[i]()
		}
	}
	plan, err := NewNativePlan(options)
	if err != nil {
		return cleanup, err
	}
	target := C.CString(plan.ImpersonateTarget)
	cleanups = append(cleanups, func() { C.free(unsafe.Pointer(target)) })
	defaultHeaders := C.int(0)
	if plan.DefaultHeaders {
		defaultHeaders = 1
	}
	if err := checkCode("curl_easy_impersonate", C.gci_impersonate(easy, target, defaultHeaders)); err != nil {
		cleanup()
		return func() {}, err
	}
	if err := checkCode("CURLOPT_NOSIGNAL", C.gci_set_nosignal(easy, 1)); err != nil {
		cleanup()
		return func() {}, err
	}
	if plan.TimeoutMillis > 0 {
		if err := checkCode("CURLOPT_TIMEOUT_MS", C.gci_set_timeout_ms(easy, C.long(plan.TimeoutMillis))); err != nil {
			cleanup()
			return func() {}, err
		}
	}
	if plan.Proxy != "" {
		proxy := C.CString(plan.Proxy)
		cleanups = append(cleanups, func() { C.free(unsafe.Pointer(proxy)) })
		if err := checkCode("CURLOPT_PROXY", C.gci_set_proxy(easy, proxy)); err != nil {
			cleanup()
			return func() {}, err
		}
	}
	if err := checkCode("CURLOPT_FOLLOWLOCATION", C.gci_set_followlocation(easy, boolLong(plan.FollowRedirect))); err != nil {
		cleanup()
		return func() {}, err
	}
	if plan.FollowRedirect && plan.MaxRedirects > 0 {
		if err := checkCode("CURLOPT_MAXREDIRS", C.gci_set_maxredirs(easy, C.long(plan.MaxRedirects))); err != nil {
			cleanup()
			return func() {}, err
		}
	}
	if err := checkCode("CURLOPT_SSL_VERIFYPEER", C.gci_set_ssl_verifypeer(easy, boolLong(plan.TLSVerify))); err != nil {
		cleanup()
		return func() {}, err
	}
	verifyHost := C.long(0)
	if plan.TLSVerify {
		verifyHost = 2
	}
	if err := checkCode("CURLOPT_SSL_VERIFYHOST", C.gci_set_ssl_verifyhost(easy, verifyHost)); err != nil {
		cleanup()
		return func() {}, err
	}
	if plan.HTTP2 {
		if err := checkCode("CURLOPT_HTTP_VERSION", C.gci_set_http_version(easy, C.CURL_HTTP_VERSION_2TLS)); err != nil {
			cleanup()
			return func() {}, err
		}
	}
	return cleanup, nil
}

func applyRequestOptions(easy unsafe.Pointer, spec RequestSpec) (func(), error) {
	var cleanups []func()
	cleanup := func() {
		for i := len(cleanups) - 1; i >= 0; i-- {
			cleanups[i]()
		}
	}
	url := C.CString(spec.URL)
	cleanups = append(cleanups, func() { C.free(unsafe.Pointer(url)) })
	if err := checkCode("CURLOPT_URL", C.gci_set_url(easy, url)); err != nil {
		cleanup()
		return func() {}, err
	}

	method := C.CString(spec.Method)
	cleanups = append(cleanups, func() { C.free(unsafe.Pointer(method)) })
	if err := checkCode("CURLOPT_CUSTOMREQUEST", C.gci_set_customrequest(easy, method)); err != nil {
		cleanup()
		return func() {}, err
	}

	var headerList *C.struct_curl_slist
	cleanups = append(cleanups, func() {
		if headerList != nil {
			C.curl_slist_free_all(headerList)
		}
	})
	for _, line := range spec.HeaderLines() {
		value := C.CString(line)
		next := C.curl_slist_append(headerList, value)
		C.free(unsafe.Pointer(value))
		if next == nil {
			cleanup()
			return func() {}, fmt.Errorf("curl: curl_slist_append failed")
		}
		headerList = next
	}
	if headerList != nil {
		if err := checkCode("CURLOPT_HTTPHEADER", C.gci_set_httpheader(easy, headerList)); err != nil {
			cleanup()
			return func() {}, err
		}
	}

	if len(spec.Body) > 0 {
		body := C.CBytes(spec.Body)
		cleanups = append(cleanups, func() { C.free(body) })
		if err := checkCode("CURLOPT_POSTFIELDSIZE_LARGE", C.gci_set_postfieldsize(easy, C.curl_off_t(len(spec.Body)))); err != nil {
			cleanup()
			return func() {}, err
		}
		if err := checkCode("CURLOPT_COPYPOSTFIELDS", C.gci_set_copy_postfields(easy, body)); err != nil {
			cleanup()
			return func() {}, err
		}
	}
	return cleanup, nil
}

func boolLong(value bool) C.long {
	if value {
		return 1
	}
	return 0
}

func checkCode(operation string, code C.CURLcode) error {
	if err := newNativeError(code); err != nil {
		return fmt.Errorf("%s: %w", operation, err)
	}
	return nil
}

func newNativeError(code C.CURLcode) error {
	if code == C.CURLE_OK {
		return nil
	}
	return NewError(int(code), C.GoString(C.curl_easy_strerror(code)))
}

//export goCurlWriteCallback
func goCurlWriteCallback(ptr *C.char, size C.size_t, nmemb C.size_t, userdata unsafe.Pointer) C.size_t {
	total := size * nmemb
	transfer := cgo.Handle(uintptr(userdata)).Value().(*nativeTransfer)
	if total == 0 {
		return 0
	}
	chunk := C.GoBytes(unsafe.Pointer(ptr), C.int(total))
	transfer.collector.AppendBody(chunk)
	return total
}

//export goCurlHeaderCallback
func goCurlHeaderCallback(ptr *C.char, size C.size_t, nmemb C.size_t, userdata unsafe.Pointer) C.size_t {
	total := size * nmemb
	transfer := cgo.Handle(uintptr(userdata)).Value().(*nativeTransfer)
	if total == 0 {
		return 0
	}
	line := C.GoBytes(unsafe.Pointer(ptr), C.int(total))
	if bytes.Equal(line, []byte("\r\n")) || bytes.Equal(line, []byte("\n")) {
		if transfer.headerBuffer.Len() > 0 {
			if err := transfer.collector.AddHeaderBlock(transfer.headerBuffer.String()); err != nil {
				transfer.headerErr = err
				return 0
			}
			transfer.headerBuffer.Reset()
		}
		return total
	}
	if _, err := transfer.headerBuffer.Write(line); err != nil {
		transfer.headerErr = err
		return 0
	}
	return total
}
