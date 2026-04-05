<script lang="ts">
	import { onMount } from 'svelte';
	import {
		listCollections,
		createCollection,
		deleteCollection,
		type Collection
	} from '$lib/api/collections';

	let collections = $state<Collection[]>([]);
	let loading = $state(true);
	let newName = $state('');
	let creating = $state(false);
	let error = $state<string | null>(null);

	onMount(async () => {
		await load();
	});

	async function load() {
		loading = true;
		try {
			collections = await listCollections();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load';
		} finally {
			loading = false;
		}
	}

	async function onCreate(e: Event) {
		e.preventDefault();
		if (!newName.trim() || creating) return;
		creating = true;
		error = null;
		try {
			await createCollection(newName.trim());
			newName = '';
			await load();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to create';
		} finally {
			creating = false;
		}
	}

	async function onDelete(c: Collection) {
		if (!confirm(`Delete collection "${c.name}"? Books are not removed.`)) return;
		try {
			await deleteCollection(c.id);
			await load();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete';
		}
	}
</script>

<h2>Collections</h2>

<form onsubmit={onCreate} class="new">
	<input
		type="text"
		bind:value={newName}
		placeholder="New collection name"
		maxlength="100"
	/>
	<button type="submit" disabled={creating || !newName.trim()}>Create</button>
</form>

{#if error}
	<p class="error">{error}</p>
{/if}

{#if loading}
	<p>Loading…</p>
{:else if collections.length === 0}
	<p class="empty">No collections yet.</p>
{:else}
	<ul class="list">
		{#each collections as c (c.id)}
			<li>
				<a href="/collections/{c.id}">
					<span class="name">{c.name}</span>
					<span class="count">{c.book_count} {c.book_count === 1 ? 'book' : 'books'}</span>
				</a>
				<button class="del" onclick={() => onDelete(c)} aria-label="Delete {c.name}">✕</button>
			</li>
		{/each}
	</ul>
{/if}

<style>
	h2 {
		margin: 0 0 1rem;
	}
	.new {
		display: flex;
		gap: 0.5rem;
		margin-bottom: 1.5rem;
	}
	.new input {
		flex: 1;
		padding: 0.5rem 0.625rem;
		border: 1px solid #ccc;
		border-radius: 4px;
	}
	.new button {
		padding: 0.5rem 1rem;
		background: #0366d6;
		color: #fff;
		border: 0;
		border-radius: 4px;
		cursor: pointer;
	}
	.new button:disabled {
		background: #999;
		cursor: not-allowed;
	}
	.list {
		list-style: none;
		padding: 0;
		margin: 0;
	}
	.list li {
		display: flex;
		align-items: center;
		gap: 1rem;
		padding: 0.75rem 1rem;
		border: 1px solid #e5e5e5;
		border-radius: 4px;
		margin-bottom: 0.5rem;
		background: #fff;
	}
	.list a {
		flex: 1;
		text-decoration: none;
		color: inherit;
		display: flex;
		align-items: baseline;
		gap: 1rem;
	}
	.name {
		font-weight: 600;
	}
	.count {
		color: #666;
		font-size: 0.875rem;
	}
	.del {
		background: transparent;
		border: 0;
		color: #b00020;
		cursor: pointer;
		font-size: 1rem;
		padding: 0.25rem 0.5rem;
	}
	.del:hover {
		background: #fee;
		border-radius: 3px;
	}
	.empty,
	.error {
		color: #666;
	}
	.error {
		color: #b00020;
	}
</style>
