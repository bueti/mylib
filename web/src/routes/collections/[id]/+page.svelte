<script lang="ts">
	import { page } from '$app/state';
	import { client, type Book } from '$lib/api/client';

	interface Collection {
		id: number;
		name: string;
		book_count: number;
	}

	let collection = $state<Collection | null>(null);
	let books = $state<Book[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	$effect(() => {
		const id = Number(page.params.id);
		if (!Number.isFinite(id) || id <= 0) {
			error = 'Invalid collection id';
			loading = false;
			return;
		}
		load(id);
	});

	async function load(id: number) {
		loading = true;
		error = null;
		try {
			const [collRes, booksRes] = await Promise.all([
				fetch(`/api/collections/${id}`, { credentials: 'same-origin' }),
				client.GET('/api/books', {
					params: { query: { collection_id: id, limit: 500, sort: 'title' } }
				})
			]);
			if (!collRes.ok) throw new Error('Collection not found');
			collection = (await collRes.json()) as Collection;
			if (booksRes.error) throw new Error(booksRes.error.detail ?? 'Failed to load books');
			books = booksRes.data?.books ?? [];
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load';
		} finally {
			loading = false;
		}
	}
</script>

<a href="/collections" class="back">← All collections</a>

{#if loading}
	<p>Loading…</p>
{:else if error}
	<p class="error">{error}</p>
{:else if collection}
	<h2>{collection.name}</h2>
	<p class="count">{books.length} {books.length === 1 ? 'book' : 'books'}</p>

	{#if books.length === 0}
		<p class="empty">No books in this collection yet. Open a book and click "Add to collection".</p>
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
{/if}

<style>
	.back {
		display: inline-block;
		margin-bottom: 1rem;
		color: #666;
		text-decoration: none;
		font-size: 0.875rem;
	}
	.back:hover {
		color: #222;
	}
	h2 {
		margin: 0 0 0.25rem;
	}
	.count {
		color: #666;
		font-size: 0.875rem;
		margin: 0 0 1.5rem;
	}
	.error {
		color: #b00020;
	}
	.empty {
		color: #666;
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
