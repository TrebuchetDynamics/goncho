package timeutil

import (
	"testing"
	"time"
)

func TestUnixUTCReturnsZeroForNonPositiveTimestamps(t *testing.T) {
	if !UnixUTC(0).IsZero() {
		t.Fatal("zero timestamp should remain an unset time")
	}
	if !UnixUTC(-1).IsZero() {
		t.Fatal("negative timestamp should remain an unset time")
	}
}

func TestUnixUTCConvertsSecondsToUTC(t *testing.T) {
	got := UnixUTC(1_700_000_100)
	want := time.Unix(1_700_000_100, 0).UTC()
	if !got.Equal(want) {
		t.Fatalf("UnixUTC() = %v, want %v", got, want)
	}
	if got.Location() != time.UTC {
		t.Fatalf("UnixUTC() location = %v, want UTC", got.Location())
	}
}

func TestUnixNanoUTCReturnsZeroForNonPositiveTimestamps(t *testing.T) {
	if !UnixNanoUTC(0).IsZero() {
		t.Fatal("zero nanosecond timestamp should remain an unset time")
	}
	if !UnixNanoUTC(-1).IsZero() {
		t.Fatal("negative nanosecond timestamp should remain an unset time")
	}
}

func TestUnixNanoUTCConvertsNanosecondsToUTC(t *testing.T) {
	got := UnixNanoUTC(1_700_000_100_123)
	want := time.Unix(0, 1_700_000_100_123).UTC()
	if !got.Equal(want) {
		t.Fatalf("UnixNanoUTC() = %v, want %v", got, want)
	}
	if got.Location() != time.UTC {
		t.Fatalf("UnixNanoUTC() location = %v, want UTC", got.Location())
	}
}
