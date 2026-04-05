<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { client, type Book } from '$lib/api/client';

	interface RecentEntry {
		book: Book;
		progress: { position: string; percent: number };
	}

	// Filter state is derived from URL search params so links are
	// shareable and the browser back button restores prior views.
	let query = $state('');
	let activeAuthor = $state<{ id: number; name: string } | null>(null);
	let activeSeries = $state<{ id: number; name: string } | null>(null);
	let activeTag = $state<string | null>(null);
	let activeFormat = $state<string | null>(null);

	let books = $state<Book[]>([]);
	let total = $state(0);
	let loading = $state(false);
	let error = $state<string | null>(null);
	let recent = $state<RecentEntry[]>([]);

	onMount(async () => {
		await loadRecent();
	});

	async function loadRecent() {
		try {
			const res = await fetch('/api/progress/recent?limit=8', { credentials: 'same-origin' });
			if (!res.ok) return;
			const data = await res.json();
			recent = (data?.entries ?? []) as RecentEntry[];
		} catch {
			// ignore — section stays hidden
		}
	}

	// Rescan state.
	let scanning = $state(false);
	let scanMessage = $state<string | null>(null);

	// Sync local state ← URL whenever the URL changes.
	$effect(() => {
		const sp = page.url.searchParams;
		query = sp.get('q') ?? '';
		const aid = Number(sp.get('author_id'));
		activeAuthor = aid > 0 ? { id: aid, name: sp.get('author_name') ?? `#${aid}` } : null;
		const sid = Number(sp.get('series_id'));
		activeSeries = sid > 0 ? { id: sid, name: sp.get('series_name') ?? `#${sid}` } : null;
		activeTag = sp.get('tag');
		activeFormat = sp.get('format');
	});

	// Debounced fetch whenever any filter changes.
	let debounceTimer: ReturnType<typeof setTimeout> | undefined;
	$effect(() => {
		// Re-subscribe to all filter signals so this effect re-runs on change.
		const snapshot = {
			q: query,
			author: activeAuthor?.id,
			series: activeSeries?.id,
			tag: activeTag,
			format: activeFormat
		};
		clearTimeout(debounceTimer);
		debounceTimer = setTimeout(() => load(snapshot), 150);
		return () => clearTimeout(debounceTimer);
	});

	async function load(f: {
		q: string;
		author?: number;
		series?: number;
		tag?: string | null;
		format?: string | null;
	}) {
		loading = true;
		error = null;
		try {
			const { data, error: err, response } = await client.GET('/api/books', {
				params: {
					query: {
						q: f.q || undefined,
						author_id: f.author || undefined,
						series_id: f.series || undefined,
						tag: f.tag || undefined,
						format: f.format || undefined,
						limit: 60,
						sort: '-added'
					}
				}
			});
			if (err) throw new Error(err.detail || response.statusText);
			books = data?.books ?? [];
			total = data?.total ?? 0;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load';
			books = [];
			total = 0;
		} finally {
			loading = false;
		}
	}

	// Push the current search query into the URL (on user input).
	function onSearch(e: Event) {
		const value = (e.target as HTMLInputElement).value;
		const sp = new URLSearchParams(page.url.searchParams);
		if (value) sp.set('q', value);
		else sp.delete('q');
		goto('/?' + sp.toString(), { keepFocus: true, replaceState: true, noScroll: true });
	}

	function clearFilter(key: 'author_id' | 'series_id' | 'tag' | 'format' | 'q') {
		const sp = new URLSearchParams(page.url.searchParams);
		sp.delete(key);
		if (key === 'author_id') sp.delete('author_name');
		if (key === 'series_id') sp.delete('series_name');
		goto('/?' + sp.toString(), { keepFocus: true, replaceState: false });
	}

	async function rescan() {
		scanning = true;
		scanMessage = 'Starting scan…';
		try {
			const { data: job, error: err } = await client.POST('/api/scan');
			if (err || !job) throw new Error('Failed to start scan');
			let current = job;
			// Poll until the job finishes.
			while (current.status === 'running') {
				await new Promise((r) => setTimeout(r, 500));
				const { data: polled } = await client.GET('/api/scan/{id}', {
					params: { path: { id: current.id } }
				});
				if (!polled) break;
				current = polled;
				scanMessage = `Scanning… ${current.files_seen} seen, ${current.files_added} new`;
			}
			if (current.status === 'done') {
				scanMessage = `Scan complete · +${current.files_added} / ~${current.files_updated} / −${current.files_removed}`;
			} else {
				scanMessage = 'Scan failed: ' + (current.error || 'unknown');
			}
			// Refresh the list.
			load({ q: query, author: activeAuthor?.id, series: activeSeries?.id, tag: activeTag, format: activeFormat });
		} catch (e) {
			scanMessage = e instanceof Error ? e.message : 'Scan failed';
		} finally {
			scanning = false;
			setTimeout(() => (scanMessage = null), 4000);
		}
	}
</script>

<div class="toolbar">
	<input
		type="search"
		placeholder="Search title, author, series…"
		value={query}
		oninput={onSearch}
		aria-label="Search"
	/>
	<button onclick={rescan} disabled={scanning} class="rescan">
		{scanning ? 'Scanning…' : 'Rescan'}
	</button>
	<span class="count">{total} {total === 1 ? 'book' : 'books'}</span>
</div>

{#if scanMessage}
	<p class="scan-msg">{scanMessage}</p>
{/if}

{#if recent.length > 0}
	<section class="continue">
		<h2>Continue reading</h2>
		<ul class="row">
			{#each recent as entry (entry.book.id)}
				<li>
					<a href="/books/{entry.book.id}/read" class="recent-card">
						{#if entry.book.has_cover}
							<img src="/api/books/{entry.book.id}/cover" alt="" loading="lazy" />
						{:else}
							<div class="placeholder">{entry.book.title.charAt(0)}</div>
						{/if}
						<div class="progress-bar">
							<div
								class="progress-fill"
								style:width="{Math.max(2, Math.round(entry.progress.percent * 100))}%"
							></div>
						</div>
						<div class="recent-title" title={entry.book.title}>{entry.book.title}</div>
					</a>
				</li>
			{/each}
		</ul>
	</section>
{/if}

{#if activeAuthor || activeSeries || activeTag || activeFormat}
	<div class="chips">
		<span class="chips-label">Filters:</span>
		{#if activeAuthor}
			<button class="chip" onclick={() => clearFilter('author_id')}>
				Author: {activeAuthor.name} ✕
			</button>
		{/if}
		{#if activeSeries}
			<button class="chip" onclick={() => clearFilter('series_id')}>
				Series: {activeSeries.name} ✕
			</button>
		{/if}
		{#if activeTag}
			<button class="chip" onclick={() => clearFilter('tag')}>
				Tag: {activeTag} ✕
			</button>
		{/if}
		{#if activeFormat}
			<button class="chip" onclick={() => clearFilter('format')}>
				Format: {activeFormat} ✕
			</button>
		{/if}
	</div>
{/if}

{#if error}
	<p class="error">Error: {error}</p>
{/if}

{#if loading && books.length === 0}
	<p>Loading…</p>
{:else if books.length === 0}
	<p class="empty">
		No books match. Point MYLIB_LIBRARY_ROOTS at a directory of EPUBs or PDFs and hit
		<button class="link" onclick={rescan}>Rescan</button>.
	</p>
{:else}
	<ul class="grid">
		{#each books as book (book.id)}
			<li class="card">
				<a href="/books/{book.id}" class="cover">
					{#if book.has_cover}
						<img src="/api/books/{book.id}/cover" alt="" loading="lazy" />
					{:else}
						<div class="placeholder">{book.title.charAt(0)}</div>
					{/if}
				</a>
				<div class="meta">
					<a href="/books/{book.id}" class="title" title={book.title}>{book.title}</a>
					<div class="authors">
						{(book.authors ?? []).map((a) => a.name).join(', ') || '—'}
					</div>
				</div>
			</li>
		{/each}
	</ul>
{/if}

<style>
	.toolbar {
		display: flex;
		gap: 1rem;
		align-items: center;
		margin-bottom: 1rem;
	}
	input[type='search'] {
		flex: 1;
		padding: 0.5rem 0.75rem;
		font-size: 1rem;
		border: 1px solid #ccc;
		border-radius: 4px;
		background: #fff;
	}
	.rescan {
		padding: 0.5rem 1rem;
		background: #222;
		color: #fff;
		border: 0;
		border-radius: 4px;
		font-size: 0.875rem;
		cursor: pointer;
	}
	.rescan:disabled {
		background: #888;
		cursor: wait;
	}
	.rescan:hover:not(:disabled) {
		background: #000;
	}
	.count {
		color: #666;
		font-size: 0.875rem;
	}
	.continue {
		margin: 0 0 2rem;
	}
	.continue h2 {
		font-size: 0.9375rem;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		color: #666;
		margin: 0 0 0.75rem;
	}
	.row {
		list-style: none;
		padding: 0;
		margin: 0;
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
		gap: 1rem;
	}
	.recent-card {
		display: flex;
		flex-direction: column;
		gap: 0.375rem;
		text-decoration: none;
		color: inherit;
	}
	.recent-card img,
	.recent-card .placeholder {
		width: 100%;
		aspect-ratio: 2 / 3;
		object-fit: cover;
		background: #eee;
		border-radius: 4px;
	}
	.recent-card .placeholder {
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 2rem;
		font-weight: 700;
		color: #aaa;
	}
	.progress-bar {
		height: 3px;
		background: #e0e0e0;
		border-radius: 2px;
		overflow: hidden;
	}
	.progress-fill {
		height: 100%;
		background: #0366d6;
	}
	.recent-title {
		font-size: 0.8125rem;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.scan-msg {
		margin: 0 0 1rem;
		padding: 0.5rem 0.75rem;
		background: #f0f7ff;
		border-left: 3px solid #0366d6;
		font-size: 0.875rem;
		color: #0366d6;
	}
	.chips {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		flex-wrap: wrap;
		margin-bottom: 1.25rem;
	}
	.chips-label {
		color: #666;
		font-size: 0.875rem;
	}
	.chip {
		display: inline-flex;
		align-items: center;
		gap: 0.375rem;
		padding: 0.25rem 0.625rem;
		background: #eef;
		border: 0;
		border-radius: 12px;
		font-size: 0.8125rem;
		color: #224;
		cursor: pointer;
	}
	.chip:hover {
		background: #dde;
	}
	.error {
		color: #b00020;
	}
	.empty {
		color: #666;
		line-height: 1.5;
	}
	.empty .link {
		background: none;
		border: 0;
		padding: 0;
		color: #0366d6;
		cursor: pointer;
		text-decoration: underline;
		font: inherit;
	}
	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
		gap: 1.25rem;
		list-style: none;
		padding: 0;
		margin: 0;
	}
	.card {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}
	.cover {
		display: block;
		aspect-ratio: 2 / 3;
		background: #eee;
		border-radius: 4px;
		overflow: hidden;
	}
	.cover img {
		width: 100%;
		height: 100%;
		object-fit: cover;
		display: block;
	}
	.placeholder {
		width: 100%;
		height: 100%;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 3rem;
		font-weight: 700;
		color: #aaa;
		background: #f0f0f0;
	}
	.meta {
		display: flex;
		flex-direction: column;
		gap: 0.125rem;
		min-width: 0;
	}
	.title {
		font-weight: 600;
		color: inherit;
		text-decoration: none;
		display: -webkit-box;
		-webkit-line-clamp: 2;
		line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
		line-height: 1.3;
	}
	.title:hover {
		color: #0366d6;
	}
	.authors {
		color: #666;
		font-size: 0.875rem;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
</style>
