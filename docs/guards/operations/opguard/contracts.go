package opguard

import (
	"testing"

	"github.com/TrebuchetDynamics/goncho/docs/guards/guardtest"
)

const InternetExposureWarning = "do not expose directly to the internet"

type FileContract struct {
	Path    string
	Label   string
	Markers []string
	Fold    bool
}

func VerifyFileContract(t *testing.T, contract FileContract) {
	t.Helper()
	text := guardtest.ReadRepoFile(t, contract.Path)
	label := contract.Label
	if label == "" {
		label = contract.Path
	}
	if contract.Fold {
		guardtest.ContainsAllFold(t, text, label, contract.Markers)
		return
	}
	guardtest.ContainsAll(t, text, label, contract.Markers)
}

func VerifyFileContracts(t *testing.T, contracts []FileContract) {
	t.Helper()
	for _, contract := range contracts {
		VerifyFileContract(t, contract)
	}
}
