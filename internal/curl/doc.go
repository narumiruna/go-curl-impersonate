// Package curl owns the boundary between Go code and libcurl-impersonate.
//
// The native implementation must keep C-owned state out of public packages.
// Easy handles, slists, C strings, callback buffers, and error buffers should
// be allocated, used, and released inside this package. Callers pass ordinary
// Go request state through Options and receive ordinary Go responses.
//
// Each native request owns a fresh easy handle for its full perform/cleanup
// cycle. Individual easy handles must not be used concurrently.
package curl
