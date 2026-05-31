package timeutil

import "time"

// UnixUTC converts a Unix timestamp in seconds to UTC. Non-positive values
// return the zero time so callers can preserve optional timestamp semantics.
func UnixUTC(seconds int64) time.Time {
	if seconds <= 0 {
		return time.Time{}
	}
	return time.Unix(seconds, 0).UTC()
}

// UnixNanoUTC converts a Unix timestamp in nanoseconds to UTC. Non-positive
// values return the zero time so callers can preserve optional timestamp
// semantics.
func UnixNanoUTC(nanos int64) time.Time {
	if nanos <= 0 {
		return time.Time{}
	}
	return time.Unix(0, nanos).UTC()
}
