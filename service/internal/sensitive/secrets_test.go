package sensitive

import "testing"

func TestContainsSecretLikeContent(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{name: "api token", in: "Remember: API token is sk-live-secret.", want: true},
		{name: "bearer", in: "Authorization: Bearer abc123", want: true},
		{name: "private key", in: "store private key elsewhere", want: true},
		{name: "ordinary preference", in: "prefers concise release notes", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ContainsSecretLikeContent(tt.in); got != tt.want {
				t.Fatalf("ContainsSecretLikeContent(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestMetadataKeySecretLike(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{name: "trimmed token", in: " api_token ", want: true},
		{name: "authorization", in: "Authorization", want: true},
		{name: "private key", in: "ssh_private_key", want: true},
		{name: "normal key", in: "session_id", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MetadataKeySecretLike(tt.in); got != tt.want {
				t.Fatalf("MetadataKeySecretLike(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
