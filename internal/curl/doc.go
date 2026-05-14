// Package curl owns the boundary between Go code and libcurl-impersonate.
//
// The native implementation must keep C-owned state out of public packages.
// Easy handles, slists, C strings, callback buffers, and error buffers should
// be allocated, used, and released inside this package. Callers pass ordinary
// Go request state through Options and receive ordinary Go responses.
//
// A Client may be shared by goroutines only after the native backend defines a
// handle reuse strategy. Individual easy handles must not be used concurrently.
// If a future handle pool is added, each request must lease one handle for the
// full perform/reset/cleanup cycle.
package curl
