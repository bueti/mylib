<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import type { Book } from '$lib/api/client';
	import UploadDialog from '$lib/UploadDialog.svelte';

	interface RecentEntry {
		book: Book;
		progress: { position: string; percent: number };
	}
	interface TagCount {
		name: string;
		count: number;
	}

	// Filter state derived from URL search params.
	let query = $state('');
	let activeAuthor = $state<{ id: number; name: string } | null>(null);
	let activeSeries = $state<{ id: number; name: string } | null>(null);
	let activeTags = $state<string[]>([]);
	let activeFormat = $state<string | null>(null);
	let activeSort = $state('-added');

	let books = $state<Book[]>([]);
	let total = $state(0);
	let loading = $state(false);
	let loadingMore = $state(false);
	let error = $state<string | null>(null);
	const PAGE_SIZE = 60;
	let recent = $state<RecentEntry[]>([]);
	let allTags = $state<TagCount[]>([]);
	let sidebarOpen = $state(true);

	// Only show tags with 2+ books in the sidebar to reduce noise.
	let sidebarTags = $derived(allTags.filter((t) => t.count >= 2));

	// Rescan state.
	let scanning = $state(false);
	let scanMessage = $state<string | null>(null);
	let uploadOpen = $state(false);

	onMount(async () => {
		await Promise.all([loadRecent(), loadTags()]);
	});

	async function loadRecent() {
		try {
			const res = await fetch('/api/progress/recent?limit=8', { credentials: 'same-origin' });
			if (!res.ok) return;
			const data = await res.json();
			recent = (data?.entries ?? []) as RecentEntry[];
		} catch {
			// ignore
		}
	}

	async function loadTags() {
		try {
			const res = await fetch('/api/tags', { credentials: 'same-origin' });
			if (!res.ok) return;
			const data = await res.json();
			allTags = (data?.tags ?? []) as TagCount[];
		} catch {
			// ignore
		}
	}

	// Sync local state ← URL.
	$effect(() => {
		const sp = page.url.searchParams;
		query = sp.get('q') ?? '';
		const aid = Number(sp.get('author_id'));
		activeAuthor = aid > 0 ? { id: aid, name: sp.get('author_name') ?? `#${aid}` } : null;
		const sid = Number(sp.get('series_id'));
		activeSeries = sid > 0 ? { id: sid, name: sp.get('series_name') ?? `#${sid}` } : null;
		activeTags = sp.getAll('tag');
		activeFormat = sp.get('format');
		activeSort = sp.get('sort') ?? '-added';
	});

	// Debounced fetch.
	let debounceTimer: ReturnType<typeof setTimeout> | undefined;
	$effect(() => {
		const snapshot = {
			q: query,
			author: activeAuthor?.id,
			series: activeSeries?.id,
			tags: activeTags,
			format: activeFormat,
			sort: activeSort
		};
		clearTimeout(debounceTimer);
		debounceTimer = setTimeout(() => load(snapshot), 150);
		return () => clearTimeout(debounceTimer);
	});

	function buildParams(f: {
		q: string;
		author?: number;
		series?: number;
		tags: string[];
		format?: string | null;
		sort: string;
	}, offset: number): URLSearchParams {
		const params = new URLSearchParams();
		if (f.q) params.set('q', f.q);
		if (f.author) params.set('author_id', String(f.author));
		if (f.series) params.set('series_id', String(f.series));
		if (f.tags.length > 0) params.set('tag', f.tags.join(','));
		if (f.format) params.set('format', f.format);
		params.set('sort', f.sort);
		params.set('limit', String(PAGE_SIZE));
		params.set('offset', String(offset));
		return params;
	}

	async function load(f: {
		q: string;
		author?: number;
		series?: number;
		tags: string[];
		format?: string | null;
		sort: string;
	}) {
		loading = true;
		error = null;
		try {
			const params = buildParams(f, 0);
			const res = await fetch('/api/books?' + params.toString(), { credentials: 'same-origin' });
			if (!res.ok) throw new Error('HTTP ' + res.status);
			const data = await res.json();
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

	async function loadMore() {
		if (loadingMore || books.length >= total) return;
		loadingMore = true;
		try {
			const params = buildParams({
				q: query, author: activeAuthor?.id, series: activeSeries?.id,
				tags: activeTags, format: activeFormat, sort: activeSort
			}, books.length);
			const res = await fetch('/api/books?' + params.toString(), { credentials: 'same-origin' });
			if (!res.ok) throw new Error('HTTP ' + res.status);
			const data = await res.json();
			books = [...books, ...(data?.books ?? [])];
		} catch {
			// silent — user can try scrolling again
		} finally {
			loadingMore = false;
		}
	}

	function onSearch(e: Event) {
		const value = (e.target as HTMLInputElement).value;
		const sp = new URLSearchParams(page.url.searchParams);
		if (value) sp.set('q', value);
		else sp.delete('q');
		goto('/?' + sp.toString(), { keepFocus: true, replaceState: true, noScroll: true });
	}

	function toggleTag(tag: string) {
		const sp = new URLSearchParams(page.url.searchParams);
		const current = sp.getAll('tag');
		sp.delete('tag');
		if (current.includes(tag)) {
			for (const t of current) {
				if (t !== tag) sp.append('tag', t);
			}
		} else {
			for (const t of current) sp.append('tag', t);
			sp.append('tag', tag);
		}
		goto('/?' + sp.toString(), { keepFocus: true, replaceState: false, noScroll: true });
	}

	function clearFilter(key: string) {
		const sp = new URLSearchParams(page.url.searchParams);
		sp.delete(key);
		if (key === 'author_id') sp.delete('author_name');
		if (key === 'series_id') sp.delete('series_name');
		goto('/?' + sp.toString(), { keepFocus: true, replaceState: false });
	}

	function setSort(s: string) {
		const sp = new URLSearchParams(page.url.searchParams);
		sp.set('sort', s);
		goto('/?' + sp.toString(), { keepFocus: true, replaceState: true, noScroll: true });
	}

	let hasFilters = $derived(
		activeTags.length > 0 || activeAuthor || activeSeries || activeFormat || query
	);

	async function rescan() {
		scanning = true;
		scanMessage = 'Starting scan…';
		try {
			const triggerRes = await fetch('/api/scan', {
				method: 'POST',
				credentials: 'same-origin'
			});
			if (!triggerRes.ok) throw new Error('Failed to start scan');
			const job = await triggerRes.json();
			await new Promise<void>((resolve) => {
				const es = new EventSource(`/api/scan/${job.id}/events`);
				es.addEventListener('job', (e) => {
					const snap = JSON.parse((e as MessageEvent).data);
					if (snap.status === 'running') {
						scanMessage = `Scanning… ${snap.files_seen} seen, ${snap.files_added} new`;
					} else if (snap.status === 'done') {
						scanMessage = `Scan complete · +${snap.files_added} / ~${snap.files_updated} / −${snap.files_removed}`;
						es.close();
						resolve();
					} else {
						scanMessage = 'Scan failed: ' + (snap.error || 'unknown');
						es.close();
						resolve();
					}
				});
				es.onerror = () => {
					es.close();
					resolve();
				};
			});
			load({
				q: query,
				author: activeAuthor?.id,
				series: activeSeries?.id,
				tags: activeTags,
				format: activeFormat,
				sort: activeSort
			});
			void loadRecent();
			void loadTags();
		} catch (e) {
			scanMessage = e instanceof Error ? e.message : 'Scan failed';
		} finally {
			scanning = false;
			setTimeout(() => (scanMessage = null), 4000);
		}
	}
</script>

<div class="layout" class:sidebar-visible={sidebarOpen && sidebarTags.length > 0}>
	<!-- Tag sidebar -->
	{#if sidebarTags.length > 0}
		<aside class="sidebar" class:open={sidebarOpen}>
			<header>
				<span>Genres & Topics</span>
				<button onclick={() => (sidebarOpen = false)} aria-label="Close sidebar">×</button>
			</header>
			<ul>
				{#each sidebarTags as tag (tag.name)}
					<li>
						<button
							class:active={activeTags.includes(tag.name)}
							onclick={() => toggleTag(tag.name)}
						>
							<span class="tag-name">{tag.name}</span>
							<span class="tag-count">{tag.count}</span>
						</button>
					</li>
				{/each}
			</ul>
		</aside>
	{/if}

	<div class="main">
		<div class="toolbar">
			{#if allTags.length > 0 && !sidebarOpen}
				<button class="sidebar-toggle" onclick={() => (sidebarOpen = true)} title="Show genres">☰</button>
			{/if}
			<input
				type="search"
				placeholder="Search title, author, series, genre…"
				value={query}
				oninput={onSearch}
				aria-label="Search"
			/>
			<select value={activeSort} onchange={(e) => setSort((e.target as HTMLSelectElement).value)}>
				<option value="-added">Recently added</option>
				<option value="title">Title A–Z</option>
				<option value="-title">Title Z–A</option>
				<option value="added">Oldest first</option>
			</select>
			<button class="upload" onclick={() => (uploadOpen = true)}>Upload</button>
			<button onclick={rescan} disabled={scanning} class="rescan">
				{scanning ? 'Scanning…' : 'Rescan'}
			</button>
			<span class="count">{total} {total === 1 ? 'book' : 'books'}</span>
		</div>

		{#if scanMessage}
			<p class="scan-msg">{scanMessage}</p>
		{/if}

		{#if activeAuthor || activeSeries || activeTags.length > 0 || activeFormat}
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
				{#each activeTags as tag}
					<button class="chip" onclick={() => toggleTag(tag)}>
						{tag} ✕
					</button>
				{/each}
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

		<!-- Browse by genre section (shown when no active filters) -->
		{#if !hasFilters && sidebarTags.length > 0}
			<section class="browse">
				<h2>Browse by genre</h2>
				<div class="genre-grid">
					{#each sidebarTags.slice(0, 20) as tag (tag.name)}
						<button class="genre-card" onclick={() => toggleTag(tag.name)}>
							<span class="genre-name">{tag.name}</span>
							<span class="genre-count">{tag.count} {tag.count === 1 ? 'book' : 'books'}</span>
						</button>
					{/each}
				</div>
			</section>
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
							<span class="format-badge">{book.format.toUpperCase()}</span>
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
			{#if books.length < total}
				<div class="load-more">
					<button onclick={loadMore} disabled={loadingMore}>
						{loadingMore ? 'Loading…' : `Load more (${books.length} of ${total})`}
					</button>
				</div>
			{/if}
		{/if}
	</div>
</div>

<UploadDialog bind:open={uploadOpen} onDone={() => {
	load({ q: query, author: activeAuthor?.id, series: activeSeries?.id, tags: activeTags, format: activeFormat, sort: activeSort });
	void loadTags();
}} />

<style>
	.layout {
		display: flex;
		gap: 0;
		margin: -2rem;
		min-height: calc(100vh - 60px);
	}
	.sidebar {
		flex: 0 0 240px;
		width: 240px;
		background: #fff;
		border-right: 1px solid #e5e5e5;
		overflow-y: auto;
		display: none;
	}
	.layout.sidebar-visible .sidebar {
		display: block;
	}
	.sidebar header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 1rem;
		font-weight: 600;
		font-size: 0.875rem;
		border-bottom: 1px solid #eee;
		position: sticky;
		top: 0;
		background: #fff;
	}
	.sidebar header button {
		background: none;
		border: 0;
		font-size: 1.25rem;
		cursor: pointer;
		color: #666;
	}
	.sidebar ul {
		list-style: none;
		padding: 0.25rem 0;
		margin: 0;
	}
	.sidebar button {
		width: 100%;
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 0.375rem 1rem;
		background: none;
		border: 0;
		font-size: 0.8125rem;
		color: #333;
		cursor: pointer;
		text-align: left;
	}
	.sidebar button:hover {
		background: #f5f5f5;
	}
	.sidebar button.active {
		background: #eef;
		font-weight: 600;
		color: #0366d6;
	}
	.tag-count {
		color: #999;
		font-size: 0.75rem;
	}
	.main {
		flex: 1;
		padding: 2rem;
		min-width: 0;
		max-width: 1200px;
	}
	.toolbar {
		display: flex;
		gap: 0.75rem;
		align-items: center;
		margin-bottom: 1rem;
		flex-wrap: wrap;
	}
	.sidebar-toggle {
		background: transparent;
		border: 1px solid #ccc;
		border-radius: 4px;
		padding: 0.375rem 0.5rem;
		cursor: pointer;
		font-size: 1rem;
	}
	input[type='search'] {
		flex: 1;
		min-width: 200px;
		padding: 0.5rem 0.75rem;
		font-size: 1rem;
		border: 1px solid #ccc;
		border-radius: 4px;
		background: #fff;
	}
	select {
		padding: 0.5rem 0.625rem;
		border: 1px solid #ccc;
		border-radius: 4px;
		font-size: 0.875rem;
		background: #fff;
	}
	.upload {
		padding: 0.5rem 1rem;
		background: #0366d6;
		color: #fff;
		border: 0;
		border-radius: 4px;
		font-size: 0.875rem;
		cursor: pointer;
	}
	.upload:hover {
		background: #0256b9;
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
	/* Continue reading */
	.continue {
		margin: 0 0 2rem;
	}
	.continue h2,
	.browse h2 {
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
	/* Browse by genre */
	.browse {
		margin: 0 0 2rem;
	}
	.genre-grid {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
	}
	.genre-card {
		display: flex;
		flex-direction: column;
		align-items: center;
		padding: 0.5rem 1rem;
		background: #fff;
		border: 1px solid #e0e0e0;
		border-radius: 8px;
		cursor: pointer;
		transition: border-color 0.1s;
	}
	.genre-card:hover {
		border-color: #0366d6;
	}
	.genre-name {
		font-weight: 600;
		font-size: 0.8125rem;
	}
	.genre-count {
		font-size: 0.6875rem;
		color: #888;
	}
	/* Book grid */
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
		position: relative;
		aspect-ratio: 2 / 3;
		background: #eee;
		border-radius: 4px;
		overflow: hidden;
	}
	.format-badge {
		position: absolute;
		top: 0.375rem;
		right: 0.375rem;
		padding: 0.1rem 0.375rem;
		background: rgba(0, 0, 0, 0.6);
		color: #fff;
		font-size: 0.625rem;
		font-weight: 700;
		letter-spacing: 0.03em;
		border-radius: 3px;
		line-height: 1.4;
	}
	.cover img {
		width: 100%;
		height: 100%;
		object-fit: cover;
		display: block;
	}
	.cover .placeholder {
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
	.load-more {
		display: flex;
		justify-content: center;
		margin-top: 2rem;
	}
	.load-more button {
		padding: 0.625rem 2rem;
		background: #fff;
		border: 1px solid #ccc;
		border-radius: 4px;
		font-size: 0.9375rem;
		color: #333;
		cursor: pointer;
	}
	.load-more button:hover:not(:disabled) {
		border-color: #888;
		background: #f5f5f5;
	}
	.load-more button:disabled {
		color: #999;
		cursor: wait;
	}
	@media (max-width: 768px) {
		.sidebar {
			display: none !important;
		}
		.layout {
			margin: 0;
		}
		.main {
			padding: 1rem;
		}
	}
</style>
