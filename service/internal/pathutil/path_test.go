package pathutil

import (
	"path/filepath"
	"testing"
)

func TestAbsNonBlank(t *testing.T) {
	got, ok, err := AbsNonBlank(" . ")
	if err != nil {
		t.Fatalf("AbsNonBlank() error = %v", err)
	}
	if !ok || !filepath.IsAbs(got) {
		t.Fatalf("AbsNonBlank() = %q, %v; want absolute path", got, ok)
	}
	got, ok, err = AbsNonBlank(" \t ")
	if err != nil || ok || got != "" {
		t.Fatalf("AbsNonBlank(blank) = %q, %v, %v; want empty false nil", got, ok, err)
	}
}

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

func TestNormalizeSlashPatterns(t *testing.T) {
	got := NormalizeSlashPatterns([]string{" ./docs/*.md ", "docs/*.md", "", "./src/**"})
	want := []string{"docs/*.md", "src/**"}
	if len(got) != len(want) {
		t.Fatalf("NormalizeSlashPatterns() = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("NormalizeSlashPatterns() = %#v, want %#v", got, want)
		}
	}
}

func TestMatchSlashGlob(t *testing.T) {
	tests := []struct {
		name    string
		rel     string
		pattern string
		want    bool
	}{
		{name: "direct path", rel: "docs/plan.md", pattern: "docs/plan.md", want: true},
		{name: "base", rel: "docs/plan.md", pattern: "plan.md", want: true},
		{name: "wildcard all", rel: "docs/plan.md", pattern: "**", want: true},
		{name: "prefix tree", rel: "docs/sub/plan.md", pattern: "docs/**", want: true},
		{name: "suffix tree", rel: "docs/sub/plan.md", pattern: "**/*.md", want: true},
		{name: "filepath pattern", rel: "docs/plan.md", pattern: "docs/*.md", want: true},
		{name: "reject", rel: "docs/plan.txt", pattern: "docs/*.md", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MatchSlashGlob(tt.rel, tt.pattern); got != tt.want {
				t.Fatalf("MatchSlashGlob(%q, %q) = %v, want %v", tt.rel, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestMatchesAnySlashGlob(t *testing.T) {
	if !MatchesAnySlashGlob("src/main.go", []string{"docs/**", "src/*.go"}) {
		t.Fatalf("MatchesAnySlashGlob should accept matching pattern")
	}
	if MatchesAnySlashGlob("src/main.go", []string{"docs/**", "*.md"}) {
		t.Fatalf("MatchesAnySlashGlob should reject when no pattern matches")
	}
}
