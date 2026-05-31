package signals

import (
	"github.com/TrebuchetDynamics/goncho/service/internal/searchtokens"
	"github.com/TrebuchetDynamics/goncho/service/internal/textutil"
)

func GenericAssistantAnswer(content string) bool {
	return textutil.ContainsAnySubstringFold(content, []string{"as an ai language model", "i cannot provide", "i don't have personal experience"})
}

func PersonalSignalCount(content string) int {
	count := 0
	for _, token := range searchtokens.Tokens(content) {
		switch token {
		case "i", "my", "me", "mine", "myself":
			count++
		}
	}
	return count
}
