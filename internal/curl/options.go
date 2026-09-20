package curl

import (
	"fmt"
	"time"
)

func validateOptions(options Options) error {
	if options.ProfileTarget == "" {
		return fmt.Errorf("curl: profile target is empty")
	}
	if options.Timeout < 0 {
		return fmt.Errorf("curl: timeout must not be negative")
	}
	if options.MaxRedirects < 0 {
		return fmt.Errorf("curl: max redirects must not be negative")
	}
	return nil
}

func durationMillis(duration time.Duration) int64 {
	if duration == 0 {
		return 0
	}
	millis := duration.Milliseconds()
	if millis == 0 {
		return 1
	}
	return millis
}
