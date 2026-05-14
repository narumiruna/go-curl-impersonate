# Quickstart

## Library Consumer

Install the Go module in your application:

```sh
go get github.com/narumiruna/go-curl-impersonate
```

Unpack the Linux amd64 native bundle and point cgo and the runtime loader at it:

```sh
tar -xzf go-curl-impersonate-native-linux-amd64.tar.gz
export GO_CURL_IMPERSONATE_NATIVE="$PWD/go-curl-impersonate-native-linux-amd64"
export PKG_CONFIG_PATH="$GO_CURL_IMPERSONATE_NATIVE/lib/pkgconfig"
export LD_LIBRARY_PATH="$GO_CURL_IMPERSONATE_NATIVE/lib"
export CGO_CFLAGS="$(pkg-config --cflags libcurl-impersonate-chrome)"
export CGO_LDFLAGS="$(pkg-config --libs libcurl-impersonate-chrome)"
```

Use the client package:

```go
package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/narumiruna/go-curl-impersonate/client"
)

func main() {
	c, err := client.NewClient(
		client.WithProfileName("chrome"),
		client.WithTimeout(20*time.Second),
	)
	if err != nil {
		panic(err)
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://app.atptour.com/api/v2/gateway/livematches/website?scoringTournamentLevel=tour", nil)
	if err != nil {
		panic(err)
	}
	resp, err := c.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	fmt.Println(resp.Status)
	fmt.Println(len(body))
}
```

Run with the native backend tags:

```sh
go run -tags="integration native" .
```

The ATP example should print an HTTP status such as:

```text
200 OK
```

For a local end-to-end consumer smoke test from this repository:

```sh
sh ./scripts/smoke-external-module.sh "$GO_CURL_IMPERSONATE_NATIVE"
```

Expected output:

```text
201 Created consumer smoke ok
```

## CLI / Diagnostic Tool

The command under `cmd/go-curl-impersonate` is for diagnostics. It still needs
the same native library environment:

```sh
go install -tags="integration native" github.com/narumiruna/go-curl-impersonate/cmd/go-curl-impersonate@latest
go-curl-impersonate -profile chrome -url 'https://app.atptour.com/api/v2/gateway/livematches/website?scoringTournamentLevel=tour'
```

`go install` installs the Go binary only. It does not download or install
`curl-impersonate` native libraries.
