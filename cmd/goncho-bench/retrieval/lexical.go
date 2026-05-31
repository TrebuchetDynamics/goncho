package retrieval

import (
	"hash/fnv"
	"math"
	"regexp"
	"sort"
	"strings"
)

type Record struct {
	ID      string
	Peer    string
	Content string
}

func ContentKey(peer, content string) string {
	return strings.TrimSpace(peer) + "\x1f" + content
}

func StableIDsForContents(peer string, contents []string, contentIDs map[string][]string, limit int) []string {
	out := []string{}
	seen := map[string]struct{}{}
	for _, content := range contents {
		for _, id := range contentIDs[ContentKey(peer, content)] {
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, id)
			if limit > 0 && len(out) >= limit {
				return out
			}
		}
	}
	return out
}

var tokenPattern = regexp.MustCompile(`[a-z0-9]+`)

func RawTokenCount(value string) int {
	return len(tokenPattern.FindAllString(strings.ToLower(value), -1))
}

func Tokens(value string) []string {
	out := []string{}
	for _, token := range tokenPattern.FindAllString(strings.ToLower(value), -1) {
		token = stem(token)
		if len(token) < 3 || stopword(token) {
			continue
		}
		out = append(out, token)
	}
	return out
}

func TokenSet(value string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, token := range Tokens(value) {
		out[token] = struct{}{}
	}
	return out
}

func FTSQuery(query string) string {
	tokens := Tokens(query)
	if len(tokens) == 0 {
		return ""
	}
	return strings.Join(tokens, " OR ")
}

func StableHash(value string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(value))
	return h.Sum64()
}

func RankBM25(query string, items []Record) []Record {
	queryTokens := TokenSet(query)
	if len(queryTokens) == 0 {
		return items
	}
	tfs := make([]map[string]int, len(items))
	lengths := make([]int, len(items))
	df := map[string]int{}
	total := 0
	for i, item := range items {
		tf := termFrequency(item.Content)
		tfs[i] = tf
		for _, count := range tf {
			lengths[i] += count
		}
		total += lengths[i]
		for token := range queryTokens {
			if tf[token] > 0 {
				df[token]++
			}
		}
	}
	avg := 1.0
	if total > 0 {
		avg = float64(total) / float64(len(items))
	}
	type scored struct {
		item  Record
		score float64
		index int
	}
	scoredItems := make([]scored, 0, len(items))
	for i, item := range items {
		scoredItems = append(scoredItems, scored{item: item, score: bm25Score(queryTokens, tfs[i], df, len(items), lengths[i], avg), index: i})
	}
	sort.SliceStable(scoredItems, func(i, j int) bool {
		if scoredItems[i].score == scoredItems[j].score {
			return scoredItems[i].index < scoredItems[j].index
		}
		return scoredItems[i].score > scoredItems[j].score
	})
	out := make([]Record, 0, len(scoredItems))
	for _, item := range scoredItems {
		out = append(out, item.item)
	}
	return out
}

func bm25Score(queryTokens map[string]struct{}, tf map[string]int, df map[string]int, docCount, docLength int, avgLength float64) float64 {
	const k1 = 1.2
	const b = 0.75
	if docCount == 0 || docLength == 0 || avgLength <= 0 {
		return 0
	}
	score := 0.0
	for token := range queryTokens {
		freq := tf[token]
		if freq == 0 {
			continue
		}
		idf := math.Log(1 + (float64(docCount)-float64(df[token])+0.5)/(float64(df[token])+0.5))
		denom := float64(freq) + k1*(1-b+b*(float64(docLength)/avgLength))
		score += idf * (float64(freq) * (k1 + 1) / denom)
	}
	return score
}

func termFrequency(value string) map[string]int {
	out := map[string]int{}
	for _, token := range Tokens(value) {
		out[token]++
	}
	return out
}

func stem(token string) string {
	for _, suffix := range []string{"ing", "edly", "ed", "es", "s"} {
		if len(token) > len(suffix)+3 && strings.HasSuffix(token, suffix) {
			return strings.TrimSuffix(token, suffix)
		}
	}
	return token
}

func stopword(token string) bool {
	switch token {
	case "the", "and", "for", "who", "what", "when", "where", "which", "should", "not", "did", "does", "with", "that", "this", "from", "are", "was", "were", "has", "have", "had", "you", "your", "about", "can", "could", "would", "there", "their", "they", "them", "then", "than":
		return true
	default:
		return false
	}
}
