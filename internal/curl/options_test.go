package curl

import (
	"testing"
	"time"
)

func TestValidateOptions(t *testing.T) {
	tests := []struct {
		name    string
		options Options
		wantErr string
	}{
		{
			name:    "valid",
			options: Options{ProfileTarget: "chrome116", Timeout: time.Second, MaxRedirects: 3},
		},
		{
			name:    "missing profile",
			options: Options{},
			wantErr: "curl: profile target is empty",
		},
		{
			name:    "negative timeout",
			options: Options{ProfileTarget: "chrome116", Timeout: -time.Second},
			wantErr: "curl: timeout must not be negative",
		},
		{
			name:    "negative max redirects",
			options: Options{ProfileTarget: "chrome116", MaxRedirects: -1},
			wantErr: "curl: max redirects must not be negative",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateOptions(test.options)
			if test.wantErr == "" {
				if err != nil {
					t.Fatalf("validateOptions returned error: %v", err)
				}
				return
			}
			if err == nil || err.Error() != test.wantErr {
				t.Fatalf("validateOptions error = %v, want %q", err, test.wantErr)
			}
		})
	}
}

func TestDurationMillis(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     int64
	}{
		{name: "zero", duration: 0, want: 0},
		{name: "sub-millisecond", duration: time.Nanosecond, want: 1},
		{name: "milliseconds", duration: 1500 * time.Millisecond, want: 1500},
		{name: "truncate fractional millisecond", duration: 1500 * time.Microsecond, want: 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := durationMillis(test.duration); got != test.want {
				t.Fatalf("durationMillis(%v) = %d, want %d", test.duration, got, test.want)
			}
		})
	}
}
