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
</script>

<h2>Duplicate candidates</h2>
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
					</li>
				{/each}
			</ul>
		</section>
	{/each}
{/if}

<style>
	h2 {
		margin: 0 0 0.5rem;
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
</style>
