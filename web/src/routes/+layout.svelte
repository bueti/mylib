<script lang="ts">
	let { children } = $props();
</script>

<header>
	<a href="/"><h1>mylib</h1></a>
</header>

<main>
	{@render children()}
</main>

<style>
	:global(body) {
		font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
		margin: 0;
		background: #fafafa;
		color: #222;
	}
	header {
		padding: 1rem 2rem;
		background: #222;
		color: #fff;
	}
	header a {
		color: inherit;
		text-decoration: none;
	}
	h1 {
		margin: 0;
		font-size: 1.25rem;
		letter-spacing: 0.02em;
	}
	main {
		padding: 2rem;
		max-width: 1200px;
		margin: 0 auto;
	}
</style>
