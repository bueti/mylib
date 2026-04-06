package enrich

import (
	"regexp"
	"strings"
)

// NormalizeSubjects takes raw Open Library/EPUB subjects and returns
// cleaner, deduplicated genre tags. It:
//  1. Maps known verbose subjects to canonical short names.
//  2. Strips common noise prefixes ("Fiction, ", "Juvenile ").
//  3. Strips trailing qualifiers (", general", ", etc.").
//  4. Filters out non-genre entries (places, languages, curricula,
//     overly specific academic labels, meta-tags).
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
		"Juvenile fiction, ", "Juvenile nonfiction, ",
		"Young adult fiction, ", "Young adult nonfiction, ",
		"Children's fiction, ",
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
		" -- Juvenile literature",
	} {
		s = strings.TrimSuffix(s, suffix)
	}
	// Strip parenthetical qualifiers like "(fictional works by one author)"
	s = parenSuffix.ReplaceAllString(s, "")
	s = strings.TrimSpace(s)
	// Title-case the first letter if it's lowercase after stripping.
	if len(s) > 0 && s[0] >= 'a' && s[0] <= 'z' {
		s = strings.ToUpper(s[:1]) + s[1:]
	}
	return s
}

var parenSuffix = regexp.MustCompile(`\s*\([^)]+\)\s*$`)

func isNoise(s string) bool {
	lower := strings.ToLower(s)
	// Exact-match blocklist for common non-genre tags.
	if _, ok := noiseExact[lower]; ok {
		return true
	}
	// Substring patterns.
	for _, noise := range noisePatterns {
		if strings.Contains(lower, noise) {
			return true
		}
	}
	// Filter entries that are just a language name.
	if _, ok := languages[lower]; ok {
		return true
	}
	// Filter single-word entries that are too vague.
	if !strings.Contains(s, " ") && len(s) < 5 {
		return true
	}
	return false
}

// noiseExact is a set of exact (lowercased) tags to drop entirely.
var noiseExact = map[string]struct{}{
	"general":                   {},
	"readers":                   {},
	"fiction":                   {},
	"nonfiction":                {},
	"literature":                {},
	"novels":                    {},
	"roman":                     {},
	"accessible book":           {},
	"protected daisy":           {},
	"open_syllabus_project":     {},
	"open syllabus project":     {},
	"social life and customs":   {},
	"social conditions":         {},
	"social aspects":            {},
	"interpersonal relations":   {},
	"man-woman relationships":   {},
	"young women":               {},
	"young men":                 {},
	"teenage boys":              {},
	"teenage girls":             {},
	"brothers and sisters":      {},
	"friendship":                {},
	"families":                  {},
	"family":                    {},
	"orphans":                   {},
	"missing persons":           {},
	"conduct of life":           {},
	"identity":                  {},
	"coming of age":             {},
	"bildungsromans":            {},
	"reading":                   {},
	"books and reading":         {},
	"authorship":                {},
	"criticism and interpretation": {},
	"application software":      {},
	"unknown":                   {},
}

var noisePatterns = []string{
	"in fiction",
	"in literature",
	"curriculum",
	"gcse",
	"key stage",
	"textbook",
	"reading level",
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
	"italian fiction",
	"nyt:",
	"new york times",
	"best seller",
	"bestseller",
	"fictional works by",
	"works by one author",
	"literature 1",  // "French literature 1900", "American literature 1900"
	"literature 2",  // "American literature 2000"
	"literature, 1",
	"literature, 2",
	"history and criticism",
	"translations into",
	"translations from",
	"examinations",
	"study and teaching",
	"problems, exercises",
	"outlines, syllabi",
}

var languages = map[string]struct{}{
	"english":    {},
	"english language": {},
	"french":     {},
	"german":     {},
	"spanish":    {},
	"italian":    {},
	"portuguese": {},
	"russian":    {},
	"chinese":    {},
	"japanese":   {},
	"arabic":     {},
	"latin":      {},
	"greek":      {},
}

// canonicalMap merges verbose OL subjects to clean genre names.
var canonicalMap = map[string]string{
	// Fiction sub-genres
	"fiction, fantasy, general":              "Fantasy",
	"fiction, fantasy, epic":                 "Epic Fantasy",
	"fiction, fantasy, urban":                "Urban Fantasy",
	"fiction, fantasy, contemporary":         "Fantasy",
	"fiction, science fiction, general":      "Science Fiction",
	"fiction, science fiction, hard":         "Hard Science Fiction",
	"fiction, science fiction, space opera":  "Space Opera",
	"fiction, science fiction, cyberpunk":    "Cyberpunk",
	"fiction, dystopian":                     "Dystopia",
	"fiction, action & adventure":            "Action & Adventure",
	"fiction, action and adventure":          "Action & Adventure",
	"fiction, thrillers, general":            "Thriller",
	"fiction, thrillers, suspense":           "Thriller",
	"fiction, mystery & detective, general":  "Mystery",
	"fiction, mystery & detective":           "Mystery",
	"fiction, historical, general":           "Historical Fiction",
	"fiction, historical":                    "Historical Fiction",
	"fiction, literary":                      "Literary Fiction",
	"fiction, romance, general":              "Romance",
	"fiction, romance, contemporary":         "Romance",
	"fiction, romance, historical":           "Historical Romance",
	"fiction, horror":                        "Horror",
	"fiction, satire":                        "Satire",
	"fiction, war & military":                "Military Fiction",
	"fiction, psychological":                 "Psychological Fiction",
	"fiction, short stories":                 "Short Stories",
	"fiction, short stories (single author)": "Short Stories",
	"fiction, humor":                         "Humor",
	"fiction, crime":                         "Crime Fiction",
	"fiction, noir":                          "Noir",
	"domestic fiction":                       "Literary Fiction",
	"dystopias":                              "Dystopia",
	"dystopian fiction":                      "Dystopia",
	"fantasy & magic":                        "Fantasy",
	"fantasy fiction":                        "Fantasy",
	"science fiction":                        "Science Fiction",
	"science fiction, english":               "Science Fiction",
	"science fiction & fantasy":              "Science Fiction & Fantasy",
	"adventure and adventurers":              "Action & Adventure",
	"spy stories":                            "Spy Fiction",
	"detective and mystery stories":          "Mystery",
	"ghost stories":                          "Horror",
	"horror fiction":                         "Horror",
	"horror tales":                           "Horror",
	"love stories":                           "Romance",
	"war stories":                            "Military Fiction",
	"suspense fiction":                       "Thriller",
	"thriller":                               "Thriller",
	"thrillers":                              "Thriller",

	// Classics normalization
	"classic":                  "Classics",
	"classic literature":       "Classics",
	"classical literature":     "Classics",

	// Children's/YA normalization
	"juvenile fiction":          "Children's",
	"children's fiction":        "Children's",
	"children's stories":        "Children's",
	"children's literature":     "Children's",
	"young adult fiction":       "Young Adult",
	"young adult literature":    "Young Adult",

	// British/Irish fiction
	"british and irish fiction":                              "British Fiction",
	"british and irish fiction (fictional works by one author)": "British Fiction",
	"english literature":                                     "British Fiction",

	// Non-fiction
	"philosophy":                 "Philosophy",
	"filosofía":                  "Philosophy",
	"filosofie":                  "Philosophy",
	"philosophie":                "Philosophy",
	"filosofia":                  "Philosophy",
	"filosofia contemporanea":    "Philosophy",
	"philosophie, histoire":      "Philosophy",
	"biography & autobiography":  "Biography",
	"biography & autobiography, general": "Biography",
	"biography":                  "Biography",
	"autobiographies":            "Biography",
	"history":                    "History",
	"history, general":           "History",
	"world history":              "History",
	"self-help":                  "Self-Help",
	"self-help, general":         "Self-Help",
	"self-improvement":           "Self-Help",
	"business & economics":       "Business",
	"business & economics, general": "Business",
	"business":                   "Business",
	"economics":                  "Economics",
	"psychology":                 "Psychology",
	"psychology, general":        "Psychology",
	"social science":             "Social Science",
	"social sciences":            "Social Science",
	"political science":          "Politics",
	"political science, general": "Politics",
	"computers":                  "Technology",
	"computers, general":         "Technology",
	"technology & engineering":   "Technology",
	"computer science":           "Computer Science",
	"computer programming":       "Programming",
	"programming languages (electronic computers)": "Programming",
	"software engineering":       "Programming",
	"web development":            "Programming",
	"mathematics":                "Mathematics",
	"science":                    "Science",
	"science, general":           "Science",
	"popular science":            "Science",
	"nature":                     "Nature",
	"religion":                   "Religion",
	"spirituality":               "Spirituality",
	"true crime":                 "True Crime",
	"travel":                     "Travel",
	"cooking":                    "Cooking",
	"health & fitness":           "Health",
	"medical":                    "Health",
	"art":                        "Art",
	"music":                      "Music",
	"education":                  "Education",

	// Literature categories
	"literary criticism":                    "Literary Criticism",
	"literary criticism, general":           "Literary Criticism",
	"english literature: literary criticism": "Literary Criticism",
	"drama":                                 "Drama",
	"drama (dramatic works by one author)":  "Drama",
	"plays":                                 "Drama",
	"poetry":                                "Poetry",
	"essays":                                "Essays",
}
