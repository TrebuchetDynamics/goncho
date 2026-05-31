package shared

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

const (
	jsonlInitialScanBuffer = 1024 * 1024
	jsonlMaxScanBuffer     = 16 * 1024 * 1024
)

// NewJSONLScanner returns a scanner sized for BEAM JSONL artifacts that can contain large prompt/context rows.
func NewJSONLScanner(r io.Reader) *bufio.Scanner {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, jsonlInitialScanBuffer), jsonlMaxScanBuffer)
	return scanner
}

// WriteJSONLRows writes BEAM JSONL rows using the shared one-object-per-line artifact convention.
func WriteJSONLRows[T any](w io.Writer, rows []T, writeContext string) error {
	encoder := json.NewEncoder(w)
	for _, row := range rows {
		if err := encoder.Encode(row); err != nil {
			return fmt.Errorf("%s: %w", writeContext, err)
		}
	}
	return nil
}

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
