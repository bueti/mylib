package enrich

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJaroWinkler_Identical(t *testing.T) {
	assert.Equal(t, 1.0, jaroWinkler("Fantasy", "Fantasy"))
	assert.Equal(t, 1.0, jaroWinkler("", ""))
}

func TestJaroWinkler_Similar(t *testing.T) {
	cases := []struct{ a, b string }{
		{"Classics", "Classic"},
		{"Science Fiction", "Science-Fiction"},
		{"Historical Fiction", "Historical fiction"},
	}
	for _, tc := range cases {
		score := jaroWinkler(tc.a, tc.b)
		assert.Greater(t, score, 0.85, "%q vs %q = %.3f", tc.a, tc.b, score)
	}
}

func TestJaroWinkler_Different(t *testing.T) {
	cases := []struct{ a, b string }{
		{"Fantasy", "Programming"},
		{"Horror", "Romance"},
		{"Science Fiction", "Biography"},
		{"History", "Mystery"},
	}
	for _, tc := range cases {
		score := jaroWinkler(tc.a, tc.b)
		assert.Less(t, score, 0.80, "%q vs %q = %.3f", tc.a, tc.b, score)
	}
}

func TestContainsTag(t *testing.T) {
	shorter, _, ok := containsTag("Classics", "FICTION / Classics")
	assert.True(t, ok)
	assert.Equal(t, "Classics", shorter)

	shorter, _, ok = containsTag("COMPUTERS / Programming Languages / General", "Programming")
	assert.True(t, ok)
	assert.Equal(t, "Programming", shorter)

	_, _, ok = containsTag("Fantasy", "Horror")
	assert.False(t, ok)

	// Same length tokens — not containment.
	_, _, ok = containsTag("Fantasy", "Romance")
	assert.False(t, ok)
}

func TestTokenSetRatio(t *testing.T) {
	// "Programming" tokens are a subset of the longer tag.
	ratio := tokenSetRatio("Programming", "COMPUTERS / Programming Languages / General")
	assert.Greater(t, ratio, 0.79, "got %.3f", ratio)

	// Unrelated tags.
	ratio = tokenSetRatio("Fantasy", "Programming")
	assert.Less(t, ratio, 0.5, "got %.3f", ratio)
}

func TestMergeSimilarTags_ThreePass(t *testing.T) {
	tags := []TagWithCount{
		{Name: "Science Fiction", Count: 10},
		{Name: "Science-Fiction", Count: 3},
		{Name: "Fantasy", Count: 8},
		{Name: "Horror", Count: 5},
		{Name: "Horror Stories", Count: 2},
		{Name: "Programming", Count: 6},
		{Name: "COMPUTERS / Programming Languages / General", Count: 2},
		{Name: "Romance", Count: 4},
		{Name: "FICTION / Classics", Count: 2},
		{Name: "Classics", Count: 8},
	}

	merges := MergeSimilarTags(tags, 0.90)

	// Containment: "FICTION / Classics" → "Classics".
	if canon, ok := merges["FICTION / Classics"]; ok {
		assert.Equal(t, "Classics", canon)
	}

	// Containment: long programming → "Programming".
	if canon, ok := merges["COMPUTERS / Programming Languages / General"]; ok {
		assert.Equal(t, "Programming", canon)
	}

	// Jaro-Winkler: "Science-Fiction" → "Science Fiction".
	if canon, ok := merges["Science-Fiction"]; ok {
		assert.Equal(t, "Science Fiction", canon)
	}

	// Fantasy, Romance should NOT be merged.
	assert.NotContains(t, merges, "Fantasy")
	assert.NotContains(t, merges, "Romance")

	require.LessOrEqual(t, len(merges), 6)
}

func TestMergeSimilarTags_Empty(t *testing.T) {
	merges := MergeSimilarTags(nil, 0)
	assert.Empty(t, merges)
}

func TestLooksLikePersonName(t *testing.T) {
	assert.True(t, looksLikePersonName("Benjamin Buetikofer"))
	assert.True(t, looksLikePersonName("John Smith"))
	assert.False(t, looksLikePersonName("Science Fiction"))
	assert.False(t, looksLikePersonName("Historical Fiction"))
	assert.False(t, looksLikePersonName("Computer Programming"))
	assert.False(t, looksLikePersonName("hello")) // too few words
}
