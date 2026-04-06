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

<!--
  The reader is a fixed overlay that covers the viewport below the site
  header. This escapes the root layout's max-width + padding without
  touching its styles, so other pages are unaffected.
-->
<div class="reader-overlay">
	{#if loading}
		<p class="status">Loading…</p>
	{:else if error}
		<p class="status error">Error: {error}</p>
	{:else if book}
		<div class="reader-header">
			<a href="/books/{book.id}" class="back">← Back to details</a>
			<span class="title">{book.title}</span>
		</div>
		<div class="reader-body">
			{#if book.format === 'pdf'}
				<PdfReader bookId={book.id} />
			{:else if book.format === 'epub'}
				<EpubReader bookId={book.id} title={book.title} />
			{:else}
				<p class="status">
					Format "{book.format}" can't be read in-browser yet. <a
						href="/api/books/{book.id}/file"
						data-sveltekit-reload
						download>Download</a
					> instead.
				</p>
			{/if}
		</div>
	{/if}
</div>

<style>
	.reader-overlay {
		position: fixed;
		top: 58px; /* sits below the site header */
		left: 0;
		right: 0;
		bottom: 0;
		display: flex;
		flex-direction: column;
		background: #fafafa;
		z-index: 10;
	}
	.reader-header {
		flex: 0 0 auto;
		display: flex;
		align-items: center;
		gap: 1rem;
		padding: 0.5rem 1.5rem;
		border-bottom: 1px solid #e0e0e0;
		background: #fff;
	}
	.reader-body {
		flex: 1 1 auto;
		min-height: 0;
		display: flex;
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
	@media (max-width: 640px) {
		.reader-overlay {
			top: 48px; /* shorter header on mobile */
		}
		.reader-header {
			padding: 0.375rem 0.75rem;
			gap: 0.5rem;
		}
		.back {
			font-size: 0.75rem;
		}
		.title {
			font-size: 0.875rem;
		}
	}
	.status {
		margin: 2rem;
		color: #666;
	}
	.status.error {
		color: #b00020;
	}
</style>
