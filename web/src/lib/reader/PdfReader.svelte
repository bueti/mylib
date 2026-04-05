<script lang="ts">
	// PDFs are rendered by the browser's built-in viewer via an iframe.
	// This works in Chrome/Edge/Safari/Firefox with no JS dependency —
	// pdf.js is only worth pulling in if we want a custom toolbar.
	let { bookId }: { bookId: number } = $props();
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
	}
</style>
