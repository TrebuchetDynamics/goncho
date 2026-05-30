package hashutil

import "testing"

func TestSHA256HexString(t *testing.T) {
	got := SHA256HexString("goncho")
	want := "e6124c96723a7595765fe6e523f95de32afed2cb3cba8bc792d78764b6b51b24"
	if got != want {
		t.Fatalf("SHA256HexString() = %q, want %q", got, want)
	}
}

func TestSHA256HexStringPrefix(t *testing.T) {
	full := SHA256HexString("goncho")
	prefix := SHA256HexStringPrefix("goncho", 8)
	if prefix != full[:16] {
		t.Fatalf("prefix = %q, want %q", prefix, full[:16])
	}
}

func TestJSONSHA256HexPrefix(t *testing.T) {
	value := struct {
		Name string `json:"name"`
	}{Name: "goncho"}
	full := JSONSHA256Hex(value)
	prefix := JSONSHA256HexPrefix(value, 12)
	if prefix != full[:24] {
		t.Fatalf("prefix = %q, want %q", prefix, full[:24])
	}
}
