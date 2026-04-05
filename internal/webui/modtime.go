package webui

import "time"

// staticModTime is a fixed timestamp used when serving the embedded
// index.html fallback. Using a fixed value makes ETag/If-Modified-Since
// behaviour predictable across restarts.
var staticModTime = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
