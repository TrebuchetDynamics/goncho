package shared

import (
	"bufio"
	"fmt"
	"strings"
)

// ForEachNonEmptyJSONLLine walks non-blank JSONL lines with BEAM's shared trim-and-scan-error convention.
func ForEachNonEmptyJSONLLine(scanner *bufio.Scanner, scanContext string, handle func(lineNumber int, line string) error) error {
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if err := handle(lineNumber, line); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("%s: %w", scanContext, err)
	}
	return nil
}
