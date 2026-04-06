

export const index = 7;
let component_cache;
export const component = async () => component_cache ??= (await import('../entries/pages/collections/_id_/_page.svelte.js')).default;
export const universal = {
  "prerender": false,
  "ssr": false
};
export const universal_id = "src/routes/collections/[id]/+page.ts";
export const imports = ["_app/immutable/nodes/7.DikFEpBK.js","_app/immutable/chunks/Bq_VSsjf.js","_app/immutable/chunks/B6MMa1Tp.js","_app/immutable/chunks/t1H3BwMP.js","_app/immutable/chunks/I4q_SWcC.js","_app/immutable/chunks/Dmkd57ne.js","_app/immutable/chunks/DO78dgk1.js","_app/immutable/chunks/B7YJYiUB.js","_app/immutable/chunks/DGOPKHdQ.js"];
export const stylesheets = ["_app/immutable/assets/7.D6A7x9Wt.css"];
export const fonts = [];
