<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { session, whoami, logout } from '$lib/api/session.svelte';
	import { migrateLocalProgress } from '$lib/api/migrate-local';

	let { children } = $props();

	onMount(async () => {
		await whoami();
		// Redirect to /login if we're on a protected route and not signed in.
		if (!session.user && !isPublicRoute(page.url.pathname)) {
			await goto('/login');
			return;
		}
		if (session.user) {
			// Best-effort one-shot migration of v0.1 localStorage progress.
			void migrateLocalProgress();
		}
	});

	$effect(() => {
		// Whenever navigation happens, re-check auth state against the route.
		const path = page.url.pathname;
		if (session.loaded && !session.user && !isPublicRoute(path)) {
			goto('/login');
		}
	});

	function isPublicRoute(path: string): boolean {
		return path === '/login';
	}
</script>

<header>
	<div class="brand">
		<a href="/"><h1>mylib</h1></a>
	</div>
	{#if session.user}
		<nav>
			<a href="/">Books</a>
			<a href="/collections">Collections</a>
			{#if session.can('admin', 'access')}
				<a href="/admin/duplicates">Admin</a>
			{/if}
		</nav>
		<div class="user">
			<span>{session.user.username}{session.user.role === 'admin' ? ' · admin' : ''}</span>
			<button onclick={logout}>Sign out</button>
		</div>
	{/if}
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
		display: flex;
		align-items: center;
		justify-content: space-between;
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
	nav {
		display: flex;
		gap: 1.5rem;
		margin-left: 2rem;
		flex: 1;
	}
	nav a {
		color: #bbb;
		font-size: 0.875rem;
	}
	nav a:hover {
		color: #fff;
	}
	.user {
		display: flex;
		align-items: center;
		gap: 1rem;
		font-size: 0.8125rem;
		color: #bbb;
	}
	.user button {
		background: transparent;
		color: #fff;
		border: 1px solid #555;
		border-radius: 3px;
		padding: 0.25rem 0.625rem;
		font-size: 0.8125rem;
		cursor: pointer;
	}
	.user button:hover {
		background: #333;
		border-color: #777;
	}
	main {
		padding: 2rem;
		max-width: 1200px;
		margin: 0 auto;
	}
</style>
