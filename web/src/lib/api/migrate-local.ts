// One-shot migration of v0.1 localStorage CFIs into server-side
// reading_progress. Called once per browser on first authenticated
// page load.

const MIGRATED_KEY = 'mylib.progressMigrated';

interface LocalEntry {
	book_id: number;
	position: string;
	percent: number;
}

export async function migrateLocalProgress(): Promise<void> {
	if (typeof localStorage === 'undefined') return;
	if (localStorage.getItem(MIGRATED_KEY)) return;

	const entries: LocalEntry[] = [];
	const keysToRemove: string[] = [];

	for (let i = 0; i < localStorage.length; i++) {
		const key = localStorage.key(i);
		if (!key) continue;
		const m = /^mylib\.read\.(\d+)\.cfi$/.exec(key);
		if (!m) continue;
		const bookId = Number(m[1]);
		const position = localStorage.getItem(key);
		if (!position) continue;
		entries.push({ book_id: bookId, position, percent: 0 });
		keysToRemove.push(key);
	}

	if (entries.length === 0) {
		localStorage.setItem(MIGRATED_KEY, '1');
		return;
	}

	try {
		const res = await fetch('/api/progress/import', {
			method: 'POST',
			credentials: 'same-origin',
			headers: { 'content-type': 'application/json' },
			body: JSON.stringify({ entries })
		});
		if (res.ok) {
			for (const k of keysToRemove) localStorage.removeItem(k);
			localStorage.setItem(MIGRATED_KEY, '1');
		}
	} catch {
		// retry on next load
	}
}
