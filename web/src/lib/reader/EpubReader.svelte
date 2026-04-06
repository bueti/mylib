<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import ePub from 'epubjs';
	import { loadProgress, makeSaver } from './progress';
	import { themes, fontSizes, type ThemeName, type FontSize } from './themes';

	let { bookId, title }: { bookId: number; title: string } = $props();

	let container: HTMLDivElement | undefined = $state();
	let loading = $state(true);
	let error = $state<string | null>(null);
	let atStart = $state(true);
	let atEnd = $state(false);
	let pageInfo = $state('');
	let tocOpen = $state(false);
	let currentTheme = $state<ThemeName>('light');
	let currentFontSize = $state<FontSize>('M');

	interface TocItem {
		id: string;
		href: string;
		label: string;
		subitems?: TocItem[];
	}
	let toc = $state<TocItem[]>([]);

	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	let book: any;
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	let rendition: any;

	let resizeObserver: ResizeObserver | undefined;
	let saver: ReturnType<typeof makeSaver> | undefined;

	onMount(async () => {
		// Lock body scroll while reader is open so mobile doesn't
		// scroll the page behind the fixed overlay.
		document.body.style.overflow = 'hidden';

		const bid = bookId;
		saver = makeSaver(bid);
		try {
			const res = await fetch(`/api/books/${bid}/file?inline=1`);
			if (!res.ok) throw new Error('HTTP ' + res.status);
			const buf = await res.arrayBuffer();

			book = ePub(buf);
			await book.ready;
			toc = book.navigation?.toc ?? [];

			rendition = book.renderTo(container, {
				width: '100%',
				height: '100%',
				flow: 'paginated',
				spread: 'none',
				allowScriptedContent: true
			});


			// Base CSS overrides to prevent book styles from breaking
			// column pagination. Applied once via default().
			rendition.themes.default({
				'img': { 'max-width': '100% !important', 'height': 'auto !important' },
				'svg': { 'max-width': '100% !important' },
				'table': { 'max-width': '100% !important' },
				'pre': { 'white-space': 'pre-wrap !important', 'word-wrap': 'break-word !important' },
				'body': { 'margin': '0 !important', 'padding': '1rem !important', 'max-width': 'none !important' },
			});

			book.locations.generate(1600).catch(() => {});

			// Prefer server-side progress/prefs; fall back to localStorage.
			const server = await loadProgress(bid);
			if (server?.theme && server.theme in themes) currentTheme = server.theme as ThemeName;
			if (server?.font_size && server.font_size in fontSizes) {
				currentFontSize = server.font_size as FontSize;
			}
			applyTheme(currentTheme);
			rendition.themes.fontSize(fontSizes[currentFontSize]);

			const localKey = `mylib.read.${bid}.cfi`;
			const local = localStorage.getItem(localKey);
			const startCfi = server?.position ?? local ?? undefined;
			await rendition.display(startCfi);

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
					saver?.save({
						position: loc.start.cfi,
						percent: loc.start.percentage ?? 0,
						theme: currentTheme,
						font_size: currentFontSize
					});
				}
			);

			loading = false;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to open book';
			loading = false;
		}
	});

	onDestroy(() => {
		document.body.style.overflow = '';
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

	// Swipe navigation for touch devices.
	let touchStartX = 0;
	let touchStartY = 0;
	function onTouchStart(e: TouchEvent) {
		touchStartX = e.touches[0].clientX;
		touchStartY = e.touches[0].clientY;
	}
	function onTouchEnd(e: TouchEvent) {
		const dx = e.changedTouches[0].clientX - touchStartX;
		const dy = e.changedTouches[0].clientY - touchStartY;
		// Only trigger on horizontal swipes (|dx| > 50px, |dy| < |dx|).
		if (Math.abs(dx) > 50 && Math.abs(dy) < Math.abs(dx)) {
			if (dx < 0) next();
			else prev();
		}
	}

	function next() {
		rendition?.next();
	}
	function prev() {
		rendition?.prev();
	}
	function gotoHref(href: string) {
		if (!rendition || !book) return;
		tocOpen = false;
		// Resolve the TOC href through the spine so epub.js finds the
		// correct section. TOC hrefs may contain fragments (#id) that
		// need to be handled by the section, not the display call.
		const section = book.spine.get(href);
		if (section) {
			rendition.display(section.href);
		} else {
			// Fallback: try the raw href.
			rendition.display(href);
		}
	}
	function applyTheme(name: ThemeName) {
		if (!rendition) return;
		const t = themes[name];
		// themes.override() directly sets CSS properties on each
		// chapter iframe — reliable even when switching back to a
		// previously used theme (unlike select() which leaves stale
		// rules in the stylesheet).
		rendition.themes.override('color', t.body.color);
		rendition.themes.override('background', t.body.background);
	}

	function setTheme(name: ThemeName) {
		currentTheme = name;
		applyTheme(name);
	}
	function setFontSize(size: FontSize) {
		currentFontSize = size;
		rendition?.themes.fontSize(fontSizes[size]);
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
	<aside class="toc" class:open={tocOpen}>
		<header>
			<span>Contents</span>
			<button class="close" onclick={() => (tocOpen = false)} aria-label="Close table of contents">×</button>
		</header>
		<ul>
			{#each toc as item (item.id)}
				<li>
					<button onclick={() => gotoHref(item.href)}>{item.label.trim()}</button>
					{#if item.subitems && item.subitems.length > 0}
						<ul>
							{#each item.subitems as sub (sub.id)}
								<li>
									<button onclick={() => gotoHref(sub.href)}>{sub.label.trim()}</button>
								</li>
							{/each}
						</ul>
					{/if}
				</li>
			{/each}
		</ul>
	</aside>

	<div class="main">
		<div class="toolbar">
			<button onclick={() => (tocOpen = !tocOpen)} title="Contents">☰</button>
			<div class="font-size" role="group" aria-label="Font size">
				{#each ['S', 'M', 'L'] as size}
					<button
						class:active={currentFontSize === size}
						onclick={() => setFontSize(size as FontSize)}
					>{size}</button>
				{/each}
			</div>
			<div class="theme" role="group" aria-label="Theme">
				{#each ['light', 'sepia', 'dark'] as name}
					<button
						class:active={currentTheme === name}
						class="theme-swatch theme-{name}"
						onclick={() => setTheme(name as ThemeName)}
						aria-label={name}
						title={name}
					></button>
				{/each}
			</div>
		</div>

		{#if loading}
			<p class="status">Opening {title}…</p>
		{/if}
		{#if error}
			<p class="status error">Error: {error}</p>
		{/if}

		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div class="viewport-wrap" ontouchstart={onTouchStart} ontouchend={onTouchEnd}>
			<button class="nav nav-prev" onclick={prev} disabled={atStart} aria-label="Previous page">‹</button>
			<div class="viewport" bind:this={container}></div>
			<button class="nav nav-next" onclick={next} disabled={atEnd} aria-label="Next page">›</button>
		</div>

		{#if pageInfo && !loading}
			<div class="page-info">{pageInfo}</div>
		{/if}
	</div>
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
	.toc {
		flex: 0 0 0;
		width: 0;
		overflow: hidden;
		background: #fff;
		border-right: 1px solid #e0e0e0;
		transition: flex-basis 0.15s, width 0.15s;
	}
	.toc.open {
		flex: 0 0 280px;
		width: 280px;
		overflow-y: auto;
	}
	.toc header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 0.75rem 1rem;
		border-bottom: 1px solid #eee;
		font-weight: 600;
		font-size: 0.875rem;
	}
	.close {
		background: none;
		border: 0;
		font-size: 1.25rem;
		cursor: pointer;
		color: #666;
	}
	.toc ul {
		list-style: none;
		padding: 0.25rem 0;
		margin: 0;
	}
	.toc li ul {
		padding-left: 1rem;
	}
	.toc button {
		display: block;
		width: 100%;
		padding: 0.375rem 1rem;
		background: none;
		border: 0;
		text-align: left;
		font-size: 0.8125rem;
		color: #333;
		cursor: pointer;
	}
	.toc button:hover {
		background: #f0f0f0;
	}
	.main {
		flex: 1 1 auto;
		display: flex;
		flex-direction: column;
		min-width: 0;
	}
	.toolbar {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		padding: 0.375rem 1rem;
		border-bottom: 1px solid #e0e0e0;
		background: #fff;
	}
	.toolbar > button {
		background: transparent;
		border: 0;
		font-size: 1.125rem;
		color: #333;
		cursor: pointer;
		padding: 0.25rem 0.5rem;
		border-radius: 3px;
	}
	.toolbar > button:hover {
		background: #f0f0f0;
	}
	.font-size,
	.theme {
		display: flex;
		gap: 0.25rem;
	}
	.font-size button {
		background: #f5f5f5;
		border: 1px solid #ddd;
		border-radius: 3px;
		padding: 0.125rem 0.5rem;
		font-size: 0.75rem;
		cursor: pointer;
	}
	.font-size button.active {
		background: #222;
		color: #fff;
		border-color: #222;
	}
	.theme-swatch {
		width: 22px;
		height: 22px;
		border-radius: 50%;
		border: 1px solid #ccc;
		cursor: pointer;
	}
	.theme-swatch.active {
		outline: 2px solid #0366d6;
		outline-offset: 2px;
	}
	.theme-light {
		background: #fff;
	}
	.theme-sepia {
		background: #f4ecd8;
	}
	.theme-dark {
		background: #1a1a1a;
	}
	.viewport-wrap {
		position: relative;
		flex: 1 1 auto;
		display: flex;
		min-height: 0;
	}
	.viewport {
		flex: 1;
		min-width: 0;
		background: #fff;
		box-shadow: 0 0 20px rgba(0, 0, 0, 0.08);
		margin: 0 1rem;
		overflow: hidden;
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

	/* Mobile: hide nav arrows (swipe instead), compact toolbar,
	   full-width viewport, collapse TOC by default */
	@media (max-width: 640px) {
		.nav {
			display: none;
		}
		.viewport {
			margin: 0;
			box-shadow: none;
		}
		.toolbar {
			padding: 0.25rem 0.5rem;
			gap: 0.375rem;
			overflow-x: auto;
		}
		.toolbar > button {
			padding: 0.25rem 0.375rem;
			font-size: 1rem;
		}
		.font-size button {
			padding: 0.125rem 0.375rem;
		}
		.toc.open {
			position: absolute;
			top: 0;
			left: 0;
			bottom: 0;
			z-index: 10;
			flex: none;
			width: 260px;
			box-shadow: 4px 0 12px rgba(0, 0, 0, 0.15);
		}
		.page-info {
			bottom: 0.25rem;
			font-size: 0.625rem;
		}
	}
</style>
