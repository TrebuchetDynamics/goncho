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
