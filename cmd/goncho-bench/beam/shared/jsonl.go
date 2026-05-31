package shared

import (
	"bufio"
	"io"

	jsonlshared "github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/shared/jsonl"
)

// NewJSONLScanner returns a scanner sized for BEAM JSONL artifacts that can contain large prompt/context rows.
func NewJSONLScanner(r io.Reader) *bufio.Scanner {
	return jsonlshared.NewJSONLScanner(r)
}

// ReadJSONLFile decodes a BEAM JSONL file using the shared scan and non-empty-line convention.
func ReadJSONLFile[T any](path, openContext, scanContext, decodeContext string) ([]T, error) {
	return jsonlshared.ReadJSONLFile[T](path, openContext, scanContext, decodeContext)
}

// WriteJSONLRows writes BEAM JSONL rows using the shared one-object-per-line artifact convention.
func WriteJSONLRows[T any](w io.Writer, rows []T, writeContext string) error {
	return jsonlshared.WriteJSONLRows(w, rows, writeContext)
}

// WriteJSONLFileWithParents creates/truncates a parent-creating BEAM JSONL artifact and writes rows to it.
func WriteJSONLFileWithParents[T any](path, mkdirContext, createContext, writeContext string, rows []T) error {
	return jsonlshared.WriteJSONLFileWithParents(path, mkdirContext, createContext, writeContext, rows)
}

// AppendJSONLFileWithParents opens a parent-creating BEAM JSONL artifact for append and writes rows to it.
func AppendJSONLFileWithParents[T any](path, mkdirContext, openContext, writeContext string, rows []T) error {
	return jsonlshared.AppendJSONLFileWithParents(path, mkdirContext, openContext, writeContext, rows)
}

// ForEachNonEmptyJSONLLine walks non-blank JSONL lines with BEAM's shared trim-and-scan-error convention.
func ForEachNonEmptyJSONLLine(scanner *bufio.Scanner, scanContext string, handle func(lineNumber int, line string) error) error {
	return jsonlshared.ForEachNonEmptyJSONLLine(scanner, scanContext, handle)
}
