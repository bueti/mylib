

export const index = 4;
let component_cache;
export const component = async () => component_cache ??= (await import('../entries/pages/books/_id_/_page.svelte.js')).default;
export const universal = {
  "prerender": false,
  "ssr": false
};
export const universal_id = "src/routes/books/[id]/+page.ts";
export const imports = ["_app/immutable/nodes/4.CVLtB7n_.js","_app/immutable/chunks/Bq_VSsjf.js","_app/immutable/chunks/B6MMa1Tp.js","_app/immutable/chunks/t1H3BwMP.js","_app/immutable/chunks/I4q_SWcC.js","_app/immutable/chunks/Dmkd57ne.js","_app/immutable/chunks/54iCTxcV.js","_app/immutable/chunks/DO78dgk1.js","_app/immutable/chunks/B7YJYiUB.js","_app/immutable/chunks/DGOPKHdQ.js","_app/immutable/chunks/DOtZQTI5.js","_app/immutable/chunks/D3CgP4Dm.js"];
export const stylesheets = ["_app/immutable/assets/4.BPog0_1L.css"];
export const fonts = [];
