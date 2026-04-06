package enrich

import (
	"strings"
	"unicode"
)

// jaroWinkler returns the Jaro-Winkler similarity between two strings
// (0.0 = no match, 1.0 = identical).
func jaroWinkler(a, b string) float64 {
	a = strings.ToLower(strings.TrimSpace(a))
	b = strings.ToLower(strings.TrimSpace(b))
	if a == b {
		return 1.0
	}
	if len(a) == 0 || len(b) == 0 {
		return 0.0
	}

	matchDist := max(len(a), len(b))/2 - 1
	if matchDist < 0 {
		matchDist = 0
	}

	aMatches := make([]bool, len(a))
	bMatches := make([]bool, len(b))

	matches := 0
	transpositions := 0

	for i := range a {
		lo := max(0, i-matchDist)
		hi := min(len(b), i+matchDist+1)
		for j := lo; j < hi; j++ {
			if bMatches[j] || a[i] != b[j] {
				continue
			}
			aMatches[i] = true
			bMatches[j] = true
			matches++
			break
		}
	}
	if matches == 0 {
		return 0.0
	}

	k := 0
	for i := range a {
		if !aMatches[i] {
			continue
		}
		for !bMatches[k] {
			k++
		}
		if a[i] != b[k] {
			transpositions++
		}
		k++
	}

	jaro := (float64(matches)/float64(len(a)) +
		float64(matches)/float64(len(b)) +
		float64(matches-transpositions/2)/float64(matches)) / 3.0

	prefix := 0
	for i := 0; i < min(4, min(len(a), len(b))); i++ {
		if a[i] != b[i] {
			break
		}
		prefix++
	}
	return jaro + float64(prefix)*0.1*(1.0-jaro)
}

// tokenize splits a string into lowercase, deduplicated, non-empty
// word tokens, stripping punctuation.
func tokenize(s string) map[string]struct{} {
	tokens := make(map[string]struct{})
	for _, word := range strings.Fields(strings.ToLower(s)) {
		cleaned := strings.TrimFunc(word, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsDigit(r)
		})
		if cleaned != "" {
			tokens[cleaned] = struct{}{}
		}
	}
	return tokens
}

// tokenSetRatio computes the ratio of shared tokens between a and b.
// Returns intersection/min(len(a), len(b)). High score means one tag
// is essentially a subset of the other.
func tokenSetRatio(a, b string) float64 {
	ta := tokenize(a)
	tb := tokenize(b)
	if len(ta) == 0 || len(tb) == 0 {
		return 0
	}
	shared := 0
	for tok := range ta {
		if _, ok := tb[tok]; ok {
			shared++
		}
	}
	smaller := len(ta)
	if len(tb) < smaller {
		smaller = len(tb)
	}
	return float64(shared) / float64(smaller)
}

// containsTag checks if the longer tag fully contains all tokens of
// the shorter tag. Returns (shorter, true) if so.
func containsTag(a, b string) (shorter string, longer string, ok bool) {
	ta := tokenize(a)
	tb := tokenize(b)
	if len(ta) == 0 || len(tb) == 0 || len(ta) == len(tb) {
		return "", "", false
	}
	// Determine which is shorter.
	small, big := ta, tb
	sName, bName := a, b
	if len(ta) > len(tb) {
		small, big = tb, ta
		sName, bName = b, a
	}
	for tok := range small {
		if _, ok := big[tok]; !ok {
			return "", "", false
		}
	}
	return sName, bName, true
}

// normalizeForFuzzy strips non-alphanumeric characters and lowercases.
func normalizeForFuzzy(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' {
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}

// looksLikePersonName returns true if the string looks like a person name
// (2-4 capitalized words, none matching common genre keywords).
func looksLikePersonName(s string) bool {
	words := strings.Fields(s)
	if len(words) < 2 || len(words) > 4 {
		return false
	}
	caps := 0
	for _, w := range words {
		if len(w) > 0 && unicode.IsUpper(rune(w[0])) {
			caps++
		}
	}
	if caps != len(words) {
		return false
	}
	lower := strings.ToLower(s)
	for _, genre := range []string{
		"fiction", "science", "fantasy", "horror", "mystery",
		"romance", "thriller", "drama", "poetry", "history",
		"philosophy", "psychology", "biography", "adventure",
		"classic", "literary", "short", "stories", "young",
		"action", "political", "historical", "computer",
		"programming", "technology", "self-help", "business",
	} {
		if strings.Contains(lower, genre) {
			return false
		}
	}
	return true
}

// MergeSimilarTags takes all tag names with their book counts and
// returns a mapping of tag → canonical tag for tags that should be
// merged. Uses three strategies in order:
//  1. Containment: if all tokens of tag A appear in tag B, merge the
//     longer into the shorter (more general) one.
//  2. Token-set ratio: if ≥80% of the shorter tag's tokens appear in
//     the longer, merge (catches "COMPUTERS / Programming Languages /
//     General" → "Programming").
//  3. Jaro-Winkler: string similarity ≥ threshold for remaining close
//     variants ("Classics" / "Classic").
//
// In all cases, the tag with the higher book count wins.
func MergeSimilarTags(tags []TagWithCount, jwThreshold float64) map[string]string {
	if jwThreshold <= 0 {
		jwThreshold = 0.92
	}

	type entry struct {
		name       string
		normalized string
		count      int
		merged     bool
	}

	entries := make([]entry, len(tags))
	for i, t := range tags {
		entries[i] = entry{name: t.Name, normalized: normalizeForFuzzy(t.Name), count: t.Count}
	}

	merges := make(map[string]string)

	// Pass 1: containment — "FICTION / Classics" contains "Classics".
	for i := range entries {
		if entries[i].merged {
			continue
		}
		for j := i + 1; j < len(entries); j++ {
			if entries[j].merged {
				continue
			}
			shorter, _, ok := containsTag(entries[i].normalized, entries[j].normalized)
			if !ok {
				continue
			}
			// The canonical is the shorter (more general) tag, but
			// use the name from whichever has higher count among the
			// two that matched.
			var winner, loser int
			if normalizeForFuzzy(entries[i].name) == normalizeForFuzzy(shorter) {
				// i is shorter
				winner, loser = i, j
			} else {
				winner, loser = j, i
			}
			// But if the loser has way more books, keep the loser's name.
			if entries[loser].count > entries[winner].count*2 {
				winner, loser = loser, winner
			}
			merges[entries[loser].name] = entries[winner].name
			entries[loser].merged = true
		}
	}

	// Pass 2: token-set ratio ≥ 0.80 for remaining unmerged tags.
	for i := range entries {
		if entries[i].merged {
			continue
		}
		for j := i + 1; j < len(entries); j++ {
			if entries[j].merged {
				continue
			}
			ratio := tokenSetRatio(entries[i].normalized, entries[j].normalized)
			if ratio < 0.80 {
				continue
			}
			if entries[i].count >= entries[j].count {
				merges[entries[j].name] = entries[i].name
				entries[j].merged = true
			} else {
				merges[entries[i].name] = entries[j].name
				entries[i].merged = true
				break
			}
		}
	}

	// Pass 3: Jaro-Winkler for close string variants.
	for i := range entries {
		if entries[i].merged {
			continue
		}
		for j := i + 1; j < len(entries); j++ {
			if entries[j].merged {
				continue
			}
			score := jaroWinkler(entries[i].normalized, entries[j].normalized)
			if score < jwThreshold {
				continue
			}
			if entries[i].count >= entries[j].count {
				merges[entries[j].name] = entries[i].name
				entries[j].merged = true
			} else {
				merges[entries[i].name] = entries[j].name
				entries[i].merged = true
				break
			}
		}
	}

	return merges
}

// TagWithCount is used by MergeSimilarTags.
type TagWithCount struct {
	Name  string
	Count int
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
