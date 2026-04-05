<script lang="ts">
	import { page } from '$app/state';
	import { client, type Book } from '$lib/api/client';

	let book = $state<Book | null>(null);
	let loading = $state(true);
	let error = $state<string | null>(null);

	$effect(() => {
		const id = Number(page.params.id);
		if (!Number.isFinite(id) || id <= 0) {
			error = 'Invalid book id';
			loading = false;
			return;
		}
		load(id);
	});

	async function load(id: number) {
		loading = true;
		error = null;
		try {
			const { data, error: err, response } = await client.GET('/api/books/{id}', {
				params: { path: { id } }
			});
			if (err) throw new Error(err.detail || response.statusText);
			book = data ?? null;
			if (!book) throw new Error('Empty response');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load';
			book = null;
		} finally {
			loading = false;
		}
	}

	function formatSize(bytes: number): string {
		const mb = bytes / (1024 * 1024);
		if (mb < 0.1) return (bytes / 1024).toFixed(1) + ' KB';
		return mb.toFixed(1) + ' MB';
	}

	function formatDate(iso: string): string {
		if (!iso) return '';
		// Published dates come back in several shapes (ISO, "1968", RFC3339);
		// trim to a year if that's all there is.
		const d = new Date(iso);
		if (Number.isFinite(d.getTime())) {
			return d.toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' });
		}
		return iso;
	}
</script>

<a href="/" class="back">← All books</a>

{#if loading}
	<p>Loading…</p>
{:else if error}
	<p class="error">Error: {error}</p>
{:else if book}
	<article class="detail">
		<div class="cover-col">
			{#if book.has_cover}
				<img src="/api/books/{book.id}/cover" alt="Cover of {book.title}" />
			{:else}
				<div class="placeholder">{book.title.charAt(0)}</div>
			{/if}
			<a
				class="download-btn"
				href="/api/books/{book.id}/file"
				data-sveltekit-reload
				download
			>
				Download {book.format.toUpperCase()} · {formatSize(book.size_bytes)}
			</a>
		</div>

		<div class="meta-col">
			<h2>{book.title}</h2>
			{#if book.subtitle}
				<p class="subtitle">{book.subtitle}</p>
			{/if}

			<dl>
				{#if (book.authors ?? []).length > 0}
					<dt>Authors</dt>
					<dd>
						{#each book.authors ?? [] as a, i}
							{#if i > 0}, {/if}
							<a href="/?author_id={a.id}&author_name={encodeURIComponent(a.name)}">{a.name}</a>
						{/each}
					</dd>
				{/if}
				{#if book.series}
					<dt>Series</dt>
					<dd>
						<a href="/?series_id={book.series.id}&series_name={encodeURIComponent(book.series.name)}">{book.series.name}</a>
						{#if book.series_index != null}
							· #{book.series_index}
						{/if}
					</dd>
				{/if}
				{#if book.publisher || book.published_at}
					<dt>Published</dt>
					<dd>
						{book.publisher ?? ''}{book.publisher && book.published_at ? ' · ' : ''}{formatDate(book.published_at ?? '')}
					</dd>
				{/if}
				{#if book.language}
					<dt>Language</dt>
					<dd>{book.language}</dd>
				{/if}
				{#if book.isbn}
					<dt>ISBN</dt>
					<dd>{book.isbn}</dd>
				{/if}
				{#if (book.tags ?? []).length > 0}
					<dt>Tags</dt>
					<dd>
						{#each book.tags ?? [] as tag}
							<a class="chip" href="/?tag={encodeURIComponent(tag)}">{tag}</a>
						{/each}
					</dd>
				{/if}
			</dl>

			{#if book.description}
				<div class="description">
					{@html book.description}
				</div>
			{/if}
		</div>
	</article>
{/if}

<style>
	.back {
		display: inline-block;
		margin-bottom: 1.5rem;
		color: #666;
		text-decoration: none;
	}
	.back:hover {
		color: #222;
	}
	.detail {
		display: grid;
		grid-template-columns: 240px 1fr;
		gap: 2.5rem;
		align-items: start;
	}
	.cover-col {
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}
	.cover-col img {
		width: 100%;
		aspect-ratio: 2 / 3;
		object-fit: cover;
		border-radius: 4px;
		box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
	}
	.placeholder {
		width: 100%;
		aspect-ratio: 2 / 3;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 4rem;
		font-weight: 700;
		color: #aaa;
		background: #f0f0f0;
		border-radius: 4px;
	}
	.download-btn {
		display: inline-block;
		padding: 0.625rem 1rem;
		background: #222;
		color: #fff;
		text-align: center;
		text-decoration: none;
		border-radius: 4px;
		font-size: 0.875rem;
		font-weight: 500;
	}
	.download-btn:hover {
		background: #000;
	}
	.meta-col h2 {
		margin: 0 0 0.25rem;
		font-size: 1.75rem;
		line-height: 1.2;
	}
	.subtitle {
		color: #666;
		font-style: italic;
		margin: 0 0 1.25rem;
	}
	dl {
		display: grid;
		grid-template-columns: max-content 1fr;
		column-gap: 1.5rem;
		row-gap: 0.5rem;
		margin: 1.5rem 0;
		font-size: 0.9375rem;
	}
	dt {
		color: #888;
		font-weight: 500;
	}
	dd {
		margin: 0;
	}
	dd a {
		color: #0366d6;
		text-decoration: none;
	}
	dd a:hover {
		text-decoration: underline;
	}
	.chip {
		display: inline-block;
		padding: 0.125rem 0.5rem;
		margin-right: 0.375rem;
		background: #eef;
		border-radius: 12px;
		font-size: 0.8125rem;
	}
	.description {
		margin-top: 1.5rem;
		line-height: 1.6;
		color: #333;
	}
	.error {
		color: #b00020;
	}
	@media (max-width: 640px) {
		.detail {
			grid-template-columns: 1fr;
		}
	}
</style>
