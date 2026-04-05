<script lang="ts">
	import { goto } from '$app/navigation';
	import { login } from '$lib/api/session.svelte';

	let username = $state('');
	let password = $state('');
	let error = $state<string | null>(null);
	let submitting = $state(false);

	async function onSubmit(e: Event) {
		e.preventDefault();
		if (submitting) return;
		submitting = true;
		error = null;
		try {
			await login(username, password);
			await goto('/');
		} catch (err) {
			error = err instanceof Error ? err.message : 'Login failed';
		} finally {
			submitting = false;
		}
	}
</script>

<div class="wrap">
	<h2>Sign in</h2>
	<form onsubmit={onSubmit}>
		<label>
			Username
			<!-- svelte-ignore a11y_autofocus -->
			<input type="text" bind:value={username} autocomplete="username" required autofocus />
		</label>
		<label>
			Password
			<input type="password" bind:value={password} autocomplete="current-password" required />
		</label>
		{#if error}
			<p class="error">{error}</p>
		{/if}
		<button type="submit" disabled={submitting}>
			{submitting ? 'Signing in…' : 'Sign in'}
		</button>
	</form>
</div>

<style>
	.wrap {
		max-width: 360px;
		margin: 4rem auto;
	}
	h2 {
		margin: 0 0 1.5rem;
	}
	form {
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}
	label {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
		font-size: 0.875rem;
		color: #444;
	}
	input {
		padding: 0.5rem 0.625rem;
		font-size: 1rem;
		border: 1px solid #ccc;
		border-radius: 4px;
	}
	button {
		padding: 0.625rem 1rem;
		background: #0366d6;
		color: #fff;
		border: 0;
		border-radius: 4px;
		font-size: 0.9375rem;
		font-weight: 500;
		cursor: pointer;
	}
	button:hover:not(:disabled) {
		background: #0256b9;
	}
	button:disabled {
		background: #888;
		cursor: wait;
	}
	.error {
		color: #b00020;
		margin: 0;
		font-size: 0.875rem;
	}
</style>
