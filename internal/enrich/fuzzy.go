package enrich

import (
	"math"
	"strings"
	"unicode"
)

// jaroWinkler returns the Jaro-Winkler similarity between two strings
// (0.0 = no match, 1.0 = identical). This is a standard fuzzy string
// matching algorithm that works well for short strings like tag names.
func jaroWinkler(a, b string) float64 {
	a = strings.ToLower(strings.TrimSpace(a))
	b = strings.ToLower(strings.TrimSpace(b))
	if a == b {
		return 1.0
	}
	if len(a) == 0 || len(b) == 0 {
		return 0.0
	}

	// Jaro distance.
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

	// Winkler modification: boost for common prefix (up to 4 chars).
	prefix := 0
	for i := 0; i < min(4, min(len(a), len(b))); i++ {
		if a[i] != b[i] {
			break
		}
		prefix++
	}
	return jaro + float64(prefix)*0.1*(1.0-jaro)
}

// normalizeForFuzzy strips non-alphanumeric characters and lowercases
// for comparison purposes.
func normalizeForFuzzy(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' {
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}

// MergeSimilarTags takes all tag names with their book counts and
// returns a mapping of tag → canonical tag for tags that should be
// merged. Two tags are considered similar if their Jaro-Winkler
// score >= threshold (default 0.92). The tag with the highest book
// count becomes the canonical name.
func MergeSimilarTags(tags []TagWithCount, threshold float64) map[string]string {
	if threshold <= 0 {
		threshold = 0.92
	}

	type entry struct {
		name       string
		normalized string
		count      int
		canonical  string // empty = self is canonical
	}

	entries := make([]entry, len(tags))
	for i, t := range tags {
		entries[i] = entry{name: t.Name, normalized: normalizeForFuzzy(t.Name), count: t.Count}
	}

	merges := make(map[string]string)

	for i := range entries {
		if entries[i].canonical != "" {
			continue // already merged into something
		}
		for j := i + 1; j < len(entries); j++ {
			if entries[j].canonical != "" {
				continue
			}
			score := jaroWinkler(entries[i].normalized, entries[j].normalized)
			if score < threshold {
				continue
			}
			// Merge the less popular into the more popular.
			if entries[i].count >= entries[j].count {
				merges[entries[j].name] = entries[i].name
				entries[j].canonical = entries[i].name
			} else {
				merges[entries[i].name] = entries[j].name
				entries[i].canonical = entries[j].name
				break // i is now merged, move on
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

// Ensure math import is used (for potential future use).
var _ = math.Abs
