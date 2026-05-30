package jsonutil

import "encoding/json"

// StableIndented returns value encoded with Goncho's stable JSON fixture shape:
// two-space indentation plus a trailing newline.
func StableIndented(value any) ([]byte, error) {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}
