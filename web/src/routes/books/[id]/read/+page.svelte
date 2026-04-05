<script lang="ts">
	import { page } from '$app/state';
	import { client, type Book } from '$lib/api/client';
	import PdfReader from '$lib/reader/PdfReader.svelte';
	import EpubReader from '$lib/reader/EpubReader.svelte';

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
		} finally {
			loading = false;
		}
	}
</script>

{#if loading}
	<p>Loading…</p>
{:else if error}
	<p class="error">Error: {error}</p>
{:else if book}
	<div class="reader-header">
		<a href="/books/{book.id}" class="back">← Back to details</a>
		<span class="title">{book.title}</span>
	</div>
	{#if book.format === 'pdf'}
		<PdfReader bookId={book.id} />
	{:else if book.format === 'epub'}
		<EpubReader bookId={book.id} title={book.title} />
	{:else}
		<p class="error">
			Format "{book.format}" can't be read in-browser yet. <a
				href="/api/books/{book.id}/file"
				data-sveltekit-reload
				download>Download</a
			> instead.
		</p>
	{/if}
{/if}

<style>
	.reader-header {
		display: flex;
		align-items: center;
		gap: 1rem;
		padding: 0.5rem 0 0.75rem;
		border-bottom: 1px solid #e0e0e0;
		margin: -2rem -2rem 0.5rem;
		padding-left: 2rem;
		padding-right: 2rem;
	}
	.back {
		color: #666;
		text-decoration: none;
		font-size: 0.875rem;
	}
	.back:hover {
		color: #222;
	}
	.title {
		font-weight: 600;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.error {
		color: #b00020;
	}
</style>
