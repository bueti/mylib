

export const index = 5;
let component_cache;
export const component = async () => component_cache ??= (await import('../entries/pages/books/_id_/read/_page.svelte.js')).default;
export const universal = {
  "prerender": false,
  "ssr": false
};
export const universal_id = "src/routes/books/[id]/read/+page.ts";
export const imports = ["_app/immutable/nodes/5.CeM0HZOd.js","_app/immutable/chunks/Bq_VSsjf.js","_app/immutable/chunks/B6MMa1Tp.js","_app/immutable/chunks/t1H3BwMP.js","_app/immutable/chunks/Dmkd57ne.js","_app/immutable/chunks/DO78dgk1.js","_app/immutable/chunks/B7YJYiUB.js","_app/immutable/chunks/DGOPKHdQ.js","_app/immutable/chunks/I4q_SWcC.js","_app/immutable/chunks/XyWiDX-_.js","_app/immutable/chunks/OvZ2lAsZ.js"];
export const stylesheets = ["_app/immutable/assets/5.fYcE-O4t.css"];
export const fonts = [];
