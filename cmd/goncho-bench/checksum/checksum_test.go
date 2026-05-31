package checksum

import "testing"

func TestSHA256BytesReturnsHexDigest(t *testing.T) {
	const want = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if got := SHA256Bytes([]byte("hello")); got != want {
		t.Fatalf("SHA256Bytes = %s, want %s", got, want)
	}
}
