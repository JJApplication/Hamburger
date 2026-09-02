package stat

import (
	"testing"
	"time"
)

func TestNormalizeDomain(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "absolute https host and port", input: "HTTPS://Example.COM:443/path?q=1", want: "example.com"},
		{name: "host trailing dot", input: "Example.COM.", want: "example.com"},
		{name: "unicode idna", input: "https://BÜCHER.Example:443/", want: "xn--bcher-kva.example"},
		{name: "ipv4 port", input: "192.0.2.10:8080", want: "192.0.2.10"},
		{name: "ipv6 port", input: "[2001:db8::1]:443", want: "2001:db8::1"},
		{name: "scheme relative", input: "//EXAMPLE.COM:8443/api", want: "example.com"},
		{name: "empty", input: "   ", want: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := NormalizeDomain(test.input); got != test.want {
				t.Fatalf("NormalizeDomain(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestParseRange(t *testing.T) {
	tests := []struct {
		value       string
		durationSec int64
		widthSec    int64
	}{
		{value: "", durationSec: 3600, widthSec: 60},
		{value: "1h", durationSec: 3600, widthSec: 60},
		{value: "5h", durationSec: 5 * 3600, widthSec: 5 * 60},
		{value: "24h", durationSec: 24 * 3600, widthSec: 15 * 60},
		{value: "7d", durationSec: 7 * 24 * 3600, widthSec: 3600},
		{value: "30d", durationSec: 30 * 24 * 3600, widthSec: 2 * 3600},
	}
	for _, test := range tests {
		spec, err := ParseRange(test.value)
		if err != nil {
			t.Fatalf("ParseRange(%q) returned error: %v", test.value, err)
		}
		if int64(spec.Duration/time.Second) != test.durationSec || int64(spec.BucketWidth/time.Second) != test.widthSec {
			t.Fatalf("ParseRange(%q) = duration %s width %s", test.value, spec.Duration, spec.BucketWidth)
		}
	}
	if _, err := ParseRange("2h"); err != ErrInvalidRange {
		t.Fatalf("ParseRange(2h) error = %v, want ErrInvalidRange", err)
	}
}

func TestLatencyHistogramAndResampling(t *testing.T) {
	var histogram [requestHistogramSize]int64
	histogram[latencyHistogramIndex(10_000)] = 90
	histogram[latencyHistogramIndex(100_000)] = 10
	if got := approximateP95(histogram); got != 100 {
		t.Fatalf("approximateP95() = %vms, want 100ms", got)
	}
	if got := outputBucketStart(359, 5*time.Minute); got != 300 {
		t.Fatalf("outputBucketStart() = %d, want 300", got)
	}
	if got := outputBucketStart(301, 5*time.Minute); got != 300 {
		t.Fatalf("outputBucketStart() = %d, want 300", got)
	}
}
