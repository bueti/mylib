<script lang="ts">
	import { onMount, onDestroy } from 'svelte';

	// PDFs are rendered by the browser's built-in viewer via an iframe.
	// This works in Chrome/Edge/Safari/Firefox with no JS dependency —
	// pdf.js is only worth pulling in if we want a custom toolbar.
	let { bookId }: { bookId: number } = $props();

	onMount(() => {
		// Lock body scroll so the page doesn't move behind the
		// fixed reader overlay on mobile.
		document.body.style.overflow = 'hidden';
	});

	onDestroy(() => {
		document.body.style.overflow = '';
	});
</script>

<iframe src="/api/books/{bookId}/file?inline=1" title="PDF viewer"></iframe>

<style>
	iframe {
		display: block;
		flex: 1 1 auto;
		width: 100%;
		min-height: 0;
		border: 0;
		background: #555;
		/* Allow the iframe's internal content to handle touch scrolling */
		touch-action: auto;
	}
</style>
