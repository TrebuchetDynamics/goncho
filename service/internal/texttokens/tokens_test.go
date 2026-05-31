package texttokens

import (
	"reflect"
	"testing"
)

func TestLowerAlnumLowercasesAndSplitsPunctuation(t *testing.T) {
	got := LowerAlnum("Docker-cache v2. API_KEY!")
	want := []string{"docker", "cache", "v2", "api", "key"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LowerAlnum() = %#v, want %#v", got, want)
	}
}

func TestLowerAlnumEmptyWhenNoTokens(t *testing.T) {
	if got := LowerAlnum("!? --"); len(got) != 0 {
		t.Fatalf("LowerAlnum() = %#v, want empty", got)
	}
}
