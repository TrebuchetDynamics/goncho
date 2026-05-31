package facttext

import (
	"reflect"
	"testing"
)

func TestSentencesPreservesDecimalMeasurements(t *testing.T) {
	got := Sentences("The cache latency is 1.5 seconds. Runtime uses SQLite.")
	want := []string{"The cache latency is 1.5 seconds.", "Runtime uses SQLite."}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Sentences() = %#v, want %#v", got, want)
	}
}

func TestSentencesTrimsAndKeepsTrailingFragment(t *testing.T) {
	got := Sentences(" Alpha owns Beta!  Gamma uses SQLite")
	want := []string{"Alpha owns Beta!", "Gamma uses SQLite"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Sentences() = %#v, want %#v", got, want)
	}
}
