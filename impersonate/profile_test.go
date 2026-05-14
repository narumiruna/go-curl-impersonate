package impersonate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestResolveAliases(t *testing.T) {
	tests := map[string]string{
		"chrome":         DefaultChrome,
		" Chrome ":       DefaultChrome,
		"firefox":        DefaultFirefox,
		"ff":             DefaultFirefox,
		"chrome_android": DefaultChromeAndroid,
	}

	for input, want := range tests {
		got, err := Resolve(input)
		if err != nil {
			t.Fatalf("Resolve(%q) returned error: %v", input, err)
		}
		if got.Target != want {
			t.Fatalf("Resolve(%q) target = %q, want %q", input, got.Target, want)
		}
		if !got.DefaultHeaders {
			t.Fatalf("Resolve(%q) should enable default headers", input)
		}
	}
}

func TestResolveNativeTargets(t *testing.T) {
	profile, err := Resolve("chrome116")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if profile.Target != "chrome116" {
		t.Fatalf("target = %q, want chrome116", profile.Target)
	}
	backend, err := profile.Backend()
	if err != nil {
		t.Fatalf("Backend returned error: %v", err)
	}
	if backend != "curl-impersonate-chrome" {
		t.Fatalf("backend = %q, want curl-impersonate-chrome", backend)
	}
}

func TestResolveUnsupported(t *testing.T) {
	if _, err := Resolve("chrome999"); err == nil {
		t.Fatal("Resolve should reject unsupported profiles")
	}
}

func TestSupportedTargetsMatchCurlImpersonateReference(t *testing.T) {
	path := referencePath(t, ".refs", "curl-impersonate", "browsers.json")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) returned error: %v", path, err)
	}
	var reference struct {
		Browsers []struct {
			Name   string `json:"name"`
			Binary string `json:"binary"`
		} `json:"browsers"`
	}
	if err := json.Unmarshal(content, &reference); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}

	want := make(map[string]string)
	for _, browser := range reference.Browsers {
		want[browser.Name] = browser.Binary
	}
	if !reflect.DeepEqual(supportedTargets, want) {
		t.Fatalf("supported targets differ from browsers.json\n got: %#v\nwant: %#v", supportedTargets, want)
	}
}

func referencePath(t *testing.T, parts ...string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Dir(filepath.Dir(file))
	return filepath.Join(append([]string{root}, parts...)...)
}
