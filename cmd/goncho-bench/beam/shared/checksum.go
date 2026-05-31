package shared

import "github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/checksum"

func ChecksumBytesSHA256(raw []byte) string {
	return checksum.SHA256Bytes(raw)
}
