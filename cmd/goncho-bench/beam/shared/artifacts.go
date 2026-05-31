package shared

import (
	"os"

	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/shared/artifactio"
)

// MarshalIndentedJSON marshals a BEAM artifact JSON document with the repository's trailing newline convention.
func MarshalIndentedJSON(value any) ([]byte, error) {
	return artifactio.MarshalIndentedJSON(value)
}

// ReadFileWithChecksum reads a BEAM artifact and returns its raw bytes plus SHA-256 checksum.
func ReadFileWithChecksum(path, readContext string) ([]byte, string, error) {
	return artifactio.ReadFileWithChecksum(path, readContext)
}

// CreateFileWithParents creates/truncates a BEAM artifact file after ensuring its parent directory exists.
func CreateFileWithParents(path, mkdirContext, createContext string) (*os.File, error) {
	return artifactio.CreateFileWithParents(path, mkdirContext, createContext)
}

// AppendFileWithParents opens a BEAM artifact file for append after ensuring its parent directory exists.
func AppendFileWithParents(path, mkdirContext, openContext string) (*os.File, error) {
	return artifactio.AppendFileWithParents(path, mkdirContext, openContext)
}

// WriteFileWithParents writes a BEAM artifact file after ensuring its parent directory exists.
func WriteFileWithParents(path string, raw []byte, mkdirContext, writeContext string) error {
	return artifactio.WriteFileWithParents(path, raw, mkdirContext, writeContext)
}

// WriteBytesArtifact writes a BEAM byte artifact to stdout when path is "-" or to a parent-creating file otherwise.
func WriteBytesArtifact(path string, raw []byte, mkdirContext, writeContext string) error {
	return artifactio.WriteBytesArtifact(path, raw, mkdirContext, writeContext)
}
