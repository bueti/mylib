// Lightweight fetch helpers for collections. We don't regenerate the
// openapi client every time the Go side changes shape; typed wrappers
// kept here.

export interface Collection {
	id: number;
	name: string;
	book_count: number;
	created_at: string;
}

export async function listCollections(): Promise<Collection[]> {
	const res = await fetch('/api/collections', { credentials: 'same-origin' });
	if (!res.ok) throw new Error('HTTP ' + res.status);
	const data = await res.json();
	return (data?.collections ?? []) as Collection[];
}

export async function createCollection(name: string): Promise<Collection> {
	const res = await fetch('/api/collections', {
		method: 'POST',
		credentials: 'same-origin',
		headers: { 'content-type': 'application/json' },
		body: JSON.stringify({ name })
	});
	if (!res.ok) throw new Error(await res.text());
	return (await res.json()) as Collection;
}

export async function deleteCollection(id: number): Promise<void> {
	const res = await fetch(`/api/collections/${id}`, {
		method: 'DELETE',
		credentials: 'same-origin'
	});
	if (!res.ok && res.status !== 204) throw new Error('HTTP ' + res.status);
}

export async function addBookToCollection(collectionId: number, bookId: number): Promise<void> {
	const res = await fetch(`/api/collections/${collectionId}/books/${bookId}`, {
		method: 'POST',
		credentials: 'same-origin'
	});
	if (!res.ok && res.status !== 204) throw new Error('HTTP ' + res.status);
}

export async function removeBookFromCollection(
	collectionId: number,
	bookId: number
): Promise<void> {
	const res = await fetch(`/api/collections/${collectionId}/books/${bookId}`, {
		method: 'DELETE',
		credentials: 'same-origin'
	});
	if (!res.ok && res.status !== 204) throw new Error('HTTP ' + res.status);
}
