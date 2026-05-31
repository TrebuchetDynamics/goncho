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

// WriteJSONLFileWithParents creates/truncates a parent-creating BEAM JSONL artifact and writes rows to it.
func WriteJSONLFileWithParents[T any](path, mkdirContext, createContext, writeContext string, rows []T) error {
	file, err := CreateFileWithParents(path, mkdirContext, createContext)
	if err != nil {
		return err
	}
	defer file.Close()
	return WriteJSONLRows(file, rows, writeContext)
}

// AppendJSONLFileWithParents opens a parent-creating BEAM JSONL artifact for append and writes rows to it.
func AppendJSONLFileWithParents[T any](path, mkdirContext, openContext, writeContext string, rows []T) error {
	file, err := AppendFileWithParents(path, mkdirContext, openContext)
	if err != nil {
		return err
	}
	defer file.Close()
	return WriteJSONLRows(file, rows, writeContext)
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
