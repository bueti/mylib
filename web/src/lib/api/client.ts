import createClient from 'openapi-fetch';
import type { components, paths } from './schema';

// Base URL is empty so relative /api paths go to the server that served
// the SPA (either Vite dev proxy or the embedded Go binary).
export const client = createClient<paths>({ baseUrl: '' });

// Re-export commonly used schema types so pages don't need to reach
// into the generated file directly.
export type Book = components['schemas']['BookDTO'];
export type Author = components['schemas']['AuthorDTO'];
export type Series = components['schemas']['SeriesDTO'];
export type ScanJob = components['schemas']['ScanJobDTO'];
