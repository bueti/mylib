package enrich

import (
	"strings"
)

// NormalizeSubjects takes raw Open Library/EPUB subjects and returns
// cleaner, deduplicated genre tags. It:
//  1. Maps known verbose subjects to canonical short names.
//  2. Strips common noise prefixes ("Fiction, ", "Juvenile ").
//  3. Strips trailing qualifiers (", general", ", etc.").
//  4. Filters out non-genre entries (places, languages, curricula).
//  5. Deduplicates case-insensitively.
func NormalizeSubjects(raw []string) []string {
	seen := make(map[string]struct{}, len(raw))
	var out []string
	for _, s := range raw {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		// Apply canonical mapping first.
		if canon, ok := canonicalMap[strings.ToLower(s)]; ok {
			s = canon
		} else {
			s = cleanSubject(s)
		}
		if s == "" {
			continue
		}
		if isNoise(s) {
			continue
		}
		lower := strings.ToLower(s)
		if _, ok := seen[lower]; ok {
			continue
		}
		seen[lower] = struct{}{}
		out = append(out, s)
	}
	return out
}

func cleanSubject(s string) string {
	// Strip prefixes that make tags overly specific.
	for _, prefix := range []string{
		"Fiction, ", "fiction, ",
		"Juvenile fiction", "Juvenile nonfiction",
		"Young adult fiction", "Young adult nonfiction",
	} {
		if strings.HasPrefix(s, prefix) {
			s = strings.TrimSpace(s[len(prefix):])
			break
		}
	}
	// Strip trailing qualifiers.
	for _, suffix := range []string{
		", general", ", General",
		" (general)", " -- Fiction",
		" -- Juvenile fiction",
	} {
		s = strings.TrimSuffix(s, suffix)
	}
	s = strings.TrimSpace(s)
	// Title-case the first letter if it's lowercase after stripping.
	if len(s) > 0 && s[0] >= 'a' && s[0] <= 'z' {
		s = strings.ToUpper(s[:1]) + s[1:]
	}
	return s
}

func isNoise(s string) bool {
	lower := strings.ToLower(s)
	// Filter out places, languages, overly meta entries.
	for _, noise := range noisePatterns {
		if strings.Contains(lower, noise) {
			return true
		}
	}
	// Filter single-word entries that are likely places or too vague.
	if !strings.Contains(s, " ") && len(s) < 4 {
		return true
	}
	return false
}

var noisePatterns = []string{
	"in fiction",
	"in literature",
	"curriculum",
	"gcse",
	"key stage",
	"textbook",
	"reading level",
	"accessible book",
	"protected daisy",
	"internet archive",
	"open library",
	"lending library",
	"large type",
	"large print",
	"fiction in english",
	"american fiction",
	"english fiction",
	"french fiction",
	"german fiction",
	"spanish fiction",
	"russian fiction",
	"nyt:",
	"new york times",
}

// canonicalMap merges verbose OL subjects to clean genre names.
var canonicalMap = map[string]string{
	// Fiction sub-genres
	"fiction, fantasy, general":          "Fantasy",
	"fiction, fantasy, epic":             "Epic Fantasy",
	"fiction, science fiction, general":   "Science Fiction",
	"fiction, science fiction, hard":      "Hard Science Fiction",
	"fiction, science fiction, space opera": "Space Opera",
	"fiction, dystopian":                 "Dystopia",
	"fiction, action & adventure":        "Action & Adventure",
	"fiction, action and adventure":      "Action & Adventure",
	"fiction, thrillers, general":        "Thriller",
	"fiction, mystery & detective, general": "Mystery",
	"fiction, historical, general":       "Historical Fiction",
	"fiction, literary":                  "Literary Fiction",
	"fiction, romance, general":          "Romance",
	"fiction, horror":                    "Horror",
	"fiction, satire":                    "Satire",
	"fiction, war & military":            "Military Fiction",
	"fiction, psychological":             "Psychological Fiction",
	"fiction, short stories":             "Short Stories",
	"fiction, humor":                     "Humor",
	"domestic fiction":                   "Fiction",
	"dystopias":                          "Dystopia",
	"dystopian fiction":                  "Dystopia",
	"fantasy & magic":                    "Fantasy",
	"fantasy fiction":                    "Fantasy",
	"science fiction":                    "Science Fiction",
	"science fiction, english":           "Science Fiction",
	"science fiction & fantasy":          "Science Fiction & Fantasy",
	"adventure and adventurers":          "Action & Adventure",
	"spy stories":                        "Spy Fiction",
	"detective and mystery stories":      "Mystery",
	"ghost stories":                      "Ghost Stories",

	// Non-fiction
	"philosophy":                         "Philosophy",
	"filosofía":                          "Philosophy",
	"filosofie":                          "Philosophy",
	"philosophie":                        "Philosophy",
	"filosofia contemporanea":            "Philosophy",
	"philosophie, histoire":              "Philosophy",
	"biography & autobiography":          "Biography",
	"biography & autobiography, general": "Biography",
	"biography":                          "Biography",
	"history":                            "History",
	"history, general":                   "History",
	"self-help":                          "Self-Help",
	"self-help, general":                 "Self-Help",
	"business & economics":              "Business",
	"business & economics, general":     "Business",
	"psychology":                         "Psychology",
	"psychology, general":               "Psychology",
	"social science":                     "Social Science",
	"political science":                  "Politics",
	"political science, general":         "Politics",
	"computers":                          "Technology",
	"computers, general":                 "Technology",
	"technology & engineering":           "Technology",

	// Literature categories
	"english literature":                 "Literature",
	"english literature: literary criticism": "Literary Criticism",
	"literary criticism":                 "Literary Criticism",
	"literary criticism, general":        "Literary Criticism",
	"drama":                              "Drama",
	"drama (dramatic works by one author)": "Drama",
	"poetry":                             "Poetry",

	// Children/YA
	"juvenile fiction":                   "Children's",
	"young adult fiction":                "Young Adult",
}
