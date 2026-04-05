<script lang="ts">
	import { client } from '$lib/api/client';
	import type { Book } from '$lib/api/schema';

	let query = $state('');
	let books = $state<Book[]>([]);
	let total = $state(0);
	let loading = $state(false);
	let error = $state<string | null>(null);

	// Debounce query → load.
	let debounceTimer: ReturnType<typeof setTimeout> | undefined;
	$effect(() => {
		const q = query;
		clearTimeout(debounceTimer);
		debounceTimer = setTimeout(() => load(q), 200);
		return () => clearTimeout(debounceTimer);
	});

	async function load(q: string) {
		loading = true;
		error = null;
		try {
			const { data, error: err } = await client.GET('/books', {
				params: { query: { q: q || undefined, limit: 60, sort: '-added' } }
			});
			if (err) throw new Error(String(err));
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
</script>

<div class="toolbar">
	<input
		type="search"
		placeholder="Search title, author, series…"
		bind:value={query}
		aria-label="Search"
	/>
	<span class="count">{total} {total === 1 ? 'book' : 'books'}</span>
</div>

{#if error}
	<p class="error">Error: {error}</p>
{/if}

{#if loading && books.length === 0}
	<p>Loading…</p>
{:else if books.length === 0}
	<p class="empty">No books yet. Point MYLIB_LIBRARY_ROOTS at a directory of EPUBs or PDFs and hit <a href="/api/docs">the API docs</a> to trigger a scan.</p>
{:else}
	<ul class="grid">
		{#each books as book (book.id)}
			<li class="card">
				<a href="/api/books/{book.id}/file" class="cover">
					{#if book.has_cover}
						<img src="/api/books/{book.id}/cover" alt="" loading="lazy" />
					{:else}
						<div class="placeholder">{book.title.charAt(0)}</div>
					{/if}
				</a>
				<div class="meta">
					<div class="title" title={book.title}>{book.title}</div>
					<div class="authors">
						{book.authors.map((a) => a.name).join(', ') || '—'}
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
		margin-bottom: 1.5rem;
	}
	input[type='search'] {
		flex: 1;
		padding: 0.5rem 0.75rem;
		font-size: 1rem;
		border: 1px solid #ccc;
		border-radius: 4px;
		background: #fff;
	}
	.count {
		color: #666;
		font-size: 0.875rem;
	}
	.error {
		color: #b00020;
	}
	.empty {
		color: #666;
		line-height: 1.5;
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
	.title {
		font-weight: 600;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.authors {
		color: #666;
		font-size: 0.875rem;
	}
</style>
