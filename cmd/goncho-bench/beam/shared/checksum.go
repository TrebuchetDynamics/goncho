package shared

import "github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/shared/artifactio"

func ChecksumBytesSHA256(raw []byte) string {
	return artifactio.ChecksumBytesSHA256(raw)
}
