package goncho

import "testing"

func TestConfigEffectiveDefaultsMatchGonchoNamespace(t *testing.T) {
	got := Config{}.Effective()

	if got.WorkspaceID != DefaultWorkspaceID {
		t.Errorf("WorkspaceID = %q, want %q", got.WorkspaceID, DefaultWorkspaceID)
	}
	if got.ObserverPeerID != DefaultObserverPeerID {
		t.Errorf("ObserverPeerID = %q, want %q", got.ObserverPeerID, DefaultObserverPeerID)
	}
	if got.RecentMessages != 4 {
		t.Errorf("RecentMessages = %d, want 4", got.RecentMessages)
	}
	if got.MaxMessageSize != 25_000 {
		t.Errorf("MaxMessageSize = %d, want 25000", got.MaxMessageSize)
	}
	if got.MaxFileSize != 5_242_880 {
		t.Errorf("MaxFileSize = %d, want 5242880", got.MaxFileSize)
	}
	if got.GetContextMaxTokens != 100_000 {
		t.Errorf("GetContextMaxTokens = %d, want 100000", got.GetContextMaxTokens)
	}
	if !got.ReasoningEnabled {
		t.Error("ReasoningEnabled = false, want true")
	}
	if !got.PeerCardEnabled {
		t.Error("PeerCardEnabled = false, want true")
	}
	if !got.SummaryEnabled {
		t.Error("SummaryEnabled = false, want true")
	}
	if got.DreamEnabled {
		t.Error("DreamEnabled = true, want false until fixtures exist")
	}
	if got.DeriverWorkers != 1 {
		t.Errorf("DeriverWorkers = %d, want 1", got.DeriverWorkers)
	}
	if got.RepresentationBatchMaxTokens != 1024 {
		t.Errorf("RepresentationBatchMaxTokens = %d, want 1024", got.RepresentationBatchMaxTokens)
	}
	if got.DialecticDefaultLevel != DialecticLevelLow {
		t.Errorf("DialecticDefaultLevel = %q, want %q", got.DialecticDefaultLevel, DialecticLevelLow)
	}
}

func TestValidDialecticLevel(t *testing.T) {
	for _, level := range []string{"minimal", "low", "medium", "high", "max"} {
		if !ValidDialecticLevel(level) {
			t.Errorf("ValidDialecticLevel(%q) = false, want true", level)
		}
	}
	if ValidDialecticLevel("extreme") {
		t.Error("ValidDialecticLevel(extreme) = true, want false")
	}
}
