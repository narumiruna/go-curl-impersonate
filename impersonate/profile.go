package impersonate

import (
	"fmt"
	"sort"
	"strings"
)

// Profile identifies a curl-impersonate browser target.
type Profile struct {
	Target         string
	DefaultHeaders bool
}

const (
	DefaultChrome        = "chrome116"
	DefaultChromeAndroid = "chrome99_android"
	DefaultFirefox       = "ff117"
)

var supportedTargets = map[string]string{
	"chrome99":         "curl-impersonate-chrome",
	"chrome100":        "curl-impersonate-chrome",
	"chrome101":        "curl-impersonate-chrome",
	"chrome104":        "curl-impersonate-chrome",
	"chrome107":        "curl-impersonate-chrome",
	"chrome110":        "curl-impersonate-chrome",
	"chrome116":        "curl-impersonate-chrome",
	"chrome99_android": "curl-impersonate-chrome",
	"edge99":           "curl-impersonate-chrome",
	"edge101":          "curl-impersonate-chrome",
	"ff91esr":          "curl-impersonate-ff",
	"ff95":             "curl-impersonate-ff",
	"ff98":             "curl-impersonate-ff",
	"ff100":            "curl-impersonate-ff",
	"ff102":            "curl-impersonate-ff",
	"ff109":            "curl-impersonate-ff",
	"ff117":            "curl-impersonate-ff",
	"safari15_3":       "curl-impersonate-chrome",
	"safari15_5":       "curl-impersonate-chrome",
}

var aliases = map[string]string{
	"chrome":         DefaultChrome,
	"chrome_android": DefaultChromeAndroid,
	"firefox":        DefaultFirefox,
	"ff":             DefaultFirefox,
}

// Chrome returns the latest Chrome target supported by the checked-in
// curl-impersonate reference.
func Chrome() Profile {
	return Profile{Target: DefaultChrome, DefaultHeaders: true}
}

// Firefox returns the latest Firefox target supported by the checked-in
// curl-impersonate reference.
func Firefox() Profile {
	return Profile{Target: DefaultFirefox, DefaultHeaders: true}
}

// Resolve turns an alias or native curl-impersonate target into a Profile.
func Resolve(name string) (Profile, error) {
	normalized := strings.ToLower(strings.TrimSpace(name))
	if normalized == "" {
		return Profile{}, fmt.Errorf("impersonate: profile name is empty")
	}
	if target, ok := aliases[normalized]; ok {
		normalized = target
	}
	if _, ok := supportedTargets[normalized]; !ok {
		return Profile{}, fmt.Errorf("impersonate: unsupported profile %q", name)
	}
	return Profile{Target: normalized, DefaultHeaders: true}, nil
}

// Backend returns the native curl-impersonate binary/library family required by
// this profile.
func (p Profile) Backend() (string, error) {
	target := strings.ToLower(strings.TrimSpace(p.Target))
	if backend, ok := supportedTargets[target]; ok {
		return backend, nil
	}
	return "", fmt.Errorf("impersonate: unsupported profile %q", p.Target)
}

// SupportedTargets returns the native curl-impersonate targets known to this
// package.
func SupportedTargets() []string {
	targets := make([]string, 0, len(supportedTargets))
	for target := range supportedTargets {
		targets = append(targets, target)
	}
	sort.Strings(targets)
	return targets
}
