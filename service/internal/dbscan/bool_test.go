package dbscan

import "testing"

func TestBoolScannerAcceptsSQLiteBoolEncodings(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want bool
	}{
		{name: "int64 one", in: int64(1), want: true},
		{name: "int zero", in: 0, want: false},
		{name: "bool", in: true, want: true},
		{name: "bytes true", in: []byte("true"), want: true},
		{name: "string one", in: "1", want: true},
		{name: "nil", in: nil, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got bool
			if err := Bool(&got).Scan(tc.in); err != nil {
				t.Fatalf("Scan() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("Scan() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestBoolInt(t *testing.T) {
	if BoolInt(true) != 1 {
		t.Fatalf("BoolInt(true) = %d, want 1", BoolInt(true))
	}
	if BoolInt(false) != 0 {
		t.Fatalf("BoolInt(false) = %d, want 0", BoolInt(false))
	}
}
