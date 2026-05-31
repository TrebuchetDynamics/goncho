package artifactio

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// MarshalIndentedJSON marshals a BEAM artifact JSON document with the repository's trailing newline convention.
func MarshalIndentedJSON(value any) ([]byte, error) {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

// ReadFileWithChecksum reads a BEAM artifact and returns its raw bytes plus SHA-256 checksum.
func ReadFileWithChecksum(path, readContext string) ([]byte, string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("%s: %w", readContext, err)
	}
	return raw, ChecksumBytesSHA256(raw), nil
}

// CreateFileWithParents creates/truncates a BEAM artifact file after ensuring its parent directory exists.
func CreateFileWithParents(path, mkdirContext, createContext string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("%s: %w", mkdirContext, err)
	}
	file, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", createContext, err)
	}
	return file, nil
}

// AppendFileWithParents opens a BEAM artifact file for append after ensuring its parent directory exists.
func AppendFileWithParents(path, mkdirContext, openContext string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("%s: %w", mkdirContext, err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", openContext, err)
	}
	return file, nil
}

// WriteFileWithParents writes a BEAM artifact file after ensuring its parent directory exists.
func WriteFileWithParents(path string, raw []byte, mkdirContext, writeContext string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("%s: %w", mkdirContext, err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("%s: %w", writeContext, err)
	}
	return nil
}

// WriteBytesArtifact writes a BEAM byte artifact to stdout when path is "-" or to a parent-creating file otherwise.
func WriteBytesArtifact(path string, raw []byte, mkdirContext, writeContext string) error {
	if path == "-" {
		_, err := os.Stdout.Write(raw)
		return err
	}
	return WriteFileWithParents(path, raw, mkdirContext, writeContext)
}
