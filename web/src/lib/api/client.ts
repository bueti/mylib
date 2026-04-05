import createClient from 'openapi-fetch';
import type { paths } from './schema';

// Base URL is empty so relative /api paths go to the server that served
// the SPA (either Vite dev proxy or the embedded Go binary).
export const client = createClient<paths>({ baseUrl: '' });
