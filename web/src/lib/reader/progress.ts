// Server-side reading-progress helper. Handles debounced saves and a
// final flush via sendBeacon on page unload.

export interface SavedProgress {
	position: string;
	percent: number;
	theme?: string;
	font_size?: string;
}

export async function loadProgress(bookId: number): Promise<SavedProgress | null> {
	try {
		const res = await fetch(`/api/books/${bookId}/progress`, { credentials: 'same-origin' });
		if (res.status === 404) return null;
		if (!res.ok) return null;
		return (await res.json()) as SavedProgress;
	} catch {
		return null;
	}
}

export function makeSaver(bookId: number, debounceMs = 5000) {
	let timer: ReturnType<typeof setTimeout> | undefined;
	let pending: SavedProgress | null = null;

	async function flush() {
		if (!pending) return;
		const body = pending;
		pending = null;
		try {
			await fetch(`/api/books/${bookId}/progress`, {
				method: 'PUT',
				headers: { 'content-type': 'application/json' },
				credentials: 'same-origin',
				body: JSON.stringify({ ...body, finished: false })
			});
		} catch {
			// Network errors are non-fatal; we'll save again on the next tick.
		}
	}

	function save(p: SavedProgress) {
		pending = p;
		clearTimeout(timer);
		timer = setTimeout(flush, debounceMs);
	}

	function flushNow() {
		clearTimeout(timer);
		if (!pending) return;
		// sendBeacon is best-effort and fire-and-forget — perfect for unload.
		const body = JSON.stringify({ ...pending, finished: false });
		const blob = new Blob([body], { type: 'application/json' });
		if (typeof navigator !== 'undefined' && navigator.sendBeacon) {
			navigator.sendBeacon(`/api/books/${bookId}/progress`, blob);
		} else {
			// best-effort non-beacon fallback
			void flush();
		}
		pending = null;
	}

	return { save, flush: flushNow };
}
