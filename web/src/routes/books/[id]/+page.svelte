<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { client, type Book } from '$lib/api/client';
	import { session } from '$lib/api/session.svelte';
	import {
		listCollections,
		createCollection,
		addBookToCollection,
		type Collection
	} from '$lib/api/collections';

	let book = $state<Book | null>(null);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let collections = $state<Collection[]>([]);
	let collectionsMenuOpen = $state(false);
	let addingTo = $state<number | null>(null);
	let addStatus = $state<string | null>(null);
	let enriching = $state(false);
	let enrichStatus = $state<string | null>(null);
	let deleting = $state(false);

	// Edit mode state.
	let editing = $state(false);
	let editForm = $state({
		title: '',
		subtitle: '',
		description: '',
		authors: '',
		series_name: '',
		series_index: '',
		tags: '',
		language: '',
		isbn: '',
		publisher: ''
	});
	let saving = $state(false);
	let saveError = $state<string | null>(null);

	function startEditing() {
		if (!book) return;
		editForm = {
			title: book.title,
			subtitle: book.subtitle ?? '',
			description: book.description ?? '',
			authors: (book.authors ?? []).map((a) => a.name).join(', '),
			series_name: book.series?.name ?? '',
			series_index: book.series_index != null ? String(book.series_index) : '',
			tags: (book.tags ?? []).join(', '),
			language: book.language ?? '',
			isbn: book.isbn ?? '',
			publisher: book.publisher ?? ''
		};
		editing = true;
		saveError = null;
	}

	function cancelEditing() {
		editing = false;
		saveError = null;
	}

	async function saveEdits() {
		if (!book || saving) return;
		saving = true;
		saveError = null;
		try {
			const authors = editForm.authors
				.split(',')
				.map((s) => s.trim())
				.filter(Boolean);
			const tags = editForm.tags
				.split(',')
				.map((s) => s.trim())
				.filter(Boolean);
			const body: Record<string, unknown> = {
				title: editForm.title,
				subtitle: editForm.subtitle,
				description: editForm.description,
				authors,
				series_name: editForm.series_name,
				tags,
				language: editForm.language,
				isbn: editForm.isbn,
				publisher: editForm.publisher
			};
			if (editForm.series_index) {
				body.series_index = parseFloat(editForm.series_index);
			}
			const res = await fetch(`/api/books/${book.id}`, {
				method: 'PATCH',
				credentials: 'same-origin',
				headers: { 'content-type': 'application/json' },
				body: JSON.stringify(body)
			});
			if (!res.ok) throw new Error(await res.text());
			editing = false;
			load(book.id);
		} catch (e) {
			saveError = e instanceof Error ? e.message : 'Save failed';
		} finally {
			saving = false;
		}
	}

	$effect(() => {
		const id = Number(page.params.id);
		if (!Number.isFinite(id) || id <= 0) {
			error = 'Invalid book id';
			loading = false;
			return;
		}
		load(id);
		void loadCollections();
	});

	async function loadCollections() {
		try {
			collections = await listCollections();
		} catch {
			// non-fatal — menu stays empty
		}
	}

	async function addTo(c: Collection) {
		if (!book || addingTo === c.id) return;
		addingTo = c.id;
		addStatus = null;
		try {
			await addBookToCollection(c.id, book.id);
			addStatus = `Added to "${c.name}"`;
			collectionsMenuOpen = false;
			await loadCollections();
		} catch (e) {
			addStatus = e instanceof Error ? e.message : 'Failed to add';
		} finally {
			addingTo = null;
			setTimeout(() => (addStatus = null), 3000);
		}
	}

	async function removeBook() {
		if (!book || deleting) return;
		if (!confirm(`Remove "${book.title}" from library?\n\nThe file on disk will be kept.`)) return;
		deleting = true;
		try {
			const res = await fetch(`/api/books/${book.id}`, {
				method: 'DELETE',
				credentials: 'same-origin'
			});
			if (!res.ok) throw new Error(await res.text());
			await goto('/');
		} catch (e) {
			alert(e instanceof Error ? e.message : 'Remove failed');
		} finally {
			deleting = false;
		}
	}

	async function deleteBookAndFile() {
		if (!book || deleting) return;
		if (!confirm(`Delete "${book.title}" AND its file from disk?\n\nThis cannot be undone.`)) return;
		deleting = true;
		try {
			const res = await fetch(`/api/books/${book.id}?delete_file=1`, {
				method: 'DELETE',
				credentials: 'same-origin'
			});
			if (!res.ok) throw new Error(await res.text());
			await goto('/');
		} catch (e) {
			alert(e instanceof Error ? e.message : 'Delete failed');
		} finally {
			deleting = false;
		}
	}

	async function refreshMetadata() {
		if (!book || enriching) return;
		enriching = true;
		enrichStatus = null;
		try {
			const res = await fetch(`/api/books/${book.id}/enrich`, {
				method: 'POST',
				credentials: 'same-origin'
			});
			if (!res.ok) throw new Error(await res.text());
			enrichStatus = 'Metadata refreshed';
			// Reload book to show updated fields.
			const id = book.id;
			load(id);
		} catch (e) {
			enrichStatus = e instanceof Error ? e.message : 'Failed to refresh';
		} finally {
			enriching = false;
			setTimeout(() => (enrichStatus = null), 4000);
		}
	}

	async function newCollectionAndAdd() {
		const name = prompt('New collection name');
		if (!name || !book) return;
		try {
			const c = await createCollection(name.trim());
			await addBookToCollection(c.id, book.id);
			addStatus = `Added to "${c.name}"`;
			collectionsMenuOpen = false;
			await loadCollections();
		} catch (e) {
			addStatus = e instanceof Error ? e.message : 'Failed';
		} finally {
			setTimeout(() => (addStatus = null), 3000);
		}
	}

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
			{#if book.format === 'epub' || book.format === 'pdf'}
				<a class="read-btn" href="/books/{book.id}/read">
					Read in browser
				</a>
			{/if}
			<a
				class="download-btn"
				href="/api/books/{book.id}/file"
				data-sveltekit-reload
				download
			>
				Download {book.format.toUpperCase()} · {formatSize(book.size_bytes)}
			</a>

			<div class="collections-menu">
				<button class="collections-toggle" onclick={() => (collectionsMenuOpen = !collectionsMenuOpen)}>
					Add to collection ▾
				</button>
				{#if collectionsMenuOpen}
					<div class="menu">
						{#each collections as c (c.id)}
							<button onclick={() => addTo(c)} disabled={addingTo === c.id}>{c.name}</button>
						{/each}
						{#if collections.length > 0}<hr />{/if}
						<button onclick={newCollectionAndAdd}>+ New collection…</button>
					</div>
				{/if}
			</div>
			{#if addStatus}
				<p class="add-status">{addStatus}</p>
			{/if}

			<button class="refresh-btn" onclick={refreshMetadata} disabled={enriching}>
				{enriching ? 'Refreshing…' : 'Refresh metadata'}
			</button>
			{#if enrichStatus}
				<p class="add-status">{enrichStatus}</p>
			{/if}

			{#if session.can('books', 'delete')}
				<button class="remove-btn" onclick={removeBook} disabled={deleting}>
					Remove from library
				</button>
				<button class="delete-btn" onclick={deleteBookAndFile} disabled={deleting}>
					Delete file from disk
				</button>
			{/if}
		</div>

		<div class="meta-col">
			{#if editing}
				<form class="edit-form" onsubmit={(e) => { e.preventDefault(); saveEdits(); }}>
					<label>Title <input type="text" bind:value={editForm.title} required /></label>
					<label>Subtitle <input type="text" bind:value={editForm.subtitle} /></label>
					<label>Authors <input type="text" bind:value={editForm.authors} placeholder="Comma-separated" /></label>
					<label>Series <input type="text" bind:value={editForm.series_name} /></label>
					<label>Series # <input type="text" bind:value={editForm.series_index} placeholder="e.g. 1" /></label>
					<label>Tags <input type="text" bind:value={editForm.tags} placeholder="Comma-separated" /></label>
					<label>Language <input type="text" bind:value={editForm.language} /></label>
					<label>ISBN <input type="text" bind:value={editForm.isbn} /></label>
					<label>Publisher <input type="text" bind:value={editForm.publisher} /></label>
					<label>Description <textarea bind:value={editForm.description} rows="5"></textarea></label>
					{#if saveError}
						<p class="save-error">{saveError}</p>
					{/if}
					<div class="edit-actions">
						<button type="submit" class="save-btn" disabled={saving}>
							{saving ? 'Saving…' : 'Save'}
						</button>
						<button type="button" class="cancel-btn" onclick={cancelEditing}>Cancel</button>
					</div>
				</form>
			{:else}
				<div class="meta-header">
					<h2>{book.title}</h2>
					<button class="edit-btn" onclick={startEditing}>Edit</button>
				</div>
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
	.read-btn,
	.download-btn {
		display: inline-block;
		padding: 0.625rem 1rem;
		text-align: center;
		text-decoration: none;
		border-radius: 4px;
		font-size: 0.875rem;
		font-weight: 500;
	}
	.read-btn {
		background: #0366d6;
		color: #fff;
	}
	.read-btn:hover {
		background: #0256b9;
	}
	.download-btn {
		background: #222;
		color: #fff;
	}
	.download-btn:hover {
		background: #000;
	}
	.collections-menu {
		position: relative;
	}
	.collections-toggle {
		width: 100%;
		padding: 0.5rem 0.75rem;
		background: #fff;
		color: #333;
		border: 1px solid #ccc;
		border-radius: 4px;
		font-size: 0.8125rem;
		cursor: pointer;
		text-align: left;
	}
	.collections-toggle:hover {
		border-color: #888;
	}
	.menu {
		position: absolute;
		top: 100%;
		left: 0;
		right: 0;
		margin-top: 0.25rem;
		background: #fff;
		border: 1px solid #ccc;
		border-radius: 4px;
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
		padding: 0.25rem 0;
		z-index: 20;
		max-height: 240px;
		overflow-y: auto;
	}
	.menu button {
		display: block;
		width: 100%;
		padding: 0.4rem 0.75rem;
		background: none;
		border: 0;
		font-size: 0.8125rem;
		text-align: left;
		cursor: pointer;
		color: #333;
	}
	.menu button:hover {
		background: #f0f0f0;
	}
	.menu hr {
		margin: 0.25rem 0;
		border: 0;
		border-top: 1px solid #eee;
	}
	.add-status {
		margin: 0.25rem 0 0;
		font-size: 0.75rem;
		color: #0366d6;
	}
	.refresh-btn {
		width: 100%;
		padding: 0.5rem 0.75rem;
		background: #f5f5f5;
		color: #333;
		border: 1px solid #ccc;
		border-radius: 4px;
		font-size: 0.8125rem;
		cursor: pointer;
	}
	.refresh-btn:hover:not(:disabled) {
		border-color: #888;
		background: #eee;
	}
	.refresh-btn:disabled {
		color: #999;
		cursor: wait;
	}
	.remove-btn {
		width: 100%;
		padding: 0.5rem 0.75rem;
		background: #f5f5f5;
		color: #333;
		border: 1px solid #ccc;
		border-radius: 4px;
		font-size: 0.8125rem;
		cursor: pointer;
	}
	.remove-btn:hover:not(:disabled) {
		background: #eee;
		border-color: #999;
	}
	.delete-btn {
		width: 100%;
		padding: 0.5rem 0.75rem;
		background: #fff;
		color: #b00020;
		border: 1px solid #b00020;
		border-radius: 4px;
		font-size: 0.8125rem;
		cursor: pointer;
	}
	.delete-btn:hover:not(:disabled) {
		background: #b00020;
		color: #fff;
	}
	.remove-btn:disabled,
	.delete-btn:disabled {
		opacity: 0.5;
		cursor: wait;
	}
	.meta-header {
		display: flex;
		align-items: baseline;
		gap: 1rem;
	}
	.meta-col h2 {
		margin: 0 0 0.25rem;
		font-size: 1.75rem;
		line-height: 1.2;
	}
	.edit-btn {
		background: none;
		border: 1px solid #ccc;
		border-radius: 3px;
		padding: 0.25rem 0.625rem;
		font-size: 0.75rem;
		color: #555;
		cursor: pointer;
		white-space: nowrap;
	}
	.edit-btn:hover {
		border-color: #888;
		color: #222;
	}
	.edit-form {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}
	.edit-form label {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
		font-size: 0.8125rem;
		font-weight: 500;
		color: #555;
	}
	.edit-form input,
	.edit-form textarea {
		padding: 0.4rem 0.5rem;
		border: 1px solid #ccc;
		border-radius: 4px;
		font-size: 0.9375rem;
		font-family: inherit;
	}
	.edit-form textarea {
		resize: vertical;
	}
	.edit-actions {
		display: flex;
		gap: 0.5rem;
	}
	.save-btn {
		padding: 0.5rem 1.25rem;
		background: #0366d6;
		color: #fff;
		border: 0;
		border-radius: 4px;
		font-size: 0.875rem;
		cursor: pointer;
	}
	.save-btn:disabled {
		background: #888;
		cursor: wait;
	}
	.cancel-btn {
		padding: 0.5rem 1.25rem;
		background: #f5f5f5;
		border: 1px solid #ccc;
		border-radius: 4px;
		font-size: 0.875rem;
		cursor: pointer;
	}
	.save-error {
		color: #b00020;
		font-size: 0.8125rem;
		margin: 0;
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
