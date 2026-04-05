<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import ePub from 'epubjs';

	let { bookId, title }: { bookId: number; title: string } = $props();

	let container: HTMLDivElement | undefined = $state();
	let loading = $state(true);
	let error = $state<string | null>(null);
	let atStart = $state(true);
	let atEnd = $state(false);
	let location = $state('');

	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	let book: any;
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	let rendition: any;

	onMount(async () => {
		// Capture the initial bookId (props are reactive, but we only
		// run onMount once — this component is remounted per book).
		const id = bookId;
		const storageKey = `mylib.read.${id}.cfi`;
		try {
			const res = await fetch(`/api/books/${id}/file?inline=1`);
			if (!res.ok) throw new Error('HTTP ' + res.status);
			const buf = await res.arrayBuffer();

			book = ePub(buf);
			rendition = book.renderTo(container, {
				width: '100%',
				height: '100%',
				flow: 'paginated',
				spread: 'auto'
			});

			const saved = localStorage.getItem(storageKey);
			await rendition.display(saved ?? undefined);

			// Track location for the progress bar and persist it.
			rendition.on('relocated', (loc: { start: { cfi: string; percentage: number }; atStart?: boolean; atEnd?: boolean }) => {
				location = loc.start.cfi;
				atStart = !!loc.atStart;
				atEnd = !!loc.atEnd;
				try {
					localStorage.setItem(storageKey, loc.start.cfi);
				} catch {
					// quota exceeded / private mode — ignore
				}
			});

			loading = false;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to open book';
			loading = false;
		}
	});

	onDestroy(() => {
		try {
			rendition?.destroy();
			book?.destroy();
		} catch {
			// ignore shutdown errors
		}
	});

	function next() {
		rendition?.next();
	}
	function prev() {
		rendition?.prev();
	}

	function onKey(e: KeyboardEvent) {
		if (e.key === 'ArrowRight' || e.key === 'PageDown' || e.key === ' ') {
			e.preventDefault();
			next();
		} else if (e.key === 'ArrowLeft' || e.key === 'PageUp') {
			e.preventDefault();
			prev();
		}
	}
</script>

<svelte:window onkeydown={onKey} />

<div class="reader">
	{#if loading}
		<p class="status">Opening {title}…</p>
	{/if}
	{#if error}
		<p class="status error">Error: {error}</p>
	{/if}

	<button class="nav nav-prev" onclick={prev} disabled={atStart} aria-label="Previous page">‹</button>
	<div class="viewport" bind:this={container}></div>
	<button class="nav nav-next" onclick={next} disabled={atEnd} aria-label="Next page">›</button>
</div>

<style>
	.reader {
		position: relative;
		display: flex;
		align-items: stretch;
		height: calc(100vh - 60px);
		background: #fafafa;
	}
	.viewport {
		flex: 1;
		background: #fff;
		box-shadow: 0 0 20px rgba(0, 0, 0, 0.08);
		margin: 0 1rem;
	}
	.nav {
		flex: 0 0 60px;
		border: 0;
		background: transparent;
		font-size: 2.5rem;
		color: #888;
		cursor: pointer;
		transition: color 0.1s;
	}
	.nav:hover:not(:disabled) {
		color: #222;
	}
	.nav:disabled {
		color: #ddd;
		cursor: default;
	}
	.status {
		position: absolute;
		inset: 0;
		display: flex;
		align-items: center;
		justify-content: center;
		color: #666;
		pointer-events: none;
	}
	.status.error {
		color: #b00020;
	}
</style>
