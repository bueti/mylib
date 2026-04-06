<script lang="ts">
	import { onMount, onDestroy } from 'svelte';

	let { bookId }: { bookId: number } = $props();
	let isMobile = $state(false);

	onMount(() => {
		document.body.style.overflow = 'hidden';
		isMobile = window.innerWidth < 768 || 'ontouchstart' in window;
	});

	onDestroy(() => {
		document.body.style.overflow = '';
	});
</script>

{#if isMobile}
	<!-- Mobile browsers can't render PDFs inside iframes. Open the
	     PDF directly in the browser's native viewer instead. -->
	<div class="mobile-fallback">
		<p>Mobile browsers can't display PDFs inline.</p>
		<a
			href="/api/books/{bookId}/file?inline=1"
			class="open-btn"
			target="_blank"
			rel="noopener"
		>
			Open PDF in viewer
		</a>
		<p class="hint">This opens the PDF in your browser's built-in viewer with full scrolling and zoom.</p>
	</div>
{:else}
	<iframe src="/api/books/{bookId}/file?inline=1" title="PDF viewer"></iframe>
{/if}

<style>
	iframe {
		display: block;
		flex: 1 1 auto;
		width: 100%;
		min-height: 0;
		border: 0;
		background: #555;
		touch-action: auto;
	}
	.mobile-fallback {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 1rem;
		padding: 2rem;
		text-align: center;
		color: #666;
	}
	.open-btn {
		display: inline-block;
		padding: 0.75rem 1.5rem;
		background: #0366d6;
		color: #fff;
		text-decoration: none;
		border-radius: 6px;
		font-size: 1rem;
		font-weight: 500;
	}
	.open-btn:hover {
		background: #0256b9;
	}
	.hint {
		font-size: 0.8125rem;
		color: #999;
		max-width: 300px;
	}
</style>
