package dbscan

import (
	"fmt"
	"strings"
)

// Bool returns a sql.Scanner-compatible target that accepts common SQLite boolean encodings.
func Bool(target *bool) *BoolScanner { return &BoolScanner{target: target} }

// BoolInt returns SQLite's conventional integer representation for booleans.
func BoolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

type BoolScanner struct{ target *bool }

func (b *BoolScanner) Scan(src any) error {
	switch v := src.(type) {
	case int64:
		*b.target = v != 0
	case int:
		*b.target = v != 0
	case bool:
		*b.target = v
	case []byte:
		*b.target = string(v) == "1" || strings.EqualFold(string(v), "true")
	case string:
		*b.target = v == "1" || strings.EqualFold(v, "true")
	case nil:
		*b.target = false
	default:
		return fmt.Errorf("unsupported bool scan type %T", src)
	}
	return nil
}
