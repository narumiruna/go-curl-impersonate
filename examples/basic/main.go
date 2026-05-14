package main

import (
	"context"
	"fmt"
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

	if !client.NativeAvailable() {
		fmt.Println("native curl-impersonate backend is not available in this build")
		return
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

	fmt.Println(resp.Status)
}
