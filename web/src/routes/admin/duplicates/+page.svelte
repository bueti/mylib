<script lang="ts">
	import { onMount } from 'svelte';

	interface DupBook {
		id: number;
		title: string;
		path: string;
		format: string;
		size_bytes: number;
		isbn?: string;
		has_cover: boolean;
	}
	interface Group {
		reason: 'isbn' | 'title';
		key: string;
		books: DupBook[];
	}

	let groups = $state<Group[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let actionRunning = $state('');
	let actionResult = $state<string | null>(null);
	let deleting = $state<number | null>(null);

	async function rescanMetadata() {
		actionRunning = 'rescan';
		actionResult = null;
		try {
			const res = await fetch('/api/admin/rescan-metadata', {
				method: 'POST',
				credentials: 'same-origin'
			});
			if (!res.ok) throw new Error(await res.text());
			const data = await res.json();
			actionResult = `Rescan complete: ${data.updated} books updated`;
		} catch (e) {
			actionResult = e instanceof Error ? e.message : 'Failed';
		} finally {
			actionRunning = '';
		}
	}

	async function enrichAll() {
		actionRunning = 'enrich';
		actionResult = null;
		try {
			const res = await fetch('/api/admin/enrich-all', {
				method: 'POST',
				credentials: 'same-origin'
			});
			if (!res.ok) throw new Error(await res.text());
			const data = await res.json();
			actionResult = `Enrichment complete: ${data.enriched} books updated`;
		} catch (e) {
			actionResult = e instanceof Error ? e.message : 'Failed';
		} finally {
			actionRunning = '';
		}
	}

	onMount(async () => {
		try {
			const res = await fetch('/api/admin/duplicates', { credentials: 'same-origin' });
			if (res.status === 403) throw new Error('Admin only');
			if (!res.ok) throw new Error('HTTP ' + res.status);
			const data = await res.json();
			groups = (data?.groups ?? []) as Group[];
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load';
		} finally {
			loading = false;
		}
	});

	function formatSize(bytes: number): string {
		return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
	}

	async function deleteBook(bookId: number, deleteFile: boolean) {
		const action = deleteFile ? 'Delete book and file from disk' : 'Remove from library (keep file)';
		if (!confirm(`${action}? This cannot be undone.`)) return;
		deleting = bookId;
		try {
			const q = deleteFile ? '?delete_file=1' : '';
			const res = await fetch(`/api/books/${bookId}${q}`, {
				method: 'DELETE',
				credentials: 'same-origin'
			});
			if (!res.ok) throw new Error(await res.text());
			// Remove from the local groups so the UI updates immediately.
			for (const g of groups) {
				g.books = g.books.filter((b) => b.id !== bookId);
			}
			groups = groups.filter((g) => g.books.length > 1);
		} catch (e) {
			alert(e instanceof Error ? e.message : 'Delete failed');
		} finally {
			deleting = null;
		}
	}
</script>

<h2>Admin</h2>

<div class="admin-actions">
	<button onclick={rescanMetadata} disabled={!!actionRunning}>
		{actionRunning === 'rescan' ? 'Rescanning…' : 'Rescan embedded metadata'}
	</button>
	<button onclick={enrichAll} disabled={!!actionRunning}>
		{actionRunning === 'enrich' ? 'Enriching…' : 'Enrich all from Open Library'}
	</button>
	{#if actionResult}
		<p class="result">{actionResult}</p>
	{/if}
</div>

<h3>Duplicate candidates</h3>
<p class="note">Books grouped by shared ISBN or matching title + author. Resolve by deleting the unwanted file on disk, then clicking Rescan on the home page.</p>

{#if loading}
	<p>Loading…</p>
{:else if error}
	<p class="error">{error}</p>
{:else if groups.length === 0}
	<p class="empty">No duplicates detected. Nice.</p>
{:else}
	{#each groups as g}
		<section class="group">
			<header>
				<span class="reason">{g.reason === 'isbn' ? 'Shared ISBN' : 'Same title + author'}</span>
				<code>{g.key}</code>
			</header>
			<ul>
				{#each g.books as b (b.id)}
					<li>
						{#if b.has_cover}
							<img src="/api/books/{b.id}/cover" alt="" />
						{:else}
							<div class="placeholder">{b.title.charAt(0)}</div>
						{/if}
						<div class="info">
							<a href="/books/{b.id}" class="title">{b.title}</a>
							<div class="meta">
								{b.format.toUpperCase()} · {formatSize(b.size_bytes)}{#if b.isbn} · ISBN {b.isbn}{/if}
							</div>
							<div class="path">{b.path}</div>
						</div>
						<div class="actions">
							<button
								class="btn-remove"
								disabled={deleting === b.id}
								onclick={() => deleteBook(b.id, false)}
								title="Remove from library but keep the file on disk"
							>Remove</button>
							<button
								class="btn-delete"
								disabled={deleting === b.id}
								onclick={() => deleteBook(b.id, true)}
								title="Remove from library AND delete the file from disk"
							>Delete file</button>
						</div>
					</li>
				{/each}
			</ul>
		</section>
	{/each}
{/if}

<style>
	h2 {
		margin: 0 0 1rem;
	}
	h3 {
		margin: 2rem 0 0.5rem;
	}
	.admin-actions {
		display: flex;
		gap: 0.75rem;
		align-items: center;
		flex-wrap: wrap;
		margin-bottom: 1rem;
	}
	.admin-actions button {
		padding: 0.5rem 1rem;
		background: #0366d6;
		color: #fff;
		border: 0;
		border-radius: 4px;
		font-size: 0.875rem;
		cursor: pointer;
	}
	.admin-actions button:hover:not(:disabled) {
		background: #0256b9;
	}
	.admin-actions button:disabled {
		background: #888;
		cursor: wait;
	}
	.result {
		font-size: 0.875rem;
		color: #0366d6;
		margin: 0;
	}
	.note {
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
	.group {
		margin-bottom: 2rem;
		border: 1px solid #e0e0e0;
		border-radius: 4px;
		overflow: hidden;
	}
	.group header {
		display: flex;
		align-items: baseline;
		gap: 0.75rem;
		padding: 0.5rem 1rem;
		background: #f5f5f5;
		border-bottom: 1px solid #e0e0e0;
	}
	.reason {
		font-weight: 600;
		font-size: 0.8125rem;
		color: #444;
	}
	code {
		font-size: 0.75rem;
		color: #666;
	}
	ul {
		list-style: none;
		padding: 0;
		margin: 0;
	}
	li {
		display: flex;
		gap: 1rem;
		padding: 0.75rem 1rem;
		border-bottom: 1px solid #f0f0f0;
	}
	li:last-child {
		border-bottom: 0;
	}
	img,
	.placeholder {
		width: 50px;
		height: 75px;
		object-fit: cover;
		border-radius: 2px;
		background: #eee;
		display: flex;
		align-items: center;
		justify-content: center;
		font-weight: 700;
		color: #aaa;
	}
	.info {
		flex: 1;
		min-width: 0;
	}
	.title {
		font-weight: 600;
		color: inherit;
		text-decoration: none;
	}
	.title:hover {
		color: #0366d6;
	}
	.meta {
		font-size: 0.75rem;
		color: #666;
	}
	.path {
		font-family: ui-monospace, monospace;
		font-size: 0.75rem;
		color: #888;
		word-break: break-all;
	}
	.actions {
		display: flex;
		flex-direction: column;
		gap: 0.375rem;
		flex-shrink: 0;
	}
	.btn-remove,
	.btn-delete {
		padding: 0.25rem 0.625rem;
		border-radius: 3px;
		font-size: 0.75rem;
		cursor: pointer;
		white-space: nowrap;
	}
	.btn-remove {
		background: #f5f5f5;
		border: 1px solid #ccc;
		color: #333;
	}
	.btn-remove:hover:not(:disabled) {
		background: #eee;
		border-color: #999;
	}
	.btn-delete {
		background: #b00020;
		border: 0;
		color: #fff;
	}
	.btn-delete:hover:not(:disabled) {
		background: #900018;
	}
	.btn-remove:disabled,
	.btn-delete:disabled {
		opacity: 0.5;
		cursor: wait;
	}
</style>
