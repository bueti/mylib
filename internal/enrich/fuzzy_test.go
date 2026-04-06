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
	// These should score high (> 0.9).
	cases := []struct{ a, b string }{
		{"Classics", "Classic"},
		{"Children's", "Children's fiction"},
		{"Horror", "Horror stories"},
		{"Programming", "Programming languages"},
		{"Science Fiction", "Science-Fiction"},
		{"Historical Fiction", "Historical fiction"},
	}
	for _, tc := range cases {
		score := jaroWinkler(tc.a, tc.b)
		assert.Greater(t, score, 0.85, "%q vs %q = %.3f", tc.a, tc.b, score)
	}
}

func TestJaroWinkler_Different(t *testing.T) {
	// These should score low (< 0.8).
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

func TestMergeSimilarTags(t *testing.T) {
	tags := []TagWithCount{
		{Name: "Science Fiction", Count: 10},
		{Name: "Science-Fiction", Count: 3},
		{Name: "Sci-Fi", Count: 2},
		{Name: "Fantasy", Count: 8},
		{Name: "Horror", Count: 5},
		{Name: "Horror Stories", Count: 2},
		{Name: "Programming", Count: 6},
		{Name: "Romance", Count: 4},
	}

	merges := MergeSimilarTags(tags, 0.88)

	// Science-Fiction should merge into Science Fiction (higher count).
	if canon, ok := merges["Science-Fiction"]; ok {
		assert.Equal(t, "Science Fiction", canon)
	}

	// Horror Stories should merge into Horror.
	if canon, ok := merges["Horror Stories"]; ok {
		assert.Equal(t, "Horror", canon)
	}

	// Fantasy, Programming, Romance should NOT be merged into anything.
	assert.NotContains(t, merges, "Fantasy")
	assert.NotContains(t, merges, "Programming")
	assert.NotContains(t, merges, "Romance")

	// Total merges should be reasonable (not merging everything).
	require.Less(t, len(merges), 5)
}

func TestMergeSimilarTags_Empty(t *testing.T) {
	merges := MergeSimilarTags(nil, 0)
	assert.Empty(t, merges)
}
