

export const index = 6;
let component_cache;
export const component = async () => component_cache ??= (await import('../entries/pages/collections/_page.svelte.js')).default;
export const universal = {
  "prerender": false,
  "ssr": false
};
export const universal_id = "src/routes/collections/+page.ts";
export const imports = ["_app/immutable/nodes/6.DPpDho-6.js","_app/immutable/chunks/Bq_VSsjf.js","_app/immutable/chunks/B6MMa1Tp.js","_app/immutable/chunks/t1H3BwMP.js","_app/immutable/chunks/I4q_SWcC.js","_app/immutable/chunks/Dmkd57ne.js","_app/immutable/chunks/54iCTxcV.js","_app/immutable/chunks/D3CgP4Dm.js"];
export const stylesheets = ["_app/immutable/assets/6.3_xAAWEN.css"];
export const fonts = [];
