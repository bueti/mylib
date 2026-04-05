// Placeholder until `pnpm gen:api` is run against a live server.
// The real file is auto-generated from /api/openapi.json.
export interface paths {
	'/books': {
		get: {
			parameters: {
				query?: {
					q?: string;
					author_id?: number;
					series_id?: number;
					tag?: string;
					format?: string;
					sort?: 'title' | '-title' | 'added' | '-added';
					limit?: number;
					offset?: number;
				};
			};
			responses: {
				200: {
					content: {
						'application/json': {
							books: Book[];
							total: number;
							limit: number;
							offset: number;
						};
					};
				};
			};
		};
	};
	'/books/{id}': {
		get: {
			parameters: { path: { id: number } };
			responses: { 200: { content: { 'application/json': Book } } };
		};
	};
}

export interface Book {
	id: number;
	title: string;
	sort_title: string;
	subtitle?: string;
	description?: string;
	authors: { id: number; name: string; sort_name: string }[];
	series?: { id: number; name: string };
	series_index?: number;
	language?: string;
	isbn?: string;
	publisher?: string;
	published_at?: string;
	format: string;
	size_bytes: number;
	tags?: string[];
	has_cover: boolean;
	added_at: string;
}
