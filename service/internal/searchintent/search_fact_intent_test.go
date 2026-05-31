package searchintent

import "testing"

func TestSequenceAnswerPartsCountsRepeatedMarkers(t *testing.T) {
	subject, steps, ok := SequenceAnswerParts("Deployment order is build, then test, then deploy.")
	if !ok {
		t.Fatal("SequenceAnswerParts did not recognize repeated sequence markers")
	}
	if subject != "Deployment order is build" {
		t.Fatalf("subject = %q, want cleaned prefix before first repeated marker", subject)
	}
	if steps != "then test, then deploy" {
		t.Fatalf("steps = %q, want repeated marker steps", steps)
	}
}

func TestSequenceMarkerCountIgnoresEmbeddedMarkerText(t *testing.T) {
	if got := searchSequenceMarkerCount("context first draft"); got != 1 {
		t.Fatalf("embedded marker count = %d, want 1", got)
	}
}

func TestDecisionFactIntentKeepsCoordinatedDecisionObject(t *testing.T) {
	if score := Score("what did we decide about redis?", "We decided to use Postgres and Redis."); score != 1 {
		t.Fatalf("decision fact score = %v, want 1 for coordinated decision object", score)
	}
}

func TestVersionFactIntentKeepsDottedVersionAfterQuestionSentence(t *testing.T) {
	if score := Score("what version is goncho?", "User asked what shipped? Goncho version is v1.2.3."); score != 1 {
		t.Fatalf("version fact score = %v, want 1 for dotted version after question sentence", score)
	}
}

func TestNegationFactIntentKeepsCoordinatedNegativeObject(t *testing.T) {
	if score := Score("have we ever used kubernetes?", "We never used Docker and Kubernetes."); score != 1 {
		t.Fatalf("negation fact score = %v, want 1 for coordinated negative object", score)
	}
}
