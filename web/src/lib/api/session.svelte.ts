// Session state for the SPA. Uses a Svelte 5 rune so pages can
// reactively read the current user and their permissions.
import { goto } from '$app/navigation';

export interface User {
	id: number;
	username: string;
	role: 'admin' | 'reader';
	created_at: string;
}

// Shared session state — a single $state backing store accessed via
// getters so consumers react to updates.
let _user = $state<User | null>(null);
let _loaded = $state(false);
let _permissions = $state<Set<string>>(new Set());

export const session = {
	get user() {
		return _user;
	},
	get loaded() {
		return _loaded;
	},
	get isAdmin() {
		return _user?.role === 'admin';
	},
	/** Check if the current user has a specific permission, e.g. can('books', 'delete') */
	can(resource: string, action: string): boolean {
		return _permissions.has(`${resource}:${action}`);
	}
};

async function loadPermissions(): Promise<void> {
	try {
		const res = await fetch('/api/auth/permissions', { credentials: 'same-origin' });
		if (!res.ok) return;
		const data = await res.json();
		_permissions = new Set((data?.permissions ?? []) as string[]);
	} catch {
		_permissions = new Set();
	}
}

export async function whoami(): Promise<User | null> {
	try {
		const res = await fetch('/api/auth/me', { credentials: 'same-origin' });
		if (res.status === 401) {
			_user = null;
			_loaded = true;
			return null;
		}
		if (!res.ok) throw new Error('HTTP ' + res.status);
		_user = (await res.json()) as User;
		_loaded = true;
		await loadPermissions();
		return _user;
	} catch {
		_user = null;
		_loaded = true;
		return null;
	}
}

export async function login(username: string, password: string): Promise<User> {
	const res = await fetch('/api/auth/login', {
		method: 'POST',
		headers: { 'content-type': 'application/json' },
		credentials: 'same-origin',
		body: JSON.stringify({ username, password })
	});
	if (!res.ok) {
		const text = await res.text();
		throw new Error(text || 'Login failed');
	}
	_user = (await res.json()) as User;
	_loaded = true;
	await loadPermissions();
	return _user;
}

export async function logout(): Promise<void> {
	await fetch('/api/auth/logout', {
		method: 'POST',
		credentials: 'same-origin'
	});
	_user = null;
	_permissions = new Set();
	await goto('/login');
}
