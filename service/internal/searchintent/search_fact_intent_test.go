package searchintent

import "testing"

func TestSpeakerFactIntentIgnoresNonAttributionSayQuestions(t *testing.T) {
	for _, query := range []string{
		"Did Melanie say the sunrise was orange?",
		"When did Melanie say the sunrise was orange?",
	} {
		if score := Score(query, "speaker Melanie"); score != 0 {
			t.Fatalf("speaker fact score for %q = %v, want 0 for non-attribution question", query, score)
		}
	}
}

func TestOwnerFactIntentIgnoresTemporalAdverbAfterObject(t *testing.T) {
	if score := Score("Who owns component A-17 now?", "Nadia owns component A-17."); score != 1 {
		t.Fatalf("owner fact score = %v, want 1 when question has trailing temporal adverb", score)
	}
}

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

func TestMetricFactIntentKeepsCommaGroupedMetricValue(t *testing.T) {
	if score := Score("how many queue rows?", "Queue rows is 1,024 rows."); score != 1 {
		t.Fatalf("metric fact score = %v, want 1 for comma-grouped metric value", score)
	}
}
