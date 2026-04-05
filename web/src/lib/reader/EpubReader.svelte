<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import ePub from 'epubjs';
	import { loadProgress, makeSaver } from './progress';

	let { bookId, title }: { bookId: number; title: string } = $props();

	let container: HTMLDivElement | undefined = $state();
	let loading = $state(true);
	let error = $state<string | null>(null);
	let atStart = $state(true);
	let atEnd = $state(false);
	let pageInfo = $state('');

	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	let book: any;
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	let rendition: any;

	let resizeObserver: ResizeObserver | undefined;

	let saver: ReturnType<typeof makeSaver> | undefined;

	onMount(async () => {
		// Props are reactive but this component is remounted per book,
		// so capturing bookId once is fine.
		const bid = bookId;
		saver = makeSaver(bid);
		try {
			const res = await fetch(`/api/books/${bid}/file?inline=1`);
			if (!res.ok) throw new Error('HTTP ' + res.status);
			const buf = await res.arrayBuffer();

			book = ePub(buf);
			// Wait for the book (manifest, spine) before rendering;
			// otherwise pagination can stall after the first chapter.
			await book.ready;

			rendition = book.renderTo(container, {
				width: '100%',
				height: '100%',
				// scrolled-doc renders each chapter as a scrollable
				// document. This sidesteps epub.js's column-based
				// pagination which breaks on books with wide images,
				// fixed-width CSS, or non-trivial layouts.
				flow: 'scrolled-doc',
				manager: 'default',
				allowScriptedContent: true
			});

			// Generate locations so percentage and atEnd work reliably.
			book.locations.generate(1600).catch(() => {
				// non-fatal — pagination still works without precise locations
			});

			// Prefer server-side progress; fall back to any legacy
			// localStorage CFI from v0.1 so users don't lose their place.
			const server = await loadProgress(bid);
			const localKey = `mylib.read.${bid}.cfi`;
			const local = localStorage.getItem(localKey);
			const startCfi = server?.position ?? local ?? undefined;
			await rendition.display(startCfi);

			// Re-paginate whenever the viewport resizes. Without this
			// the first layout can be wrong (before flex settles) and
			// chapter content ends up clipped.
			if (container) {
				resizeObserver = new ResizeObserver(() => {
					if (!container) return;
					try {
						rendition.resize(container.clientWidth, container.clientHeight);
					} catch {
						// ignore transient layout errors
					}
				});
				resizeObserver.observe(container);
			}

			rendition.on(
				'relocated',
				(loc: {
					start: { cfi: string; percentage: number; displayed: { page: number; total: number } };
					atStart?: boolean;
					atEnd?: boolean;
				}) => {
					atStart = !!loc.atStart;
					atEnd = !!loc.atEnd;
					const pct = loc.start.percentage ? Math.round(loc.start.percentage * 100) : 0;
					pageInfo = `${loc.start.displayed.page}/${loc.start.displayed.total} · ${pct}%`;
					saver?.save({ position: loc.start.cfi, percent: loc.start.percentage ?? 0 });
				}
			);

			loading = false;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to open book';
			loading = false;
		}
	});

	onDestroy(() => {
		try {
			saver?.flush();
			resizeObserver?.disconnect();
			rendition?.destroy();
			book?.destroy();
		} catch {
			// ignore shutdown errors
		}
	});

	function onBeforeUnload() {
		saver?.flush();
	}

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

<svelte:window onkeydown={onKey} onbeforeunload={onBeforeUnload} />

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

	{#if pageInfo && !loading}
		<div class="page-info">{pageInfo}</div>
	{/if}
</div>

<style>
	.reader {
		position: relative;
		flex: 1 1 auto;
		display: flex;
		align-items: stretch;
		min-width: 0;
		min-height: 0;
		background: #fafafa;
	}
	.viewport {
		flex: 1;
		min-width: 0;
		background: #fff;
		box-shadow: 0 0 20px rgba(0, 0, 0, 0.08);
		margin: 0 1rem;
		overflow-y: auto;
	}
	.nav {
		flex: 0 0 60px;
		border: 0;
		background: transparent;
		font-size: 2.5rem;
		color: #888;
		cursor: pointer;
		transition: color 0.1s;
		align-self: center;
		height: 100%;
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
	.page-info {
		position: absolute;
		bottom: 0.5rem;
		left: 50%;
		transform: translateX(-50%);
		font-size: 0.75rem;
		color: #888;
		background: rgba(255, 255, 255, 0.85);
		padding: 0.125rem 0.5rem;
		border-radius: 10px;
	}
</style>
