package pathutil

import "testing"

func TestCleanRelativeRejectsEscapesAndEmpty(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
		ok   bool
	}{
		{name: "normal", in: " src/main.go ", want: "src/main.go", ok: true},
		{name: "dot", in: ".", ok: false},
		{name: "parent", in: "../secret.txt", ok: false},
		{name: "parentish prefix preserved unsafe", in: "..secret/file.txt", ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := CleanRelative(tt.in)
			if ok != tt.ok || got != tt.want {
				t.Fatalf("CleanRelative(%q) = %q, %v; want %q, %v", tt.in, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestSlashPathContracts(t *testing.T) {
	if got := CleanSlashPath(" ./docs/../src/main.go "); got != "src/main.go" {
		t.Fatalf("CleanSlashPath() = %q", got)
	}
	if got := NormalizeSlashPattern(" ./docs/**/*.md "); got != "docs/**/*.md" {
		t.Fatalf("NormalizeSlashPattern() = %q", got)
	}
	if got := SlashBase("docs/plan.md"); got != "plan.md" {
		t.Fatalf("SlashBase() = %q", got)
	}
}
